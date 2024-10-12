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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/pkg/constant"
)

func (r *EntryPointReconciler) handlePendingState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)
	log.Info("Handling Pending state")

	// Validate EntryPoint configuration
	if err := r.Validator.Validate(ctx, entryPoint); err != nil {
		log.Error(err, "EntryPoint validation failed")

		// Update status to reflect validation failure
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ValidationFailed", fmt.Sprintf("EntryPoint validation failed: %v", err))
		entryPoint.Status.Phase = kodev1alpha2.EntryPointPhaseFailed
		entryPoint.Status.LastError = common.StringPtr(err.Error())
		entryPoint.Status.LastErrorTime = &metav1.Time{Time: time.Now()}

		return ctrl.Result{}, nil // Will trigger status update in Reconcile
	}

	// Check if the GatewayClass exists (if specified)
	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.GatewayClassName != nil {
		if err := r.checkGatewayClassExists(ctx, string(*entryPoint.Spec.GatewaySpec.GatewayClassName)); err != nil {
			log.Error(err, "GatewayClass check failed")

			// Update status to reflect resource error
			entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "GatewayClassNotFound", fmt.Sprintf("GatewayClass not found: %v", err))
			entryPoint.Status.Phase = kodev1alpha2.EntryPointPhaseFailed
			entryPoint.Status.LastError = common.StringPtr(err.Error())
			entryPoint.Status.LastErrorTime = &metav1.Time{Time: time.Now()}

			return ctrl.Result{}, nil
		}
	}

	// Set conditions for Pending state
	entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Pending", "EntryPoint resource is pending configuration")
	entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Pending", "EntryPoint resource is not yet available")
	entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "EntryPoint resource is being configured")

	// Remove any error conditions if present
	entryPoint.DeleteCondition(constant.ConditionTypeError)

	// Clear any previous error
	entryPoint.Status.LastError = nil
	entryPoint.Status.LastErrorTime = nil

	// Transition to Configuring state
	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
}

func (r *EntryPointReconciler) handleConfiguringState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)
	log.Info("Handling Configuring state")

	// Initialize EntryPoint resources configuration
	config := InitEntryPointResourcesConfig(entryPoint)

	// Ensure all necessary resources are created
	if err := r.ensureGatewayResources(ctx, entryPoint, config); err != nil {
		log.Error(err, "Failed to ensure gateway resources")
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ResourceCreationFailed", fmt.Sprintf("Failed to create resources: %v", err))
		entryPoint.Status.LastError = common.StringPtr(err.Error())
		entryPoint.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		entryPoint.Status.Phase = kodev1alpha2.EntryPointPhaseFailed
		return ctrl.Result{}, nil
	}

	// Update ObservedGeneration if it's out of sync
	if entryPoint.Generation != entryPoint.Status.ObservedGeneration {
		entryPoint.Status.ObservedGeneration = entryPoint.Generation
		log.V(1).Info("Updated ObservedGeneration", "Generation", entryPoint.Generation, "ObservedGeneration", entryPoint.Status.ObservedGeneration)
	}

	// Update conditions
	entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Configuring", "EntryPoint resources are being configured")
	entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Configuring", "EntryPoint is not yet available")
	entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "EntryPoint resources are being configured")

	// Clear any previous error
	entryPoint.Status.LastError = nil
	entryPoint.Status.LastErrorTime = nil

	// Transition to Provisioning state
	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseProvisioning)
}

func (r *EntryPointReconciler) handleProvisioningState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)
	log.Info("Handling Provisioning state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if entryPoint.Generation != entryPoint.Status.ObservedGeneration {
		entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "EntryPoint resource is being reconfigured")
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
	}

	// Initialize EntryPoint resources configuration
	config := InitEntryPointResourcesConfig(entryPoint)

	// Check if all resources are ready
	ready, err := r.checkGatewayResources(ctx, entryPoint, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ResourceCheckFailed", fmt.Sprintf("Failed to check resource readiness: %v", err))
		entryPoint.Status.LastError = common.StringPtr(err.Error())
		entryPoint.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		entryPoint.Status.Phase = kodev1alpha2.EntryPointPhaseFailed
		return ctrl.Result{}, nil
	}

	if ready {
		log.Info("Resources are ready, transitioning to Active state")

		// Set conditions for ready state
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All EntryPoint resources are ready")
		entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ResourcesReady", "EntryPoint is waiting for the URL to be available")
		entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "ResourcesReady", "EntryPoint resources are fully provisioned")

		// Clear any previous error
		entryPoint.Status.LastError = nil
		entryPoint.Status.LastErrorTime = nil

		// Transition to Active state
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseActive)
	}

	// Resources are not ready yet, update conditions and requeue
	entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Provisioning", "EntryPoint resources are still being provisioned")
	entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Provisioning", "EntryPoint is not yet available")
	entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Provisioning", "EntryPoint resources are being provisioned")

	// Requeue to check again
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *EntryPointReconciler) handleActiveState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)
	log.Info("Handling Active state")

	// When the observed generation is not equal to the generation, it needs to be reconfigured
	if entryPoint.Generation != entryPoint.Status.ObservedGeneration {
		entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "EntryPoint resource is being reconfigured")
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
	}

	// Initialize EntryPoint resources configuration
	config := InitEntryPointResourcesConfig(entryPoint)

	// Check if resources are still ready
	ready, err := r.checkGatewayResources(ctx, entryPoint, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ResourceCheckFailed", fmt.Sprintf("Failed to check resource readiness: %v", err))
		entryPoint.Status.LastError = common.StringPtr(err.Error())
		entryPoint.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		entryPoint.Status.Phase = kodev1alpha2.EntryPointPhaseFailed
		return ctrl.Result{}, nil
	}

	if !ready {
		log.Info("Resources not ready, transitioning back to Configuring state")
		entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ResourcesNotReady", "EntryPoint resources are no longer ready")
		entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ResourcesNotReady", "EntryPoint is no longer available")
		entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "EntryPoint is being reconfigured")
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
	}

	// Update conditions to ensure they reflect the current state
	entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "ResourcesReady", "All EntryPoint resources are ready")
	entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "EntryPoint is available")
	entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "Stable", "EntryPoint is stable and not progressing")

	// Clear any previous error
	entryPoint.Status.LastError = nil
	entryPoint.Status.LastErrorTime = nil

	// No state transition needed, return without error
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

func (r *EntryPointReconciler) handleDeletingState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)
	log.Info("Handling Deleting state")

	// Set conditions for deleting state
	entryPoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "Deleting", "EntryPoint resource is being deleted")
	entryPoint.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "Deleting", "EntryPoint resource is being deleted")
	entryPoint.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Deleting", "EntryPoint resource is being deleted")

	// Check if all child resources are deleted
	log.V(1).Info("Checking if child resources are deleted")
	childResourcesDeleted, err := r.checkResourcesDeleted(ctx, entryPoint)
	if err != nil {
		log.Error(err, "Failed to check child resources deletion status")
		return ctrl.Result{Requeue: true}, err
	}

	if !childResourcesDeleted {
		if entryPoint.Status.DeletionCycle == 0 {
			entryPoint.Status.DeletionCycle = 1
			cleanupResource := NewEntryPointCleanupResource(entryPoint)
			result, err := r.CleanupManager.Cleanup(ctx, cleanupResource)
			if err != nil {
				log.Error(err, "Failed to initiate cleanup")
				entryPoint.SetCondition(constant.ConditionTypeError, metav1.ConditionTrue, "CleanupFailed", fmt.Sprintf("Failed to initiate cleanup: %v", err))
				return result, err
			}
			return result, nil
		}

		log.Info("Child resources still exist")

		log.V(1).Info("Incrementing deletion cycle")
		entryPoint.Status.DeletionCycle++

		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	log.V(1).Info("All child resources are deleted")

	// Remove finalizer
	if controllerutil.ContainsFinalizer(entryPoint, constant.EntryPointFinalizerName) {
		log.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(entryPoint, constant.EntryPointFinalizerName)
		if err := r.Client.Update(ctx, entryPoint); err != nil {
			log.Error(err, "Failed to remove finalizer")
			entryPoint.SetCondition(constant.ConditionTypeError, metav1.ConditionTrue, "FinalizerRemovalFailed", fmt.Sprintf("Failed to remove finalizer: %v", err))
			return ctrl.Result{Requeue: true}, err
		}
	}

	log.Info("EntryPoint resource deletion complete")
	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) handleFailedState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Failed state")

	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) handleUnknownState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Unknown state")

	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) checkResourcesDeleted(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (bool, error) {
	// TODO: Implement the logic to check if all child resources (HTTPRoutes, Gateway, etc.) are deleted
	return true, nil
}
