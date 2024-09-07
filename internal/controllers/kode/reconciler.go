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

package kode

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

type KodeReconciler struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	ResourceManager resource.ResourceManager
	TemplateManager template.TemplateManager
	CleanupManager  cleanup.CleanupManager
	StatusUpdater   status.StatusUpdater
	Validator       validation.Validator
	EventManager    events.EventManager
}

const (
	RequeueInterval = 250 * time.Millisecond
)

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=podtemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clusterpodtemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=tofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clustertofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;watch;create;update;patch;delete,namespace=kode-system
// +kubebuilder:rbac:groups="storage.k8s.io",resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", req.NamespacedName)
	log.V(1).Info("Starting reconciliation")

	// Fetch the Kode instance
	kode := &kodev1alpha2.Kode{}
	if err := r.Client.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Unable to fetch Kode")
			return ctrl.Result{}, err
		}
		// Kode not found, likely deleted, nothing to do
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Fetched Kode resource", "Name", kode.Name, "Namespace", kode.Namespace, "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration, "Phase", kode.Status.Phase)

	// Handle deletion
	if !kode.DeletionTimestamp.IsZero() {
		return r.handleFinalizer(ctx, kode)
	}

	// Ensure finalizer is present (fallback)
	if !controllerutil.ContainsFinalizer(kode, common.KodeFinalizerName) {
		controllerutil.AddFinalizer(kode, common.KodeFinalizerName)
		if err := r.Client.Update(ctx, kode); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("Added finalizer to Kode resource")
		return ctrl.Result{Requeue: true}, nil
	}

	// Initialize status if it's a new resource
	if kode.Status.Phase == "" {
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhasePending)
	}

	// Reset retry count if we're not in a failed state
	if kode.Status.Phase != kodev1alpha2.KodePhaseFailed && kode.Status.RetryCount > 0 {
		if err := r.updateRetryCount(ctx, kode, 0); err != nil {
			log.Error(err, "Failed to reset retry count")
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Handle state transition
	var result ctrl.Result
	var err error

	switch kode.Status.Phase {
	case kodev1alpha2.KodePhasePending:
		result, err = r.handlePendingState(ctx, kode)
	case kodev1alpha2.KodePhaseConfiguring:
		result, err = r.handleConfiguringState(ctx, kode)
	case kodev1alpha2.KodePhaseProvisioning:
		result, err = r.handleProvisioningState(ctx, kode)
	case kodev1alpha2.KodePhaseActive:
		result, err = r.handleActiveState(ctx, kode)
	case kodev1alpha2.KodePhaseInactive:
		result, err = r.handleInactiveState(ctx, kode)
	case kodev1alpha2.KodePhaseSuspending:
		result, err = r.handleSuspendingState(ctx, kode)
	case kodev1alpha2.KodePhaseSuspended:
		result, err = r.handleSuspendedState(ctx, kode)
	case kodev1alpha2.KodePhaseResuming:
		result, err = r.handleResumingState(ctx, kode)
	case kodev1alpha2.KodePhaseDeleting:
		result, err = r.handleDeletingState(ctx, kode)
	case kodev1alpha2.KodePhaseFailed:
		result, err = r.handleFailedState(ctx, kode)
	case kodev1alpha2.KodePhaseUnknown:
		result, err = r.handleUnknownState(ctx, kode)
	default:
		log.Info("Unknown phase, transitioning to Unknown", "phase", kode.Status.Phase)
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseUnknown)
	}

	// Handle errors from state handlers
	if err != nil {
		log.Error(err, "Error handling state", "phase", kode.Status.Phase)
		if kode.Status.Phase != kodev1alpha2.KodePhaseFailed {
			// Transition to failed state if not already there
			return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
		}
		// If already in failed state, just requeue
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Reconciliation completed", "Phase", kode.Status.Phase, "result", result)
	return result, nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.Kode{}).
		// Owns(&corev1.Service{}).
		// Owns(&corev1.Secret{}).
		// Owns(&corev1.PersistentVolumeClaim{}).
		// Owns(&appsv1.StatefulSet{}).
		Watches(&kodev1alpha2.PodTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.ClusterPodTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.TofuTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.ClusterTofuTemplate{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
