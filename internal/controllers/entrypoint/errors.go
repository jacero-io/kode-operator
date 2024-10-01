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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/constant"
	"github.com/jacero-io/kode-operator/internal/event"
)

// handleReconcileError updates the Kode status with the error and returns a Result for requeuing
func (r *EntryPointReconciler) handleReconcileError(ctx context.Context, kode *kodev1alpha2.Kode, entryPoint *kodev1alpha2.EntryPoint, err error, message string) (ctrl.Result, error) {
	log := r.Log.WithName("ErrorHandler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)
	log.Error(err, message)

	// Update EntryPoint status if it exists
	if entryPoint != nil {
		entryPointUpdateErr := r.updatePhaseFailed(ctx, entryPoint, err, []metav1.Condition{
			{
				Type:    string(constant.ConditionTypeError),
				Status:  metav1.ConditionTrue,
				Reason:  "ReconciliationFailed",
				Message: fmt.Sprintf("%s: %v", message, err),
			},
			{
				Type:    string(constant.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceNotReady",
				Message: "Resource is not ready due to reconciliation failure",
			},
			{
				Type:    string(constant.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceUnavailable",
				Message: "Resource is not available due to reconciliation failure",
			},
			{
				Type:    string(constant.ConditionTypeProgressing),
				Status:  metav1.ConditionFalse,
				Reason:  "ProgressHalted",
				Message: "Progress halted due to resource reconciliation failure",
			},
		})

		if entryPointUpdateErr != nil {
			log.Error(entryPointUpdateErr, "Failed to update EntryPoint status")
			// Continue to update the Kode status even if EntryPoint update fails
		}
	} else {
		log.V(1).Info("EntryPoint is nil, skipping EntryPoint status update")
	}

	err = r.EventManager.Record(ctx, entryPoint, event.EventTypeWarning, event.ReasonFailed, fmt.Sprintf("Failed to reconcile Entrypoint: %v", err))
	if err != nil {
		log.Error(err, "Failed to record event")
	}

	// Update Kode status
	kodeUpdateErr := r.updateKodeStatus(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{{
		Type:               string(constant.ConditionTypeError),
		Status:             metav1.ConditionTrue,
		Reason:             "ReconciliationFailed",
		Message:            fmt.Sprintf("%s: %v", message, err),
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: kode.Generation,
	}}, "", err)
	if kodeUpdateErr != nil {
		log.Error(kodeUpdateErr, "Failed to update Kode status")
		// If we can't update the status, return both errors
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("primary error: %v, kode status update error: %v", err, kodeUpdateErr)
	}

	err = r.EventManager.Record(ctx, kode, event.EventTypeWarning, event.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
	if err != nil {
		log.Error(err, "Failed to record event")
	}

	// Requeue after a delay
	return ctrl.Result{RequeueAfter: 5 * time.Second}, err
}

func (r *EntryPointReconciler) handleValidationError(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, err error) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Error(err, "EntryPoint validation failed")

	// Update status with validation error
	if updateErr := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPoint, kodev1alpha2.EntryPointPhaseFailed, []metav1.Condition{
		{
			Type:    string(constant.ConditionTypeError),
			Status:  metav1.ConditionTrue,
			Reason:  "ValidationFailed",
			Message: fmt.Sprintf("EntryPoint validation failed: %v", err),
		},
	}, nil, err.Error(), nil); updateErr != nil {
		log.Error(updateErr, "Failed to update EntryPoint status after validation error")
		return ctrl.Result{}, updateErr
	}

	// Record an event for the validation failure
	if eventErr := r.EventManager.Record(ctx, entryPoint, event.EventTypeWarning, event.ReasonEntryPointValidationFailed, fmt.Sprintf("EntryPoint validation failed: %v", err)); eventErr != nil {
		log.Error(eventErr, "Failed to record validation failure event")
	}

	// Requeue after some time to allow for manual intervention
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *EntryPointReconciler) handleResourceError(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, err error, reason string) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})
	log.Error(err, "Resource check failed", "reason", reason)

	// Update status with resource error
	if updateErr := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPoint, kodev1alpha2.EntryPointPhaseFailed, []metav1.Condition{
		{
			Type:    string(constant.ConditionTypeError),
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: fmt.Sprintf("Resource check failed: %v", err),
		},
	}, nil, err.Error(), nil); updateErr != nil {
		log.Error(updateErr, "Failed to update EntryPoint status after resource error")
		return ctrl.Result{}, updateErr
	}

	// Record an event for the resource failure
	if eventErr := r.EventManager.Record(ctx, entryPoint, event.EventTypeWarning, event.ReasonEntryPointResourceCheckFailed, fmt.Sprintf("Resource check failed: %v", err)); eventErr != nil {
		log.Error(eventErr, "Failed to record resource failure event")
	}

	// Requeue after some time to allow for resource creation or manual intervention
	return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
}
