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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

func handlePendingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Pending state")

	// Validate the Kode resource
	result := kode.Validate(ctx)
	if !result.Valid {
		var errMsgs []string
		for _, err := range result.Errors {
			errMsgs = append(errMsgs, err.Error())
			log.Error(err, "Kode validation failed")
		}
		combinedErrMsg := strings.Join(errMsgs, "; ")
		return handleReconcileError(ctx, r, kode, fmt.Errorf(combinedErrMsg), "Validation failed")
	}

	// Set conditions for Pending state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Pending", "Kode resource is pending configuration")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Pending", "Kode resource is not yet available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "Kode resource is being configured")

	// Update status
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
	}

	// Move to Configuring state
	log.Info("Moving to Configuring state")
	return kodev1alpha2.PhaseConfiguring, ctrl.Result{Requeue: true}, nil
}

func handleConfiguringState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Configuring state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}
	config := InitKodeResourcesConfig(kode, template)

	// Apply configuration
	if err := applyConfiguration(ctx, r, kode, config, nil); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to apply configuration")
	}

	// Validate configuration
	if err := validateConfiguration(ctx, r, kode, config); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to validate configuration")
	}

	// Update conditions
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Configuring", "Kode resources are being configured")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Configuring", "Kode is not yet available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "Kode resources are being configured")

	// Move to Provisioning state
	log.Info("Moving to Provisioning state")
	return kodev1alpha2.PhaseProvisioning, ctrl.Result{Requeue: true}, nil
}

func handleProvisioningState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Provisioning state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if all resources are ready
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources not ready, remaining in Provisioning state")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Provisioning", "Kode resources are being provisioned")
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
		}
		return kodev1alpha2.PhaseProvisioning, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// All resources are ready, move to Active state
	log.Info("All resources ready, moving to Active state")
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All Kode resources are ready")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "Kode is available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "Provisioned", "Kode resources are fully provisioned")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
	}

	return kodev1alpha2.PhaseActive, ctrl.Result{}, nil
}

func handleActiveState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Active state")

	// Check if any updates are needed
	if kode.Generation != kode.Status.ObservedGeneration {
		log.Info("Spec changed, transitioning to Updating state")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "SpecChanged", "Kode spec has changed and needs updating")
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
		}
		return kodev1alpha2.PhaseUpdating, ctrl.Result{Requeue: true}, nil
	}

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if resources are still ready
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources not ready, transitioning to Provisioning state")
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "KodeNotReady", "Kode resources are no longer ready")
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "KodeNotAvailable", "Kode may not be fully available")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Reprovisioning", "Kode is being reprovisioned")
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
		}
		return kodev1alpha2.PhaseProvisioning, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// Perform any regular active state tasks
	// TODO: Implement any regular active state tasks

	// Update conditions to ensure they reflect the current state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All Kode resources are ready")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesAvailable", "Kode is available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "Stable", "Kode is stable and not progressing")

	// Update status
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
	}

	log.V(1).Info("Kode remains in Active state")
	return kodev1alpha2.PhaseActive, ctrl.Result{RequeueAfter: r.GetLongReconcileInterval()}, nil
}

func handleUpdatingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Updating state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}
	config := InitKodeResourcesConfig(kode, template)

	// Detect changes
	changes, err := detectChanges(ctx, r, kode, config)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to detect changes")
	}

	// If no changes, move to Active state
	if len(changes) == 0 {
		log.Info("No changes detected, moving to Active state")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "NoChanges", "No changes detected during update")
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
		}
		return kodev1alpha2.PhaseActive, ctrl.Result{Requeue: true}, nil
	}

	// Apply updates
	if err := applyConfiguration(ctx, r, kode, config, changes); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to apply updates")
	}

	// Validate updates
	if err := validateConfiguration(ctx, r, kode, config); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to validate updates")
	}

	// Check if resources are ready after update
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources not ready after update, remaining in Updating state")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Updating", "Resources are still being updated")
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
		}
		return kodev1alpha2.PhaseUpdating, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// Update status
	kode.Status.ObservedGeneration = kode.Generation
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "UpdateComplete", "Kode resource has been successfully updated")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "UpdateComplete", "All Kode resources are ready after update")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "UpdateComplete", "Update process has completed")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
	}

	log.Info("Update completed successfully, moving to Active state")
	return kodev1alpha2.PhaseActive, ctrl.Result{}, nil
}

func handleDeletingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Deleting state")

	// Set conditions for deleting state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Deleting", "Kode resource is being deleted")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Deleting", "Kode resource is being deleted")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Deleting", "Kode resource is being deleted")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status")
	}

	// Check if child resources are deleted
	childResourcesDeleted, err := checkResourcesDeleted(ctx, r, kode)
	if err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to check child resources deletion status")
	}

	if !childResourcesDeleted {
		if kode.Status.DeletionCycle == 0 {
			kode.Status.DeletionCycle = 1
			log.Info("Initiating cleanup")
			cleanupResource := NewKodeCleanupResource(kode)
			result, err := r.GetCleanupManager().Cleanup(ctx, cleanupResource)
			if err != nil {
				return handleReconcileError(ctx, r, kode, err, "Failed to initiate cleanup")
			}

			// Update the status
			if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
				log.Error(err, "Failed to update Kode status after initiating cleanup")
				return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
			}

			return kodev1alpha2.PhaseDeleting, result, nil
		}

		log.Info("Child resources still exist")
		log.V(1).Info("Incrementing deletion cycle")
		kode.Status.DeletionCycle++

		// Update the status
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update Kode status after incrementing deletion cycle")
			return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
		}

		return kodev1alpha2.PhaseDeleting, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	log.Info("All child resources deleted, finalizing deletion")

	// Remove finalizer
	if controllerutil.ContainsFinalizer(kode, constant.KodeFinalizerName) {
		log.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(kode, constant.KodeFinalizerName)
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			if err := r.GetClient().Update(ctx, kode); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return handleReconcileError(ctx, r, kode, err, "Failed to remove finalizer")
			}
		}
	}

	log.Info("Kode resource deletion completed")
	return kodev1alpha2.PhaseDeleting, ctrl.Result{}, nil
}

func handleFailedState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Failed state")

	// Increment the retry count
	kode.Status.RetryCount++

	// Check if we've exceeded the maximum number of retries
	maxRetries := 5 // You might want to make this configurable
	if kode.Status.RetryCount > maxRetries {
		log.Info("Exceeded maximum number of retries", "retries", kode.Status.RetryCount, "max", maxRetries)
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "MaxRetriesExceeded", "Exceeded maximum number of retry attempts")
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "MaxRetriesExceeded", "Kode is not available due to persistent failures")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "MaxRetriesExceeded", "Reconciliation attempts have stopped due to persistent failures")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update Kode status")
		}

		// Remain in Failed state, but with a longer requeue time
		return kodev1alpha2.PhaseFailed, ctrl.Result{RequeueAfter: r.GetLongReconcileInterval()}, nil
	}

	// Log the retry attempt
	log.Info("Attempting to recover from failed state", "retry", kode.Status.RetryCount)

	// Update conditions to reflect recovery attempt
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "AttemptingRecovery", fmt.Sprintf("Attempting to recover (try %d/%d)", kode.Status.RetryCount, maxRetries))
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "RecoveryInProgress", "Kode is not available while recovery is in progress")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "AttemptingRecovery", "Attempting to recover from failed state")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		log.Error(err, "Failed to update Kode status")
		return kodev1alpha2.PhaseFailed, ctrl.Result{Requeue: true}, nil
	}

	// Attempt recovery by transitioning back to the Pending state
	// This will trigger a full reconciliation cycle
	log.Info("Transitioning back to Pending state for recovery")
	return kodev1alpha2.PhasePending, ctrl.Result{Requeue: true}, nil
}

func handleUnknownState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Unknown state")

	// Log the occurrence of an unknown state
	log.Error(nil, "Kode entered Unknown state", "currentPhase", kode.Status.Phase)

	// Update conditions to reflect the unknown state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionUnknown, "UnknownState", "Kode is in an unknown state")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionUnknown, "UnknownState", "Kode availability cannot be determined")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionUnknown, "UnknownState", "Kode progression cannot be determined")

	// Attempt to determine the correct state
	determinedPhase, err := determineCorrectState(ctx, r, kode)
	if err != nil {
		log.Error(err, "Failed to determine correct state")
		return handleReconcileError(ctx, r, kode, err, "Failed to determine correct state")
	}

	// If we successfully determined a phase, update the status and transition
	if determinedPhase != kodev1alpha2.PhaseUnknown {
		log.Info("Determined correct state", "newPhase", determinedPhase)
		kode.Status.Phase = determinedPhase
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status after determining correct state")
		}
		return determinedPhase, ctrl.Result{Requeue: true}, nil
	}

	// If we couldn't determine the correct state, fall back to Pending
	log.Info("Unable to determine correct state, falling back to Pending")
	kode.Status.Phase = kodev1alpha2.PhasePending
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update Kode status when falling back to Pending")
	}

	return kodev1alpha2.PhasePending, ctrl.Result{Requeue: true}, nil
}

// determineCorrectState attempts to figure out the correct state for the Kode
// based on its current status and the state of its resources
func determineCorrectState(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (kodev1alpha2.Phase, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	// Check if the Kode is being deleted
	if !kode.DeletionTimestamp.IsZero() {
		return kodev1alpha2.PhaseDeleting, nil
	}

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return kodev1alpha2.PhaseUnknown, err
	}

	config := InitKodeResourcesConfig(kode, template)

	// Check if resources exist
	// TODO: IMPLEMENT THIS FUNCTION
	// resourcesExist, err := checkResourcesExist(ctx, r, kode, config)
	// if err != nil {
	//     log.Error(err, "Failed to check if resources exist")
	//     return kodev1alpha2.PhaseUnknown, err
	// }

	// if !resourcesExist {
	//     return kodev1alpha2.PhasePending, nil
	// }

	// Check if resources are ready
	resourcesReady, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		log.Error(err, "Failed to check if resources are ready")
		return kodev1alpha2.PhaseUnknown, err
	}

	if !resourcesReady {
		return kodev1alpha2.PhaseProvisioning, nil
	}

	// If everything is set up and ready, consider it active
	return kodev1alpha2.PhaseActive, nil
}
