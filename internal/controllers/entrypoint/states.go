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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/constants"
)

func (r *EntryPointReconciler) handlePendingState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Pending state")

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(entryPoint, constants.EntryPointFinalizerName) {
		controllerutil.AddFinalizer(entryPoint, constants.EntryPointFinalizerName)
		if err := r.Client.Update(ctx, entryPoint); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("Added finalizer to EntryPoint resource")
	}

	// Validate EntryPoint configuration
	if err := r.Validator.ValidateEntryPoint(ctx, entryPoint); err != nil {
		log.Error(err, "EntryPoint validation failed")
		return r.handleValidationError(ctx, entryPoint, err)
	}

	// Check if the GatewayClass exists (if specified)
	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.GatewayClassName != nil {
		if err := r.checkGatewayClassExists(ctx, string(*entryPoint.Spec.GatewaySpec.GatewayClassName)); err != nil {
			log.Error(err, "GatewayClass check failed")
			return r.handleResourceError(ctx, entryPoint, err, "GatewayClassNotFound")
		}
	}

	// Check if the specified certificates exist (if HTTPS is configured)
	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.CertificateRefs != nil {
		if err := r.checkCertificatesExist(ctx, entryPoint); err != nil {
			log.Error(err, "Certificate check failed")
			return r.handleResourceError(ctx, entryPoint, err, "CertificateNotFound")
		}
	}

	// Check if the BaseDomain is available (not used by another EntryPoint)
	if err := r.checkBaseDomainAvailability(ctx, entryPoint); err != nil {
		log.Error(err, "BaseDomain availability check failed")
		return r.handleResourceError(ctx, entryPoint, err, "BaseDomainUnavailable")
	}

	// Set conditions for Pending state
	entryPoint.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Pending", "EntryPoint resource is pending configuration")
	entryPoint.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Pending", "EntryPoint resource is not yet available")
	entryPoint.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Configuring", "EntryPoint resource is being configured")

	// Transition to Configuring state
	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
}

func (r *EntryPointReconciler) handleConfiguringState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Configuring state")

	// Initialize EntryPoint resources configuration
	// config := InitEntryPointResourcesConfig(entryPoint)

	// Create or update Gateway
	// gateway, err := r.ensureGateway(ctx, entryPoint, config)
	// if err != nil {
	// 	log.Error(err, "Failed to ensure Gateway")
	// 	return r.handleResourceError(ctx, entryPoint, err, "GatewayCreationFailed")
	// }

	// Transition to Provisioning state
	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseProvisioning)
}

func (r *EntryPointReconciler) handleProvisioningState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Provisioning state")

	// Check Gateway readiness
	gateway, err := r.getGateway(ctx, entryPoint)
	if err != nil {
		log.Error(err, "Failed to get Gateway")
		return r.handleResourceError(ctx, entryPoint, err, "GatewayNotFound")
	}

	gatewayReady, err := r.isGatewayReady(gateway)
	if err != nil {
		log.Error(err, "Failed to check Gateway readiness")
		return r.handleResourceError(ctx, entryPoint, err, "GatewayCheckFailed")
	}

	if !gatewayReady {
		log.Info("Gateway is not ready yet")
		// Requeue to check again later
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Check SecurityPolicy readiness if specified
	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.SecurityPolicySpec != nil {
		if ready, err := r.isSecurityPolicyReady(ctx, entryPoint); err != nil {
			log.Error(err, "Failed to check SecurityPolicy readiness")
			return r.handleResourceError(ctx, entryPoint, err, "SecurityPolicyCheckFailed")
		} else if !ready {
			log.Info("SecurityPolicy is not ready yet")
			// Requeue to check again later
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// Check EnvoyPatchPolicy readiness if specified
	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.EnvoyPatchPolicySpec != nil {
		if ready, err := r.isEnvoyPatchPolicyReady(ctx, entryPoint); err != nil {
			log.Error(err, "Failed to check EnvoyPatchPolicy readiness")
			return r.handleResourceError(ctx, entryPoint, err, "EnvoyPatchPolicyCheckFailed")
		} else if !ready {
			log.Info("EnvoyPatchPolicy is not ready yet")
			// Requeue to check again later
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// All resources are ready, transition to Active state
	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseActive)
}

func (r *EntryPointReconciler) handleActiveState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Active state")

	ready := false

	if !ready {
		log.Info("Resources are not ready yet")
		// Requeue to check again later
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
	}

	// Do not requeue if resources are ready
	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) handleDeletingState(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Info("Handling Deleting state")

	// Set conditions for deleting state
	entryPoint.SetCondition(constants.ConditionTypeReady, metav1.ConditionFalse, "Deleting", "EntryPoint resource is being deleted")
	entryPoint.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionFalse, "Deleting", "EntryPoint resource is being deleted")
	entryPoint.SetCondition(constants.ConditionTypeProgressing, metav1.ConditionTrue, "Deleting", "EntryPoint resource is being deleted")

	// Check if all child resources are deleted
	childResourcesDeleted, err := r.areChildResourcesDeleted(ctx, entryPoint)
	if err != nil {
		log.Error(err, "Failed to check child resources deletion status")
		return ctrl.Result{Requeue: true}, err
	}

	if !childResourcesDeleted {
		log.Info("Child resources are still being deleted, requeuing")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(entryPoint, constants.EntryPointFinalizerName)
	if err := r.Client.Update(ctx, entryPoint); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{Requeue: true}, err
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

func (r *EntryPointReconciler) areChildResourcesDeleted(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (bool, error) {
	// TODO: Implement the logic to check if all child resources (HTTPRoutes, Gateway, etc.) are deleted
	return true, nil
}
