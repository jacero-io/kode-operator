/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Configuration constants
const (
	ContainerRestartPolicyAlways corev1.ContainerRestartPolicy = "Always"
	PersistentVolumeClaimName                                  = "kode-pvc"
	LoopRetryTime                                              = 10 * time.Second
)

type KodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("Reconcile")
	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	log.Info("Fetching Kode instance", "Namespace", req.Namespace, "Name", req.Name)
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to fetch Kode instance", "Namespace", req.Namespace, "Name", req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logKodeManifest(log, kode)

	// Validate references
	if err := r.validateReferences(kode); err != nil {
		log.Error(err, "Invalid references in Kode", "Namespace", kode.Namespace, "Name", kode.Name)
		return ctrl.Result{}, err
	}

	// Fetch the KodeTemplate instance referenced by TemplateRef
	kodeTemplate := &kodev1alpha1.KodeTemplate{}
	if kode.Spec.TemplateRef.Name != "" {
		kodeTemplateName := client.ObjectKey{Name: kode.Spec.TemplateRef.Name}
		log.Info("Fetching KodeTemplate instance", "Name", kode.Spec.TemplateRef.Name)
		if err := r.Get(ctx, kodeTemplateName, kodeTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("KodeTemplate instance not found, requeuing", "Name", kode.Spec.TemplateRef.Name)
				return ctrl.Result{RequeueAfter: LoopRetryTime}, nil // Retry after some time
			}
			log.Error(err, "Failed to fetch KodeTemplate instance", "Name", kode.Spec.TemplateRef.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	logKodeTemplateManifest(log, kodeTemplate)

	// Fetch the EnvoyProxyTemplate instance referenced by EnvoyProxyTemplateReference
	envoyProxyTemplate := &kodev1alpha1.EnvoyProxyTemplate{}
	envoyProxyTemplateName := client.ObjectKey{Name: kodeTemplate.Spec.EnvoyProxyTemplateRef.Name}
	if kodeTemplate.Spec.EnvoyProxyTemplateRef.Name != "" {
		log.Info("Fetching EnvoyProxyTemplate instance", "Name", envoyProxyTemplateName)
		if err := r.Get(ctx, envoyProxyTemplateName, envoyProxyTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("EnvoyProxyTemplate instance not found, requeuing", "Name", envoyProxyTemplateName)
				return ctrl.Result{RequeueAfter: LoopRetryTime}, nil // Retry after some time
			}
			log.Error(err, "Failed to fetch EnvoyProxyTemplate instance", "Name", envoyProxyTemplateName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	logEnvoyProxyTemplateManifest(log, envoyProxyTemplate)

	// Ensure the Deployment and Service exist
	if err := r.ensureDeployment(ctx, kode, kodeTemplate, envoyProxyTemplate); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureService(ctx, kode, kodeTemplate); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure PVC exists
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		if _, err := r.ensurePVC(ctx, kode); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) validateReferences(kode *kodev1alpha1.Kode) error {
	if kode.Spec.TemplateRef.Kind != "KodeTemplate" {
		return fmt.Errorf("invalid reference kind for TemplateRef: %s", kode.Spec.TemplateRef.Kind)
	}
	return nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("Kode")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
