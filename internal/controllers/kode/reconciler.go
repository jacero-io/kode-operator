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
	"reflect"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/constant"
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

type KodeReconciler struct {
	Client            client.Client
	Scheme            *runtime.Scheme
	Log               logr.Logger
	ResourceManager   resource.ResourceManager
	TemplateManager   template.TemplateManager
	CleanupManager    cleanup.CleanupManager
	Validator         validation.Validator
	EventManager      event.EventManager
	IsTestEnvironment bool
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=containertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clustercontainertemplates,verbs=get;list;watch
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
// +kubebuilder:rbac:groups="",resources=event,verbs=create;patch;update

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
		log.V(1).Info("Kode resource not found, likely deleted")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Fetched Kode resource", "Name", kode.Name, "Namespace", kode.Namespace, "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration, "Phase", kode.Status.Phase)

	// **Add finalizer if not present**
	if !controllerutil.ContainsFinalizer(kode, constant.KodeFinalizerName) {
		controllerutil.AddFinalizer(kode, constant.KodeFinalizerName)
		if err := r.Client.Update(ctx, kode); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("Added finalizer to Kode resource")
		// Requeue to ensure the updated resource is processed
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle phase transitions
	var result ctrl.Result
	var err error

	// Transition to Deleting state if deletion timestamp is set and not already in deleting state
	if !kode.DeletionTimestamp.IsZero() && kode.Status.Phase != kodev1alpha2.KodePhaseDeleting {
		result, err = r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseDeleting)
		return result, err // Early return after transition
	} else if kode.Generation != kode.Status.ObservedGeneration && kode.Status.Phase != kodev1alpha2.KodePhaseDeleting && kode.Status.Phase != kodev1alpha2.KodePhaseConfiguring { // Transition to Configuring state if generation mismatch
		log.Info("Generation mismatch, transitioning to Configuring state")
		result, err = r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
		return result, err // Early return after transition
	}

	// Transition to Pending state if no phase is set
	if kode.Status.Phase == "" {
		log.Info("Transitioning to Pending state")
		result, err := r.transitionTo(ctx, kode, kodev1alpha2.KodePhasePending)
		return result, err // Early return after transition
	}

	// Reset retry count if we're not in a failed state
	if kode.Status.Phase != kodev1alpha2.KodePhaseFailed && kode.Status.RetryCount > 0 {
		if err := r.updateRetryCount(ctx, kode, 0); err != nil {
			log.Error(err, "Failed to reset retry count")
			return ctrl.Result{Requeue: true}, err
		}
	}

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
		result, err = r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseUnknown)
		return result, err // Early return after transition
	}

	// Handle errors from state handlers
	if err != nil {
		log.Error(err, "Error handling state", "phase", kode.Status.Phase)
		if kode.Status.Phase != kodev1alpha2.KodePhaseFailed {
			// Transition to failed state if not already there
			result, err = r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
			return result, err // Early return after transition
		}
		// If already in failed state, just requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Kode resource still exists before updating status
	latestKode := &kodev1alpha2.Kode{}
	if err := r.Client.Get(ctx, req.NamespacedName, latestKode); err != nil {
		if errors.IsNotFound(err) {
			// Kode resource has been deleted, nothing to update
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get latest Kode")
		return ctrl.Result{Requeue: true}, err
	}

	// Update the whole status if it has changed
	if !reflect.DeepEqual(latestKode.Status, kode.Status) {
		if err := r.updateStatus(ctx, kode); err != nil {
			// If we fail to update the status, requeue
			return ctrl.Result{Requeue: true}, err
		}
	}

	log.V(1).Info("Reconciliation completed", "Phase", kode.Status.Phase, "result", result)
	return result, nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.Kode{}).
		// Watches(&kodev1alpha2.ContainerTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.ClusterContainerTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.TofuTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.ClusterTofuTemplate{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
