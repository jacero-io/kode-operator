/*
Copyright 2024 Emil Larsson.

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

package entrypoint

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

type EntryPointReconciler struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	ResourceManager resource.ResourceManager
	TemplateManager template.TemplateManager
	CleanupManager  cleanup.CleanupManager
	StatusUpdater   status.StatusUpdater
	Validator       validation.Validator
}

const (
	RequeueInterval = 250 * time.Millisecond
)

// +kubebuilder:rbac:groups=kode.jacero.io,resources=clusterentrypoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clusterentrypoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clusterentrypoints/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/finalizers,verbs=update

func (r *EntryPointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("clusterEntryPoint", req.NamespacedName)

	var obj client.Object
	if req.Namespace == "" {
		obj = &kodev1alpha2.ClusterEntryPoint{}
	} else {
		obj = &kodev1alpha2.Kode{}
	}

	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch v := obj.(type) {
	case *kodev1alpha2.ClusterEntryPoint:
		return r.reconcileEntryPoint(ctx, v)
	case *kodev1alpha2.Kode:
		return r.reconcileKode(ctx, v)
	default:
		log.Error(nil, "Unknown resource type", "type", fmt.Sprintf("%T", obj))
		return ctrl.Result{}, nil
	}
}

func (r *EntryPointReconciler) reconcileEntryPoint(ctx context.Context, entryPoint *kodev1alpha2.ClusterEntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entryPoint", entryPoint.Name)

	// Fetch the latest reconcileEntryPoint object
	latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, entryPoint.Name)
	if err != nil {
		log.Error(err, "Failed to get latest ClusterEntryPoint")
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	config := InitEntryPointResourcesConfig(latestEntryPoint)

	// Ensure Gateway
	if err := r.ensureGateway(ctx, config, latestEntryPoint); err != nil {
		log.Error(err, "Failed to ensure Gateway")
		r.updateStatus(ctx, latestEntryPoint, kodev1alpha2.EntryPointPhaseFailed, []metav1.Condition{{
			Type:    "GatewayCreationFailed",
			Status:  metav1.ConditionTrue,
			Reason:  "GatewayCreationError",
			Message: "Failed to create Gateway",
		}}, err)
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) reconcileKode(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", kode.Name)

	// Find the associated ClusterEntryPoint
	entryPoint, err := r.findEntryPointForKode(ctx, kode)
	if err != nil {
		log.Error(err, "Unable to find ClusterEntryPoint for Kode")
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	config := InitEntryPointResourcesConfig(entryPoint)

	// Ensure HTTPRoute
	if err := r.ensureHTTPRoute(ctx, config, entryPoint, *kode); err != nil {
		log.Error(err, "Failed to ensure HTTPRoute")
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) findEntryPointForKode(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.ClusterEntryPoint, error) {
	template, err := r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	var entryPointRef kodev1alpha2.EntryPointReference
	switch template.Kind {
	case "PodTemplate", "ClusterPodTemplate":
		entryPointRef = template.PodTemplateSpec.EntryPointRef
	case "TofuTemplate", "ClusterTofuTemplate":
		entryPointRef = template.TofuTemplateSpec.EntryPointRef
	default:
		return nil, fmt.Errorf("unknown template kind: %s", template.Kind)
	}

	// Fetch the latest reconcileEntryPoint object
	latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, entryPointRef.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterEntryPoint: %w", err)
	}

	return latestEntryPoint, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryPointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.Kode{}).
		For(&kodev1alpha2.ClusterEntryPoint{}).
		Complete(r)
}
