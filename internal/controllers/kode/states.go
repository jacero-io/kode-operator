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
	"github.com/jacero-io/kode-operator/internal/constants"
)

func (r *KodeReconciler) handlePendingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Pending state")

	// Validate the Kode resource
	if err := r.Validator.ValidateKode(ctx, kode); err != nil {
		log.Error(err, "Kode validation failed")

		// Update status to reflect validation failure
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "ValidationFailed", fmt.Sprintf("Kode validation failed: %v", err))
		kode.Status.Phase = kodev1alpha2.KodePhaseFailed
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}

		return ctrl.Result{}, nil // Will trigger status update in Reconcile
	}

	// Set conditions for Pending state
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Pending", "Kode resource is pending configuration")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Pending", "Kode resource is not yet available")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "Kode resource is being configured")

	// Remove any error conditions if present
	kode.DeleteCondition(constants.ConditionTypeError)

	// Clear any previous error
	kode.Status.LastError = nil
	kode.Status.LastErrorTime = nil

	// Transition to Configuring state
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
}

func (r *KodeReconciler) handleConfiguringState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Configuring state")

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Ensure all necessary resources are created
	if err := r.ensurePodResources(ctx, kode, config); err != nil {
		log.Error(err, "Failed to ensure pod resources")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "ResourceCreationFailed", fmt.Sprintf("Failed to create resources: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}

	// Update ObservedGeneration if it's out of sync
	if kode.Generation != kode.Status.ObservedGeneration {
		kode.Status.ObservedGeneration = kode.Generation
		log.V(1).Info("Updated ObservedGeneration", "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration)
	}

	// Update conditions
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Configuring", "Kode resources are being configured")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Configuring", "Kode is not yet available")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "Kode resources are being configured")

	// Clear any previous error
	kode.Status.LastError = nil
	kode.Status.LastErrorTime = nil

	// Transition to Provisioning state
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseProvisioning)
}

func (r *KodeReconciler) handleProvisioningState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Provisioning state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if kode.Generation != kode.Status.ObservedGeneration {
		kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "Kode resource is being reconfigured")
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if all resources are ready
	ready, err := r.checkPodResources(ctx, kode, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "ResourceCheckFailed", fmt.Sprintf("Failed to check resource readiness: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}

	if ready {
		// Update the port status
		if err := kode.UpdatePort(ctx, r.Client, template.Port); err != nil {
			log.Error(err, "Failed to update port status")
			kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "PortUpdateFailed", fmt.Sprintf("Failed to update port status: %v", err))
			kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "PortUpdateFailed", "Kode is not available due to port update failure")
			return ctrl.Result{Requeue: true}, nil
		}

		log.Info("Resources are ready, transitioning to Active state")

		// Set conditions for ready state
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All Kode resources are ready")
		kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "ResourcesReady", "Kode is waiting for the url to be available")
		kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionFalse, "ResourcesReady", "Kode resources are fully provisioned")

		// Clear any previous error
		kode.Status.LastError = nil
		kode.Status.LastErrorTime = nil

		// Transition to Active state
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseActive)
	}

	// Resources are not ready yet, update conditions and requeue
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Provisioning", "Kode resources are still being provisioned")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Provisioning", "Kode is not yet available")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Provisioning", "Kode resources are being provisioned")

	// Requeue to check again
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *KodeReconciler) handleActiveState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Active state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if kode.Generation != kode.Status.ObservedGeneration {
		kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "Kode resource is being reconfigured")
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if resources are still ready
	ready, err := r.checkPodResources(ctx, kode, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "ResourceCheckFailed", fmt.Sprintf("Failed to check resource readiness: %v", err))
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
	}

	if !ready {
		log.Info("Resources not ready, transitioning back to Configuring state")
		kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "ResourcesNotReady", "Kode resources are no longer ready")
		kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "ResourcesNotReady", "Kode is no longer available")
		kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "Kode is being reconfigured")
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// Check if the Kode should be suspended due to inactivity
	// TODO: Write checks for inactivity

	// Update conditions to ensure they reflect the current state
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All Kode resources are ready")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "Kode is available")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionFalse, "Stable", "Kode is stable and not progressing")

	// Clear any previous error
	kode.Status.LastError = nil
	kode.Status.LastErrorTime = nil

	// TODO: Implement level-triggered updates instead of edge-triggered with requeue delay
	// No state transition needed, return without error
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

func (r *KodeReconciler) handleInactiveState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Inactive state")

	// Start suspension process
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseSuspending)
}

// TODO: Implement suspending logic
func (r *KodeReconciler) handleSuspendingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Suspending state")

	// Update conditions
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Suspending", "Kode is being suspended")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Suspending", "Kode is not available during suspension")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Suspending", "Kode is being suspended")

	// Transition to Suspended state
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseSuspended)
}

// TODO: Implement suspended logic
func (r *KodeReconciler) handleSuspendedState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Suspended state")

	// Update conditions
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Suspended", "Kode is suspended")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Suspended", "Kode is not available while suspended")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionFalse, "Suspended", "Kode is suspended and not progressing")

	// No state transition needed, return without error
	return ctrl.Result{}, nil
}

// TODO: Implement resuming logic
func (r *KodeReconciler) handleResumingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Resuming state")

	// Update conditions
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Resuming", "Kode is being resumed")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Resuming", "Kode is not yet available")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Resuming", "Kode is being resumed")

	// Transition to Configuring state to ensure everything is properly set up
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
}

func (r *KodeReconciler) handleDeletingState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Deleting state")

	// Set conditions for deleting state
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Deleting", "Kode resource is being deleted")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Deleting", "Kode resource is being deleted")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Deleting", "Kode resource is being deleted")

	// Check if all child resources are deleted
	log.V(1).Info("Checking if child resources are deleted")
	childResourcesDeleted, err := r.checkResourcesDeleted(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to check child resources deletion status")
		return ctrl.Result{Requeue: true}, err
	}

	if !childResourcesDeleted {
		log.Info("Child resources still exist")
		if kode.Status.DeletionCycle == 0 {
			log.Info("Initiating cleanup")
			cleanupResource := NewKodeCleanupResource(kode)
			result, err := r.CleanupManager.Cleanup(ctx, cleanupResource)
			if err != nil {
				log.Error(err, "Failed to initiate cleanup")
				kode.SetCondition(constants.ConditionTypeError, metav1.ConditionTrue, "CleanupFailed", fmt.Sprintf("Failed to initiate cleanup: %v", err))
				return result, err
			}
			return result, nil
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	log.V(1).Info("All child resources are deleted")

	// All resources are deleted, remove finalizer
	if controllerutil.ContainsFinalizer(kode, constants.KodeFinalizerName) {
		log.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(kode, constants.KodeFinalizerName)
		if err := r.Client.Update(ctx, kode); err != nil {
			log.Error(err, "Failed to remove finalizer")
			kode.SetCondition(constants.ConditionTypeError, metav1.ConditionTrue, "FinalizerRemovalFailed", fmt.Sprintf("Failed to remove finalizer: %v", err))
			return ctrl.Result{Requeue: true}, err
		}
	}

	log.Info("Kode resource deletion complete")
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) handleFailedState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Handling Failed state")

	// Increment retry count
	kode.Status.RetryCount++

	// Set conditions for failed state
	kode.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Failed", "Kode resource is in a failed state")
	kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Failed", "Kode resource is not available due to failure")
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionFalse, "Failed", "Kode resource progression halted due to failure")

	// Check if max retries exceeded
	if kode.Status.RetryCount > 10 {
		log.Info("Max retries exceeded, manual intervention required")
		kode.SetCondition(constants.ConditionTypeError, metav1.ConditionTrue, "MaxRetriesExceeded", "Maximum retry attempts reached. Manual intervention required.")
		return ctrl.Result{}, nil
	}

	// Attempt to recover by transitioning back to Pending state
	log.Info("Attempting recovery", "retryCount", kode.Status.RetryCount)
	kode.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Recovering", "Attempting to recover from failed state")

	// Clear error conditions as we're attempting recovery
	kode.DeleteCondition(constants.ConditionTypeError)
	kode.Status.LastError = nil
	kode.Status.LastErrorTime = nil

	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhasePending)
}

func (r *KodeReconciler) handleUnknownState(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
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
