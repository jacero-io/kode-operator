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
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/constant"
	"github.com/jacero-io/kode-operator/pkg/validation"
)

func handlePendingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.Info("Handling Pending state")

	// Validate the Kode resource
	result := validation.ValidateKode(ctx, kode)
	if !result.Valid {
		for _, errMsg := range result.Errors {
			log.Error(fmt.Errorf(errMsg), "Kode validation failed")
		}
		combinedErrMsg := strings.Join(result.Errors, "; ")
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

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Configuring state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}

	config := InitKodeResourcesConfig(kode, template)

	// Detect required changes
	changes, err := detectChanges(ctx, r, kode, config)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ChangeDetectionFailed", fmt.Sprintf("Failed to detect configuration changes: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to detect changes")
	}

	// Record significant changes if any
	if len(changes) > 0 {
		changeList := make([]string, 0, len(changes))
		for resource := range changes {
			changeList = append(changeList, resource)
		}
		er.Record(ctx, kode, event.EventTypeNormal, "ConfigurationRequired", fmt.Sprintf("Configuration required for: %s", strings.Join(changeList, ", ")))
	}

	// Apply configuration
	if err := applyConfiguration(ctx, r, kode, config, changes); err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ConfigurationFailed", fmt.Sprintf("Failed to apply configuration: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to apply configuration")
	}

	// Record successful configuration
	if len(changes) > 0 {
		er.Record(ctx, kode, event.EventTypeNormal, "ConfigurationApplied", "Successfully applied resource configuration")
	}

	// Validate configuration
	if err := validateConfiguration(ctx, r, kode, config); err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ValidationFailed", fmt.Sprintf("Configuration validation failed: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to validate configuration")
	}

	// Update conditions
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Configuring", "Kode resources are being configured")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Configuring", "Kode is not yet available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "Kode resources are being configured")

	// Update status
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update status")
	}

	// Move to Provisioning state
	log.Info("Moving to Provisioning state")

	return kodev1alpha2.PhaseProvisioning, ctrl.Result{Requeue: true}, nil
}

func handleProvisioningState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Provisioning state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}

	config := InitKodeResourcesConfig(kode, template)

	// Check if all resources are ready
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ReadinessCheckFailed", fmt.Sprintf("Failed to check resource readiness: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources not ready, remaining in Provisioning state")

		// Update conditions for ongoing provisioning
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Provisioning", "Resources are still being provisioned")
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Provisioning", "Resources are not yet available")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Provisioning", "Resources are being provisioned")

		// Update status
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}

		// Requeue for continued provisioning check
		return kodev1alpha2.PhaseProvisioning, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// All resources are ready, prepare for transition to Active state
	log.Info("All resources ready, moving to Active state")

	// Update conditions for successful provisioning
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All Kode resources are ready")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "Kode is available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "Provisioned", "Kode resources are fully provisioned")

	// Update status with port information
	kode.Status.KodePort = template.Port

	// Update status
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update status")
	}

	// Record successful provisioning
	er.Record(ctx, kode, event.EventTypeNormal, "ProvisioningComplete", "All resources successfully provisioned and ready")

	return kodev1alpha2.PhaseActive, ctrl.Result{}, nil
}

func handleActiveState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Active state")

	// Check for generation mismatch (spec changes)
	if kode.Generation != kode.Status.ObservedGeneration {
		log.Info("Spec changed, transitioning to Updating state", "currentGeneration", kode.Generation, "observedGeneration", kode.Status.ObservedGeneration)

		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "UpdateRequired", "Kode spec has changed and needs updating")
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "UpdatePending", "Update is pending due to spec changes")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}

		er.Record(ctx, kode, event.EventTypeNormal, "UpdateDetected", "Configuration changes detected, initiating update")
		return kodev1alpha2.PhaseUpdating, ctrl.Result{Requeue: true}, nil
	}

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template during active check: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}

	config := InitKodeResourcesConfig(kode, template)

	// Verify resources are still healthy
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ReadinessCheckFailed", fmt.Sprintf("Failed to check resource health: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources no longer ready, transitioning to Updating state")

		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ResourcesNotReady", "One or more resources are no longer ready")
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ServiceDegraded", "Service may be degraded")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Updating", "Attempting to restore resource health")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}

		er.Record(ctx, kode, event.EventTypeWarning, "HealthDegraded", "Resource health check failed, initiating recovery")
		return kodev1alpha2.PhaseUpdating, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// Update URL if template changed
	if kode.Status.KodePort != template.Port {
		kode.Status.KodePort = template.Port

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}
	}

	// Ensure conditions reflect healthy state
	// Only update if conditions have changed to avoid unnecessary updates
	conditionsChanged := false

	if kode.GetCondition(constant.ConditionTypeReady).Status != metav1.ConditionTrue ||
		kode.GetCondition(constant.ConditionTypeReady).Reason != "ResourcesHealthy" {
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesHealthy", "All resources are healthy")
		conditionsChanged = true
	}

	if kode.GetCondition(constant.ConditionTypeAvailable).Status != metav1.ConditionTrue ||
		kode.GetCondition(constant.ConditionTypeAvailable).Reason != "ServiceAvailable" {
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ServiceAvailable", "Service is operating normally")
		conditionsChanged = true
	}

	if kode.GetCondition(constant.ConditionTypeProgressing).Status != metav1.ConditionFalse ||
		kode.GetCondition(constant.ConditionTypeProgressing).Reason != "Stable" {
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "Stable", "No ongoing changes")
		conditionsChanged = true
	}

	// Only update status if conditions changed
	if conditionsChanged {
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}
	}

	log.Info("Kode remains in Active state")
	return kodev1alpha2.PhaseActive, ctrl.Result{RequeueAfter: r.GetLongReconcileInterval()}, nil
}

func handleUpdatingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Updating state")

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template during update: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to fetch template")
	}

	config := InitKodeResourcesConfig(kode, template)

	// Detect changes
	changes, err := detectChanges(ctx, r, kode, config)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ChangeDetectionFailed", fmt.Sprintf("Failed to detect configuration changes: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to detect changes")
	}

	// If no changes, move to Active state
	if len(changes) == 0 {
		log.Info("No changes detected, moving to Active state")

		kode.Status.ObservedGeneration = kode.Generation
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "NoChanges", "No changes detected during update")
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesHealthy", "All resources are healthy")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}

		return kodev1alpha2.PhaseActive, ctrl.Result{Requeue: true}, nil
	}

	// Apply updates
	log.Info("Applying configuration changes", "changes", changes)
	if err := applyConfiguration(ctx, r, kode, config, changes); err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "UpdateFailed", fmt.Sprintf("Failed to apply updates: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to apply updates")
	}

	// Validate updates
	if err := validateConfiguration(ctx, r, kode, config); err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ValidationFailed", fmt.Sprintf("Update validation failed: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to validate updates")
	}

	// Check if resources are ready after update
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "ReadinessCheckFailed", fmt.Sprintf("Failed to check resource readiness after update: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to check resource readiness")
	}

	if !ready {
		log.Info("Resources not ready after update, remaining in Updating state")

		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "UpdateInProgress", "Update in progress, waiting for resources to be ready")
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "UpdateInProgress", "Resources are being updated")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			return handleReconcileError(ctx, r, kode, err, "Failed to update status")
		}

		return kodev1alpha2.PhaseUpdating, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// Update completed successfully
	log.Info("Update completed successfully")

	// Update status
	kode.Status.ObservedGeneration = kode.Generation
	kode.Status.KodePort = template.Port

	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "UpdateComplete", "Update completed successfully")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ServiceAvailable", "Service is operating normally")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "UpdateComplete", "Update process has completed")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		return handleReconcileError(ctx, r, kode, err, "Failed to update status")
	}

	er.Record(ctx, kode, event.EventTypeNormal, "UpdateComplete", "Resource update completed successfully")

	log.Info("Moving to Active state")
	return kodev1alpha2.PhaseActive, ctrl.Result{}, nil
}

func handleDeletingState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Deleting state")

	// Set conditions for deleting state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Deleting", "Resource is being deleted")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Deleting", "Resource is shutting down")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Deleting", "Resource deletion in progress")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		log.Error(err, "Failed to update status during deletion")
		return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
	}

	// Check if child resources are deleted
	childResourcesDeleted, err := checkResourcesDeleted(ctx, r, kode)
	if err != nil {
		er.Record(ctx, kode, event.EventTypeWarning, "DeletionCheckFailed", fmt.Sprintf("Failed to check child resources deletion status: %v", err))
		return handleReconcileError(ctx, r, kode, err, "Failed to check child resources deletion")
	}

	if !childResourcesDeleted {
		if kode.Status.DeletionCycle == 0 {
			// First deletion attempt
			kode.Status.DeletionCycle = 1
			log.Info("Initiating resource cleanup", "deletionCycle", kode.Status.DeletionCycle)

			cleanupResource := NewKodeCleanupResource(kode)
			result, err := r.GetCleanupManager().Cleanup(ctx, cleanupResource)
			if err != nil {
				er.Record(ctx, kode, event.EventTypeWarning, "CleanupFailed", fmt.Sprintf("Failed to initiate cleanup: %v", err))
				return handleReconcileError(ctx, r, kode, err, "Failed to initiate cleanup")
			}

			er.Record(ctx, kode, event.EventTypeNormal, "DeletionStarted", "Resource cleanup initiated")

			// Update the status
			if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
				log.Error(err, "Failed to update status after cleanup initiation")
				return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
			}

			return kodev1alpha2.PhaseDeleting, result, nil
		}

		// Resources still exist, increment deletion cycle
		log.Info("Child resources still exist", "deletionCycle", kode.Status.DeletionCycle)

		kode.Status.DeletionCycle++
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update deletion cycle")
			return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
		}

		// If deletion is taking too long, record a warning
		if kode.Status.DeletionCycle > 10 {
			er.Record(ctx, kode, event.EventTypeWarning, "DeletionDelayed", fmt.Sprintf("Resource deletion taking longer than expected (cycle %d)", kode.Status.DeletionCycle))
		}

		return kodev1alpha2.PhaseDeleting, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, nil
	}

	// All child resources are deleted, remove finalizer
	log.Info("All child resources deleted, removing finalizer")

	if controllerutil.ContainsFinalizer(kode, constant.KodeFinalizerName) {
		if err := kode.RemoveFinalizer(ctx, r.GetClient()); err != nil {
			er.Record(ctx, kode, event.EventTypeWarning, "FinalizerRemovalFailed", fmt.Sprintf("Failed to remove finalizer: %v", err))
			return handleReconcileError(ctx, r, kode, err, "Failed to remove finalizer")
		}

		er.Record(ctx, kode, event.EventTypeNormal, "DeletionComplete", "Resource cleanup completed successfully")
	}

	log.Info("Resource deletion completed")
	return kodev1alpha2.PhaseDeleting, ctrl.Result{}, nil
}

func handleFailedState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Failed state")

	// Check maximum retries (consider making this configurable)
	const maxRetries = 5
	if kode.Status.RetryCount >= maxRetries {
		log.Info("Maximum retry attempts exceeded", "retryCount", kode.Status.RetryCount, "maxRetries", maxRetries)

		// Update conditions to reflect permanent failure
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "MaxRetriesExceeded", fmt.Sprintf("Recovery failed after %d attempts", maxRetries))
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ServiceUnavailable", "Service is unavailable due to persistent failures")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "RecoveryAbandoned", "Recovery attempts abandoned due to maximum retries")

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update status for max retries")
			return kodev1alpha2.PhaseFailed, ctrl.Result{Requeue: true}, nil
		}

		er.Record(ctx, kode, event.EventTypeWarning, "RecoveryAbandoned", fmt.Sprintf("Recovery abandoned after %d failed attempts. Manual intervention required", maxRetries))

		// Stay in Failed state but with longer requeue time
		return kodev1alpha2.PhaseFailed, ctrl.Result{RequeueAfter: r.GetLongReconcileInterval()}, nil
	}

	// Increment retry count
	kode.Status.RetryCount++
	log.Info("Attempting recovery", "attempt", kode.Status.RetryCount, "maxRetries", maxRetries)

	// Update conditions for recovery attempt
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "AttemptingRecovery", fmt.Sprintf("Recovery attempt %d of %d", kode.Status.RetryCount, maxRetries))
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "RecoveryInProgress", "Service availability unknown during recovery")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "RecoveryInProgress", fmt.Sprintf("Attempting recovery (try %d/%d)", kode.Status.RetryCount, maxRetries))

	// Clear error state if this is a new retry attempt
	if kode.Status.LastError != nil {
		kode.Status.LastError = nil
		kode.Status.LastErrorTime = nil
	}

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		log.Error(err, "Failed to update status before recovery attempt")
		return kodev1alpha2.PhaseFailed, ctrl.Result{Requeue: true}, nil
	}

	er.Record(ctx, kode, event.EventTypeNormal, "RecoveryAttempt", fmt.Sprintf("Starting recovery attempt %d of %d", kode.Status.RetryCount, maxRetries))

	// Determine appropriate recovery phase
	recoveryPhase := determineRecoveryPhase(ctx, r, kode)
	log.Info("Determined recovery phase", "recoveryPhase", recoveryPhase)

	// Record recovery transition
	er.Record(ctx, kode, event.EventTypeNormal, "RecoveryTransition", fmt.Sprintf("Transitioning to %s phase for recovery", recoveryPhase))

	// Return to appropriate phase for recovery
	return recoveryPhase, ctrl.Result{Requeue: true}, nil
}

func handleUnknownState(ctx context.Context, r statemachine.ReconcilerInterface, resource statemachine.StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	er := r.GetEventRecorder()

	log.Info("Handling Unknown state")

	// Update conditions to reflect unknown state
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionUnknown, "StateUnknown", "Resource state cannot be determined")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionUnknown, "StateUnknown", "Resource availability cannot be determined")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionUnknown, "StateUnknown", "Resource progress cannot be determined")

	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		log.Error(err, "Failed to update status in unknown state")
		return kodev1alpha2.PhaseUnknown, ctrl.Result{Requeue: true}, nil
	}

	er.Record(ctx, kode, event.EventTypeWarning, "StateUnknown", "Resource entered unknown state, attempting to determine correct state")

	// Check if resource is being deleted
	if !kode.DeletionTimestamp.IsZero() {
		log.Info("Resource is marked for deletion")
		return kodev1alpha2.PhaseDeleting, ctrl.Result{Requeue: true}, nil
	}

	// Attempt to determine current state based on resource status
	determinedPhase, err := determineResourceState(ctx, r, kode)
	if err != nil {
		log.Error(err, "Failed to determine resource state")
		er.Record(ctx, kode, event.EventTypeWarning, "StateDeterminationFailed", fmt.Sprintf("Failed to determine resource state: %v", err))

		// If we can't determine state, start from Pending
		log.Info("Unable to determine state, falling back to Pending phase")
		kode.Status.Phase = kodev1alpha2.PhasePending

		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update status when falling back to Pending")
			return kodev1alpha2.PhaseUnknown, ctrl.Result{Requeue: true}, nil
		}

		er.Record(ctx, kode, event.EventTypeNormal, "StateResolution", "Unable to determine state, falling back to Pending phase")

		return kodev1alpha2.PhasePending, ctrl.Result{Requeue: true}, nil
	}

	// If we successfully determined a phase
	if determinedPhase != kodev1alpha2.PhaseUnknown {
		log.Info("Determined correct state", "newPhase", determinedPhase)

		kode.Status.Phase = determinedPhase
		if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update status after determining state")
			return kodev1alpha2.PhaseUnknown, ctrl.Result{Requeue: true}, nil
		}

		er.Record(ctx, kode, event.EventTypeNormal, "StateDetermined", fmt.Sprintf("Successfully determined resource state: %s", determinedPhase))

		return determinedPhase, ctrl.Result{Requeue: true}, nil
	}

	// If we still can't determine the state, fall back to Pending
	log.Info("State determination inconclusive, falling back to Pending phase")

	kode.Status.Phase = kodev1alpha2.PhasePending
	if err := kode.UpdateStatus(ctx, r.GetClient()); err != nil {
		log.Error(err, "Failed to update status when falling back to Pending")
		return kodev1alpha2.PhaseUnknown, ctrl.Result{Requeue: true}, nil
	}

	er.Record(ctx, kode, event.EventTypeNormal, "StateFallback", "State determination inconclusive, falling back to Pending phase")

	return kodev1alpha2.PhasePending, ctrl.Result{Requeue: true}, nil
}

// determineResourceState attempts to determine the current state of the resource
func determineResourceState(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (kodev1alpha2.Phase, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	// First check if template exists and can be fetched
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		log.V(1).Info("Failed to fetch template", "error", err)
		return kodev1alpha2.PhasePending, nil
	}

	config := InitKodeResourcesConfig(kode, template)

	// Check for generation mismatch
	if kode.Generation != kode.Status.ObservedGeneration {
		log.V(1).Info("Detected generation mismatch", "current", kode.Generation, "observed", kode.Status.ObservedGeneration)
		return kodev1alpha2.PhaseUpdating, nil
	}

	// Check resource readiness
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		log.V(1).Info("Error checking resource readiness", "error", err)
		return kodev1alpha2.PhaseConfiguring, nil
	}

	if !ready {
		// Resources exist but aren't ready
		return kodev1alpha2.PhaseProvisioning, nil
	}

	// All resources are ready and no updates needed
	return kodev1alpha2.PhaseActive, nil
}

// determineRecoveryPhase decides which phase to transition to for recovery
func determineRecoveryPhase(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) kodev1alpha2.Phase {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)

	// Check if resource exists on the cluster
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		log.V(1).Info("Failed to fetch template, starting from Pending phase")
		return kodev1alpha2.PhasePending
	}

	config := InitKodeResourcesConfig(kode, template)

	// Check existing resources
	ready, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		log.V(1).Info("Error checking resources, starting from Pending phase")
		return kodev1alpha2.PhasePending
	}

	if !ready {
		// Resources exist but aren't ready
		log.V(1).Info("Resources exist but not ready, starting from Provisioning phase")
		return kodev1alpha2.PhaseProvisioning
	}

	// If we have configuration changes
	if kode.Generation != kode.Status.ObservedGeneration {
		log.V(1).Info("Configuration changes detected, starting from Configuring phase")
		return kodev1alpha2.PhaseConfiguring
	}

	// Default to Pending if we can't determine a better phase
	log.V(1).Info("No specific recovery phase determined, defaulting to Pending")
	return kodev1alpha2.PhasePending
}
