// internal/controller/controller_utils.go

/*
Copyright emil@jacero.se 2024.

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

package controller

import (
	"context"
	"time"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KodeReconciler) updateKodeStatusWithSuccess(ctx context.Context, config *common.KodeResourcesConfig) error {

	now := metav1.NewTime(time.Now())
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeCreated),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourcesCreated",
			Message:            "Kode resources created successfully",
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionTrue,
			Reason:             "ReconciliationSucceeded",
			Message:            "Kode resource reconciled successfully",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateKodeStatus(ctx, config, kodev1alpha1.KodePhaseActive, conditions, "", nil)
}

func (r *KodeReconciler) updateKodeStatusWithError(ctx context.Context, config *common.KodeResourcesConfig, err error) error {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(&config.Kode))
	log.Error(err, "Reconciliation error")

	now := metav1.NewTime(time.Now())
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeCreated),
			Status:             metav1.ConditionFalse,
			Reason:             "ResourceCreationFailed",
			Message:            err.Error(),
			LastTransitionTime: now,
		},
		{
			Type:               string(common.ConditionTypeReady),
			Status:             metav1.ConditionFalse,
			Reason:             "ReconciliationError",
			Message:            err.Error(),
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateKodeStatus(ctx, config, kodev1alpha1.KodePhaseFailed, conditions, err.Error(), &now)
}

func (r *KodeReconciler) updateKodeStatusInactive(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(&config.Kode))
	log.Info("Resource marked as inactive")

	now := metav1.NewTime(time.Now())
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeInactive),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceInactive",
			Message:            "Kode resource has been marked as inactive",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateKodeStatus(ctx, config, kodev1alpha1.KodePhaseInactive, conditions, "", nil)
}

func (r *KodeReconciler) updateKodeStatusRecycled(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(&config.Kode))
	log.Info("Resource recycled")

	now := metav1.NewTime(time.Now())
	conditions := []metav1.Condition{
		{
			Type:               string(common.ConditionTypeRecycled),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourceRecycled",
			Message:            "Kode resource has been recycled",
			LastTransitionTime: now,
		},
	}

	return r.StatusUpdater.UpdateKodeStatus(ctx, config, kodev1alpha1.KodePhaseRecycling, conditions, "", nil)
}

func (r *KodeReconciler) clearErrorStatus(ctx context.Context, config *common.KodeResourcesConfig) error {
	config.Kode.Status.LastError = ""
	config.Kode.Status.LastErrorTime = nil
	meta.RemoveStatusCondition(&config.Kode.Status.Conditions, common.ConditionTypeError)
	if err := r.Status().Update(ctx, &config.Kode); err != nil {
		r.Log.WithValues("kode", client.ObjectKeyFromObject(&config.Kode)).Error(err, "Failed to clear error status")
		return err
	}
	return nil
}

func (r *KodeReconciler) GetCurrentTime() metav1.Time {
	return metav1.NewTime(time.Now())
}
