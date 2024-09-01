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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

func (r *KodeReconciler) handlePendingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Pending state")

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(kode, common.KodeFinalizerName) {
		controllerutil.AddFinalizer(kode, common.KodeFinalizerName)
		if err := r.Client.Update(ctx, kode); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("Added finalizer to Kode resource")
	}

	// Validate the Kode resource
	if err := r.Validator.ValidateKode(ctx, kode); err != nil {
		log.Error(err, "Kode validation failed")
		// Update status to reflect validation failure
		if updateErr := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{
			{
				Type:    "ValidationFailed",
				Status:  metav1.ConditionTrue,
				Reason:  "InvalidConfiguration",
				Message: fmt.Sprintf("Kode validation failed: %v", err),
			},
		}, nil, "", nil); updateErr != nil {
			log.Error(updateErr, "Failed to update status for validation failure")
		}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}

	// Transition to Configuring state
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
}

func (r *KodeReconciler) handleConfiguringState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Configuring state")

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return r.handleTemplateFetchError(ctx, kode, err)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Ensure all necessary resources are created
	if err := r.ensurePodResources(ctx, kode, config); err != nil {
		log.Error(err, "Failed to ensure pod resources")
		return r.handleResourceCreationError(ctx, kode, err)
	}

	// Update ObservedGeneration if it's out of sync
	if kode.Generation != kode.Status.ObservedGeneration {
		if err := r.updateObservedGeneration(ctx, kode); err != nil {
			log.Error(err, "Failed to update ObservedGeneration")
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("Updated ObservedGeneration", "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration)
	}

	// Transition to Provisioning state
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseProvisioning)
}

func (r *KodeReconciler) handleProvisioningState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Provisioning state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if kode.Generation != kode.Status.ObservedGeneration {
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return r.handleTemplateFetchError(ctx, kode, err)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if all resources are ready
	ready, err := r.checkPodResources(ctx, kode, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		return r.handleResourceCheckError(ctx, kode, err)
	}

	if ready {
		// Update the port status
		if err := r.updatePortStatus(ctx, kode, template); err != nil {
			log.Error(err, "Failed to update port status")
			return ctrl.Result{Requeue: true}, err
		}

		// Transition to Active state
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseActive)
	}

	// Requeue to check again
	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *KodeReconciler) handleActiveState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Active state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if kode.Generation != kode.Status.ObservedGeneration {
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return r.handleTemplateFetchError(ctx, kode, err)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if resources are still ready
	ready, err := r.checkPodResources(ctx, kode, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		return r.handleResourceCheckError(ctx, kode, err)
	}

	if !ready {
		log.Info("Resources not ready, transitioning back to Configuring state")
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Do not requeue if resources are ready
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) handleInactiveState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Inactive state")

	// Start suspension process
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseSuspending)
}

// TODO: Implement suspending logic
func (r *KodeReconciler) handleSuspendingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Suspending state")
	if kode.Status.Phase != kodev1alpha2.KodePhaseSuspending {
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseSuspending, []metav1.Condition{
			{
				Type:    "Suspending",
				Status:  metav1.ConditionTrue,
				Reason:  "EnteringSuspendingState",
				Message: "Kode is being suspended.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for Suspending state")
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{Requeue: true}, nil
}

// TODO: Implement suspended logic
func (r *KodeReconciler) handleSuspendedState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Suspending state")

	return ctrl.Result{Requeue: true}, nil
}

// TODO: Implement resuming logic
func (r *KodeReconciler) handleResumingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Suspending state")

	if kode.Status.Phase != kodev1alpha2.KodePhaseResuming {
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseResuming, []metav1.Condition{
			{
				Type:    "Resuming",
				Status:  metav1.ConditionTrue,
				Reason:  "EnteringResumingState",
				Message: "Kode is resuming.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for Resuming state")
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *KodeReconciler) handleDeletingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Deleting state")

	if kode.Status.Phase != kodev1alpha2.KodePhaseDeleting {
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseDeleting, []metav1.Condition{
			{
				Type:    "Deleting",
				Status:  metav1.ConditionTrue,
				Reason:  "EnteringDeletingState",
				Message: "Kode is being deleted.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for Deleting state")
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Check if deletion is complete
	if kode.DeletionTimestamp.IsZero() {
		// Deletion is complete, transition to a final state or remove the resource
		log.Info("Deletion complete, removing resource")
		return ctrl.Result{}, nil
	}

	// Deletion is still in progress
	log.Info("Deletion in progress, waiting for completion")
	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *KodeReconciler) handleFailedState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Failed state")

	// Increment retry count
	newRetryCount := kode.Status.RetryCount + 1
	if err := r.updateRetryCount(ctx, kode, newRetryCount); err != nil {
		log.Error(err, "Failed to update retry count")
		return ctrl.Result{Requeue: true}, err
	}

	// Check if max retries exceeded
	if newRetryCount > 5 {
		log.Info("Max retries exceeded, manual intervention required")
		// Update status to reflect max retries exceeded
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{
			{
				Type:    "MaxRetriesExceeded",
				Status:  metav1.ConditionTrue,
				Reason:  "RetryLimitReached",
				Message: "Maximum retry attempts reached. Manual intervention required.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for max retries exceeded")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// Attempt to recover by transitioning back to Pending state
	log.Info("Attempting recovery", "retryCount", newRetryCount)
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhasePending)
}

func (r *KodeReconciler) handleUnknownState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Handling Unknown state")

	// Attempt to determine the correct state
	currentState, err := r.determineCurrentState(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to determine current state")
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}

	// Transition to the determined state
	return r.transitionTo(ctx, kode, currentState)
}
