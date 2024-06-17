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
	client "sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/controller"
)

// Configuration constants
const (
	ContainerRestartPolicyAlways corev1.ContainerRestartPolicy = "Always"
	PersistentVolumeClaimName                                  = "kode-pvc"
	LoopRetryTime                                              = 1 * time.Second
	OperatorName                                               = "kode-operator"
)

type KodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodeclustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyclusterconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// TODO: When requeued (EnvoyProxyConfig, EnvoyProxyClusterConfig) or (KodeTemplate, KodeClusterTemplate), force reconcile of Kode resource

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("Reconcile")

	labels := map[string]string{}

	// Fetch the Kode resource
	kode := &kodev1alpha1.Kode{}
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to fetch Kode resource", "Namespace", req.Namespace, "Name", req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Fetching Kode resource", "Namespace", req.Namespace, "Name", req.Name)
	logKodeManifest(log, kode)

	// Validate references
	if err := r.validateReferences(kode); err != nil {
		log.Error(err, "Invalid references in Kode", "Namespace", kode.Namespace, "Name", kode.Name)
		return ctrl.Result{}, err
	}

	var sharedKodeTemplateSpec kodev1alpha1.SharedKodeTemplateSpec
	var sharedEnvoyProxyConfigSpec kodev1alpha1.SharedEnvoyProxyConfigSpec

	// Fetch the KodeTemplate or KodeClusterTemplate resource and EnvoyProxyConfig resource
	if kode.Spec.TemplateRef.Name != "" {
		labels["app.kubernetes.io/name"] = "kode-" + kode.Name
		labels["app.kubernetes.io/managed-by"] = OperatorName
		labels["kode.jacero.io/name"] = kode.Name

		if kode.Spec.TemplateRef.Kind == "KodeTemplate" {
			kodeTemplate := &kodev1alpha1.KodeTemplate{}
			kodeTemplateName := kode.Spec.TemplateRef.Name
			kodeTemplateNamespace := kode.Spec.TemplateRef.Namespace

			// If no namespace is provided, use the namespace of the Kode resource
			if kodeTemplateNamespace == "" {
				kodeTemplateNamespace = kode.Namespace
			}
			kodeTemplateNameObject := client.ObjectKey{Name: kodeTemplateName, Namespace: kodeTemplateNamespace}
			log.Info("Fetching KodeTemplate resource", "Namespace", kodeTemplateNamespace, "Name", kodeTemplateName)

			// Attempt to fetch the KodeTemplate resource
			if err := r.Get(ctx, kodeTemplateNameObject, kodeTemplate); err != nil {
				if errors.IsNotFound(err) {
					log.Info("KodeTemplate resource not found in namespace, requeuing", "Namespace", kodeTemplateNamespace, "Name", kodeTemplateName)
					return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
				}
				log.Error(err, "Failed to fetch KodeTemplate resource", "Namespace", kodeTemplateNamespace, "Name", kode.Spec.TemplateRef.Name)
				return ctrl.Result{Requeue: true}, err
			}
			log.Info("KodeTemplate resource found", "Namespace", kodeTemplateNamespace, "Name", kodeTemplate.Name)
			labels["kode-template.kode.jacero.io/name"] = kodeTemplate.Name
			sharedKodeTemplateSpec = kodeTemplate.Spec.SharedKodeTemplateSpec

			// Attempt to fetch the EnvoyProxyConfig resource
			if kodeTemplate.Spec.EnvoyProxyRef.Name != "" {
				envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{}
				envoyProxyConfigName := kodeTemplate.Spec.EnvoyProxyRef.Name
				envoyProxyConfigNamespace := kodeTemplate.Spec.EnvoyProxyRef.Namespace
				if envoyProxyConfigNamespace == "" {
					envoyProxyConfigNamespace = kode.Namespace
				}

				envoyProxyConfigNameObject := client.ObjectKey{Name: envoyProxyConfigName, Namespace: envoyProxyConfigNamespace}
				log.Info("Fetching EnvoyProxyConfig resource", "Name", envoyProxyConfigName, "Namespace", envoyProxyConfigNamespace)
				if err := r.Get(ctx, envoyProxyConfigNameObject, envoyProxyConfig); err != nil {
					if errors.IsNotFound(err) {
						log.Info("EnvoyProxyConfig resource not found in namespace, requeuing", "Namespace", envoyProxyConfigNamespace, "Name", envoyProxyConfigName)
						return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
					}
					log.Error(err, "Failed to fetch EnvoyProxyConfig resource",
						"EnvoyProxy_Namespace", envoyProxyConfig.Namespace,
						"EnvoyProxy_Name", envoyProxyConfig.Name,
						"KodeTemplate_Namespace", kodeTemplate.Namespace,
						"KodeTemplate_Name", kodeTemplate.Name)
					return ctrl.Result{Requeue: true}, err
				}
				log.Info("EnvoyProxyConfig resource found",
					"EnvoyProxy_Namespace", envoyProxyConfig.Namespace,
					"EnvoyProxy_Name", envoyProxyConfig.Name,
					"KodeTemplate_Namespace", kodeTemplate.Namespace,
					"KodeTemplate_Name", kodeTemplate.Name)
				labels["envoyproxy-config.kode.jacero.io/name"] = envoyProxyConfig.Name
				sharedEnvoyProxyConfigSpec = envoyProxyConfig.Spec.SharedEnvoyProxyConfigSpec
			}
		} else if kode.Spec.TemplateRef.Kind == "KodeClusterTemplate" {
			kodeClusterTemplate := &kodev1alpha1.KodeClusterTemplate{}
			kodeClusterTemplateName := kode.Spec.TemplateRef.Name
			kodeClusterTemplateNameObject := client.ObjectKey{Name: kodeClusterTemplateName}

			// Attempt to fetch the KodeClusterTemplate resource
			log.Info("Fetching KodeClusterTemplate resource", "Name", kodeClusterTemplateName)
			if err := r.Get(ctx, kodeClusterTemplateNameObject, kodeClusterTemplate); err != nil {
				if errors.IsNotFound(err) {
					log.Info("KodeClusterTemplate resource not found, requeuing", "Name", kodeClusterTemplateName)
					return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
				}
				log.Error(err, "Failed to fetch KodeClusterTemplate resource", "Name", kodeClusterTemplateName)
				return ctrl.Result{Requeue: true}, err
			}
			log.Info("KodeClusterTemplate resource found", "Name", kodeClusterTemplateName)
			labels["kode-cluster-template.kode.jacero.io/name"] = kodeClusterTemplate.Name
			sharedKodeTemplateSpec = kodeClusterTemplate.Spec.SharedKodeTemplateSpec

			// Attempt to fetch the EnvoyProxyClusterConfig resource
			if kodeClusterTemplate.Spec.EnvoyProxyRef.Name != "" {
				envoyProxyClusterConfig := &kodev1alpha1.EnvoyProxyClusterConfig{}
				envoyProxyClusterConfigName := kodeClusterTemplate.Spec.EnvoyProxyRef.Name
				envoyProxyClusterConfigNameObject := client.ObjectKey{Name: envoyProxyClusterConfigName}
				log.Info("Fetching EnvoyProxyClusterConfig resource", "Name", envoyProxyClusterConfigName)
				if err := r.Get(ctx, envoyProxyClusterConfigNameObject, envoyProxyClusterConfig); err != nil {
					if errors.IsNotFound(err) {
						log.Info("EnvoyProxyClusterConfig resource not found, requeuing", "Name", envoyProxyClusterConfigName)
						return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
					}
					log.Error(err, "Failed to fetch EnvoyProxyClusterConfig resource", "Name", envoyProxyClusterConfigName)
					return ctrl.Result{Requeue: true}, err
				}
				log.Info("EnvoyProxyClusterConfig resource found", "Name", envoyProxyClusterConfig.Name)
				log.Info("EnvoyProxyClusterConfig resource found",
					"EnvoyProxy_Name", envoyProxyClusterConfig.Name,
					"KodeTemplate_Name", kodeClusterTemplate.Name)
				labels["envoyproxy-cluster-config.kode.jacero.io/name"] = envoyProxyClusterConfig.Name
				sharedEnvoyProxyConfigSpec = envoyProxyClusterConfig.Spec.SharedEnvoyProxyConfigSpec
			}
		}
	}

	// Ensure the Deployment and Service exist
	if err := r.ensureDeployment(ctx, kode, labels, &sharedKodeTemplateSpec, &sharedEnvoyProxyConfigSpec); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureService(ctx, kode, labels, &sharedKodeTemplateSpec); err != nil {
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
	if kode.Spec.TemplateRef.Kind != "KodeTemplate" && kode.Spec.TemplateRef.Kind != "KodeClusterTemplate" {
		return fmt.Errorf("invalid reference kind for TemplateRef: %s", kode.Spec.TemplateRef.Kind)
	}
	return nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("Kode")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		// Watches(&kodev1alpha1.KodeTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.KodeClusterTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.EnvoyProxyConfig{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.EnvoyProxyClusterConfig{}, &handler.EnqueueRequestForObject{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
