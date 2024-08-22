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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// updatePhaseFailed updates the Kode status to indicate that the resource has failed.
func (r *KodeReconciler) updatePhaseFailed(ctx context.Context, kode *kodev1alpha2.Kode, err error, additionalConditions []metav1.Condition) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseFailed
	errorMessage := err.Error()

	// Create a map to store conditions, keyed by Type
	conditionMap := make(map[string]metav1.Condition)

	// Define default conditions
	defaultConditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeError),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceFailed",
			Message:            fmt.Sprintf("Resource failed: %s", errorMessage),
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotReady",
			Message:            "Resource is not ready due to failure",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceUnavailable",
			Message:            "Resource is not available due to failure",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ProgressHalted",
			Message:            "Progress halted due to resource failure",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

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

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, finalConditions, conditionsToRemove, "", nil)
}

// updatePhaseCreating updates the Kode status to indicate that the resources are being created.
func (r *KodeReconciler) updatePhaseCreating(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseCreating
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Resources are not ready",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

// updatePhaseCreated updates the Kode status to indicate that the resources have been created.
func (r *KodeReconciler) updatePhaseCreated(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseCreated
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Resources are not ready",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

// updatePhasePending updates the Kode status to indicate that the resources are pending.
func (r *KodeReconciler) updatePhasePending(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhasePending
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourcePending",
			Message:            "Resource is pending and waiting for dependencies",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotReady",
			Message:            "Resource is not ready due to pending state",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotAvailable",
			Message:            "Resource is not available while in pending state",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

// updatePhaseActive updates the Kode status to indicate that the resources are active.
func (r *KodeReconciler) updatePhaseActive(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseActive
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceReady",
			Message:            "Resource is ready and active",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceAvailable",
			Message:            "Resource is available",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceStable",
			Message:            "Resource is stable and not progressing",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

// updatePhaseInactive updates the Kode status to indicate that the resources are inactive.
func (r *KodeReconciler) updatePhaseInactive(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseInactive
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Resource is not ready due to inactive state",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Resource is inactive and not available",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Resource is not progressing while inactive",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

// updatePhaseRecycling updates the Kode status to indicate that the resources are being recycled.
func (r *KodeReconciler) updatePhaseRecycling(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseRecycling
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceRecycling",
			Message:            "Resource is being recycled",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycling",
			Message:            "Resource is not available during recycling",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycling",
			Message:            "Resource is not ready while being recycled",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

func (r *KodeReconciler) updatePhaseRecycled(ctx context.Context, kode *kodev1alpha2.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.KodePhaseRecycled
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycled",
			Message:            "Recycled Resource is not ready for use",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycled",
			Message:            "Recycled Resource is not available",
			LastTransitionTime: now,
			ObservedGeneration: kode.Generation,
		},
	}

	conditionsToRemove := []string{string(common.ConditionTypeError)}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, conditionsToRemove, "", nil)
}

func (r *KodeReconciler) updatePortStatus(ctx context.Context, kode *kodev1alpha2.Kode, template *kodev1alpha2.Template) error {
	// Fetch the latest version of Kode
	latestKode, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
	if err != nil {
		return err
	}

	// Update the Kode port
	err = latestKode.UpdateKodePort(ctx, r.Client, template.Port)
	if err != nil {
		return err
	}
	return nil
}
