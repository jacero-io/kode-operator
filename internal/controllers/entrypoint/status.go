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
)

func (r EntryPointReconciler) updateStatus(ctx context.Context, entry *kodev1alpha2.EntryPoint, phase kodev1alpha2.EntryPointPhase, conditions []metav1.Condition, err error) error {
	switch phase {
	case kodev1alpha2.EntryPointPhaseCreating:
		if err := r.updatePhaseCreating(ctx, entry); err != nil {
			return err
		}
	case kodev1alpha2.EntryPointPhaseCreated:
		if err := r.updatePhaseCreated(ctx, entry); err != nil {
			return err
		}
	case kodev1alpha2.EntryPointPhaseFailed:
		if err := r.updatePhaseFailed(ctx, entry, err, conditions); err != nil {
			return err
		}
	case kodev1alpha2.EntryPointPhasePending:
		if err := r.updatePhasePending(ctx, entry); err != nil {
			return err
		}
	case kodev1alpha2.EntryPointPhaseActive:
		if err := r.updatePhaseActive(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// updatePhaseCreating updates the Kode status to indicate that the resources are being created.
func (r EntryPointReconciler) updatePhaseCreating(ctx context.Context, entry *kodev1alpha2.EntryPoint) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseCreating
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Kode resources are not ready",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entry, phase, conditions, "", nil)
}

// updatePhaseCreated updates the Kode status to indicate that the resources have been created.
func (r EntryPointReconciler) updatePhaseCreated(ctx context.Context, entry *kodev1alpha2.EntryPoint) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseCreated
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Kode resources are not ready",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entry, phase, conditions, "", nil)
}

// updatePhaseFailed updates the Kode status to indicate that the resource has failed.
func (r EntryPointReconciler) updatePhaseFailed(ctx context.Context, entry *kodev1alpha2.EntryPoint, err error, additionalConditions []metav1.Condition) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseFailed
	errorMessage := err.Error()

	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeError),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceFailed",
			Message:            fmt.Sprintf("Kode resource failed: %s", errorMessage),
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotReady",
			Message:            "Kode resource is not ready due to failure",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceUnavailable",
			Message:            "Kode resource is not available due to failure",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ProgressHalted",
			Message:            "Progress halted due to resource failure",
			LastTransitionTime: now,
		},
	}

	// Add additional conditions
	conditions = append(conditions, additionalConditions...)

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entry, phase, conditions, errorMessage, &now)
}

// updatePhasePending updates the Kode status to indicate that the resources are pending.
func (r EntryPointReconciler) updatePhasePending(ctx context.Context, entry *kodev1alpha2.EntryPoint) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhasePending
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourcePending",
			Message:            "Kode resource is pending and waiting for dependencies",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotReady",
			Message:            "Kode resource is not ready due to pending state",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceNotAvailable",
			Message:            "Kode resource is not available while in pending state",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entry, phase, conditions, "", nil)
}

// updatePhaseActive updates the Kode status to indicate that the resources are active.
func (r EntryPointReconciler) updatePhaseActive(ctx context.Context, entry *kodev1alpha2.EntryPoint) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha2.EntryPointPhaseActive
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceReady",
			Message:            "Kode resource is ready and active",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceAvailable",
			Message:            "Kode resource is available",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceStable",
			Message:            "Kode resource is stable and not progressing",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusEntryPoint(ctx, entry, phase, conditions, "", nil)
}
