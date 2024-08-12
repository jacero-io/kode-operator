// internal/controllers/kode/utils.go

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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *KodeReconciler) updateStatus(ctx context.Context, kode *kodev1alpha1.Kode, phase kodev1alpha1.KodePhase, conditions []metav1.Condition, err error) error {
	switch phase {
	case kodev1alpha1.KodePhaseCreating:
		if err := r.updatePhaseCreating(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseCreated:
		if err := r.updatePhaseCreated(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseFailed:
		if err := r.updatePhaseFailed(ctx, kode, err, conditions); err != nil {
			return err
		}
	case kodev1alpha1.KodePhasePending:
		if err := r.updatePhasePending(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseActive:
		if err := r.updatePhaseActive(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseInactive:
		if err := r.updatePhaseInactive(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseRecycling:
		if err := r.updatePhaseRecycling(ctx, kode); err != nil {
			return err
		}
	case kodev1alpha1.KodePhaseRecycled:
		if err := r.updatePhaseRecycled(ctx, kode); err != nil {
			return err
		}
	}
	return nil
}

// updatePhaseCreating updates the Kode status to indicate that the resources are being created.
func (r *KodeReconciler) updatePhaseCreating(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseCreating
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Kode resources are not ready",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

// updatePhaseCreated updates the Kode status to indicate that the resources have been created.
func (r *KodeReconciler) updatePhaseCreated(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseCreated
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotReady",
			Message:            "Kode resources are not ready",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

// updatePhaseFailed updates the Kode status to indicate that the resource has failed.
func (r *KodeReconciler) updatePhaseFailed(ctx context.Context, kode *kodev1alpha1.Kode, err error, additionalConditions []metav1.Condition) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseFailed
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

	// // If the resource was in the process of being created when it failed
	// // We need to fetch the current status to check this
	// kode := &kodev1alpha1.Kode{}
	// if err := u.Get(ctx, types.NamespacedName{Name: config.KodeName, Namespace: config.KodeNamespace}, kode); err != nil {
	// 	log.Error(err, "Failed to fetch Kode for status check")
	// } else if kode.Status.Phase == kodev1alpha1.KodePhaseCreating {
	// 	conditions = append(conditions, metav1.Condition{
	// 		Type:			   string(common.ConditionTypeReady),
	// 		Status:			   metav1.ConditionFalse,
	// 		Reason:			   "ResourceNotReady",
	// 		Message:		   "Kode resource is not ready due to failure",
	// 		LastTransitionTime: now,
	// 	})
	// }

	// Add additional conditions
	conditions = append(conditions, additionalConditions...)

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, errorMessage, &now)
}

// updatePhasePending updates the Kode status to indicate that the resources are pending.
func (r *KodeReconciler) updatePhasePending(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhasePending
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

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

// updatePhaseActive updates the Kode status to indicate that the resources are active.
func (r *KodeReconciler) updatePhaseActive(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseActive
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

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

// updatePhaseInactive updates the Kode status to indicate that the resources are inactive.
func (r *KodeReconciler) updatePhaseInactive(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseInactive
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Kode resource is not ready due to inactive state",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Kode resource is inactive and not available",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceInactive",
			Message:            "Kode resource is not progressing while inactive",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

// updatePhaseRecycling updates the Kode status to indicate that the resources are being recycled.
func (r *KodeReconciler) updatePhaseRecycling(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseRecycling
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeProgressing),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceRecycling",
			Message:            "Kode resource is being recycled",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycling",
			Message:            "Kode resource is not available during recycling",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycling",
			Message:            "Kode resource is not ready while being recycled",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}

func (r *KodeReconciler) updatePhaseRecycled(ctx context.Context, kode *kodev1alpha1.Kode) error {
	now := metav1.NewTime(time.Now())
	phase := kodev1alpha1.KodePhaseRecycled
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycled",
			Message:            "Recycled Kode resource is not ready for use",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceRecycled",
			Message:            "Recycled Kode resource is not available",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateStatusKode(ctx, kode, phase, conditions, "", nil)
}
