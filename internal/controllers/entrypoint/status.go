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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// updatePhaseFailed updates the EntryPoint status to indicate that the resource has failed.
func (r *EntryPointReconciler) updatePhaseFailed(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, err error, additionalConditions []metav1.Condition) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseFailed
	errorMessage := err.Error()

	// Create a map to store conditions, keyed by Type
	conditionMap := make(map[string]metav1.Condition)

	// Define default conditions
	defaultConditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeError),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceFailed",
			Message:            fmt.Sprintf("EntryPoint resource failed: %s", errorMessage),
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotReady",
			Message:            "Resource is not ready due to failure",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceUnavailable",
			Message:            "Resource is not available due to failure",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ProgressHalted",
			Message:            "Progress halted due to resource failure",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
	}

	// Add default conditions to the map
	for _, condition := range defaultConditions {
		conditionMap[condition.Type] = condition
	}

	// Override or add additional conditions
	for _, condition := range additionalConditions {
		// Ensure the additional condition has the correct generation and timestamp
		condition.ObservedGeneration = entryPoint.Generation
		condition.LastTransitionTime = now
		conditionMap[condition.Type] = condition
	}

	// Convert the map back to a slice
	finalConditions := make([]metav1.Condition, 0, len(conditionMap))
	for _, condition := range conditionMap {
		finalConditions = append(finalConditions, condition)
	}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPoint, phase, finalConditions, nil, errorMessage, &now)
}

// updatePhaseActive updates the Kode status to indicate that the resources are active.
func (r EntryPointReconciler) updatePhaseActive(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseActive
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceReady",
			Message:            "Resource is ready and active",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceAvailable",
			Message:            "Resource is available",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceStable",
			Message:            "Resource is stable and not progressing",
			LastTransitionTime: now,
			ObservedGeneration: entryPoint.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPoint, phase, conditions, conditionsToRemove, "", nil)
}

func (r *EntryPointReconciler) updateKodeStatus(ctx context.Context, kode *kodev1alpha2.Kode, phase kodev1alpha2.KodePhase, additionalConditions []metav1.Condition, kodeUrl kodev1alpha2.KodeUrl, err error) error {
	log := r.Log.WithValues("kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace})
	log.V(1).Info("Updating Kode status", "phase", phase, "kodeUrl", kodeUrl)

	// Prepare the status update
	var lastError string
	var lastErrorTime *metav1.Time
	now := metav1.Now()

	if err != nil {
		lastError = err.Error()
		lastErrorTime = &now
	}

	// Create a map to store conditions, keyed by Type
	conditionMap := make(map[string]metav1.Condition)

	defaultConditions := []metav1.Condition{}

	// Add default conditions to the map
	for _, condition := range defaultConditions {
		conditionMap[condition.Type] = condition
	}

	// Override or add additional conditions
	for _, condition := range additionalConditions {
		// Ensure the additional condition has the correct generation and timestamp
		condition.ObservedGeneration = kode.Generation
		condition.LastTransitionTime = now
		conditionMap[condition.Type] = condition
	}

	// Convert the map back to a slice
	finalConditions := make([]metav1.Condition, 0, len(conditionMap))
	for _, condition := range conditionMap {
		finalConditions = append(finalConditions, condition)
	}

	// Use the StatusUpdater to update the Kode status
	updateErr := r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, finalConditions, nil, lastError, lastErrorTime)
	if updateErr != nil {
		log.Error(updateErr, "Failed to update Kode status")
		return fmt.Errorf("failed to update Kode status: %w", updateErr)
	}

	// If the update was successful and a new URL is provided, update the Kode URL
	if kodeUrl != "" {
		updateUrlErr := kode.UpdateKodeUrl(ctx, r.Client, kodeUrl)
		if updateUrlErr != nil {
			log.Error(updateUrlErr, "Failed to update Kode URL")
			return fmt.Errorf("failed to update Kode URL: %w", updateUrlErr)
		}
	}

	log.Info("Successfully updated Kode status", "phase", phase, "kodeUrl", kodeUrl)
	return nil
}
