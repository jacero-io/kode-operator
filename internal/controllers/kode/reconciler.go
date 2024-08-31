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
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
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
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
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

	// Update ObservedGeneration if it's out of sync and we're not requeuing
	if kode.Generation != kode.Status.ObservedGeneration && result.IsZero() {
		if err := r.updateObservedGeneration(ctx, kode); err != nil {
			log.Error(err, "Failed to update ObservedGeneration")
			return ctrl.Result{Requeue: true}, nil
		}
	}

	log.V(1).Info("Reconciliation completed", "result", result)
	return result, nil
}

func (r *KodeReconciler) transitionTo(ctx context.Context, kode *kodev1alpha2.Kode, newPhase kodev1alpha2.KodePhase) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode.Status.Phase == newPhase {
		// No transition needed
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Transitioning Kode state", "from", kode.Status.Phase, "to", newPhase)

	// Create a deep copy of the kode object to avoid modifying the cache
	kodeCopy := kode.DeepCopy()

	// Update the phase
	oldPhase := kodeCopy.Status.Phase
	kodeCopy.Status.Phase = newPhase

	// Add a condition to reflect the state change
	condition := metav1.Condition{
		Type:               string(newPhase),
		Status:             metav1.ConditionTrue,
		Reason:             "StateTransition",
		Message:            fmt.Sprintf("Kode transitioned from %s to %s state", oldPhase, newPhase),
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: kodeCopy.Generation,
	}

	// Update the status
	if err := r.StatusUpdater.UpdateStatusKode(ctx, kodeCopy, newPhase, []metav1.Condition{condition}, nil, "", nil); err != nil {
		log.Error(err, "Failed to update status for state transition")
		return ctrl.Result{Requeue: true}, err
	}

	// Perform any additional actions based on the new state
	var requeueAfter time.Duration
	switch newPhase {
	case kodev1alpha2.KodePhaseConfiguring:
		if kode.Status.Phase != kodev1alpha2.KodePhaseConfiguring {
			if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseConfiguring, []metav1.Condition{
				{
					Type:    "Configuring",
					Status:  metav1.ConditionTrue,
					Reason:  "EnteringConfiguringState",
					Message: "Kode is being configured.",
				},
			}, nil, "", nil); err != nil {
				log.Error(err, "Failed to update status for Configuring state")
				return ctrl.Result{Requeue: true}, err
			}
		}

		// Trigger immediate reconciliation to start configuration
		requeueAfter = 1 * time.Millisecond
	case kodev1alpha2.KodePhaseProvisioning:
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeProvisioning, "Kode is being provisioned")
		if err != nil {
			log.Error(err, "Failed to record Kode provisioning event")
		}
		// Trigger quick reconciliation to check resource readiness
		requeueAfter = 1 * time.Millisecond
	case kodev1alpha2.KodePhaseActive:
		// Maybe update related resources or send notifications
		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeActive, "Kode is now active"); err != nil {
			log.Error(err, "Failed to record Kode active event")
		}
	case kodev1alpha2.KodePhaseSuspending:
		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeSuspended, "Kode is being suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}
		// Start resource cleanup process
		requeueAfter = 1 * time.Millisecond
	case kodev1alpha2.KodePhaseSuspended:
		// Record suspension event
		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeSuspended, "Kode has been suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}
	case kodev1alpha2.KodePhaseResuming:
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeResuming, "Kode is resuming")
		if err != nil {
			log.Error(err, "Failed to record Kode resuming event")
		}
		// Start resuming process
		requeueAfter = 1 * time.Millisecond
	case kodev1alpha2.KodePhaseDeleting:
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeDeleting, "Kode is being deleted")
		if err != nil {
			log.Error(err, "Failed to record Kode deleting event")
		}
		// Trigger immediate reconciliation to start deletion process
		requeueAfter = 1 * time.Millisecond
	case kodev1alpha2.KodePhaseFailed:
		// Record failure event
		if err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonKodeFailed, "Kode has entered Failed state"); err != nil {
			log.Error(err, "Failed to record Kode failed event")
		}
		// Set up for potential retry
		requeueAfter = time.Minute
	case kodev1alpha2.KodePhaseUnknown:
		// Record unknown state event
		if err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonKodeUnknown, "Kode has entered Unknown state"); err != nil {
			log.Error(err, "Failed to record Kode unknown state event")
		}
		// Set up for quick recheck
		requeueAfter = 10 * time.Second
	}

	// Requeue to handle the new state
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *KodeReconciler) fetchTemplatesWithRetry(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.Template, error) {
	var template *kodev1alpha2.Template
	var lastErr error

	backoff := wait.Backoff{
		Steps:    5,                      // Maximum number of retries
		Duration: 100 * time.Millisecond, // Initial backoff duration
		Factor:   2.0,                    // Factor to increase backoff each try
		Jitter:   0.1,                    // Jitter factor
	}

	retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
		var err error
		template, err = r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
		if err == nil {
			return true, nil // Success
		}

		if errors.IsNotFound(err) {
			r.Log.Info("Template not found, will not retry", "error", err)
			return false, err // Don't retry if not found
		}

		// For other errors, log and retry
		r.Log.Error(err, "Failed to fetch template, will retry")
		lastErr = err
		return false, nil // Retry
	})

	if retryErr != nil {
		if errors.IsNotFound(retryErr) {
			return nil, fmt.Errorf("template not found after retries: %w", retryErr)
		}
		return nil, fmt.Errorf("failed to fetch template after retries: %w", lastErr)
	}

	return template, nil
}

func (r *KodeReconciler) updateObservedGeneration(ctx context.Context, kode *kodev1alpha2.Kode) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		if latest.Status.ObservedGeneration != latest.Generation {
			r.Log.Info("Updating ObservedGeneration",
				"Name", latest.Name,
				"Namespace", latest.Namespace,
				"OldObservedGeneration", latest.Status.ObservedGeneration,
				"NewObservedGeneration", latest.Generation)

			latest.Status.ObservedGeneration = latest.Generation
			return r.Client.Status().Update(ctx, latest)
		}

		r.Log.V(1).Info("ObservedGeneration already up to date",
			"Name", latest.Name,
			"Namespace", latest.Namespace,
			"Generation", latest.Generation)
		return nil
	})
}

func (r *KodeReconciler) updateRetryCount(ctx context.Context, kode *kodev1alpha2.Kode, count int) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		latest.Status.RetryCount = count
		return r.Client.Status().Update(ctx, latest)
	})
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
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
