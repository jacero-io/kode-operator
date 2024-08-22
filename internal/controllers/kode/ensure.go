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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *KodeReconciler) ensureResources(ctx context.Context, config *common.KodeResourceConfig) error {
	log := r.Log.WithName("ResoruceEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	// Fetch the latest Kode object
	kode, err := common.GetLatestKode(ctx, r.Client, config.CommonConfig.Name, config.CommonConfig.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get Kode: %w", err)
	}

	// Check if the resource has changed since last reconciliation
	if kode.Generation != kode.Status.ObservedGeneration {
		log.Info("Resource has changed, ensuring all resources", "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration)
		// Update status depending on the current phase
		if kode.Status.Phase != kodev1alpha2.KodePhasePending {
			// If the Kode is already Active, update the status to Pending
			if kode.Status.Phase == kodev1alpha2.KodePhaseActive {
				if err := r.updatePhasePending(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else { // If the Kode is not Active, update the status to Creating
				if err := r.updatePhaseCreating(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Creating")
					return err
				}
			}
		}

		// Ensure Secret
		log.V(1).Info("Ensuring Secret", "ConfigNil", config == nil, "CredentialsNil", config.Credentials == nil)
		if err := r.ensureSecret(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure Secret")
			r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
				{
					Type:    string(common.ConditionTypeError),
					Status:  metav1.ConditionTrue,
					Reason:  "SecretCreation",
					Message: fmt.Sprintf("Failed to create secret: %s", err.Error()),
				},
				{
					Type:    string(common.ConditionTypeReady),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceNotReady",
					Message: "Kode resource is not ready due to failed secret creation",
				},
				{
					Type:    string(common.ConditionTypeAvailable),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceUnavailable",
					Message: "Kode resource is not available due to failed secret creation",
				},
				{
					Type:    string(common.ConditionTypeProgressing),
					Status:  metav1.ConditionFalse,
					Reason:  "ProgressHalted",
					Message: "Progress halted due to failed failed secret creation",
				},
			})

			err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
			if err != nil {
				log.Error(err, "Failed to record event")
			}

			return err
		}

		// Ensure StatefulSet
		if err := r.ensureStatefulSet(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure StatefulSet")
			r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
				{
					Type:    string(common.ConditionTypeError),
					Status:  metav1.ConditionTrue,
					Reason:  "StatefulSetCreation",
					Message: fmt.Sprintf("Failed to create statefulset: %s", err.Error()),
				},
				{
					Type:    string(common.ConditionTypeReady),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceNotReady",
					Message: "Kode resource is not ready due to failed statefulset creation",
				},
				{
					Type:    string(common.ConditionTypeAvailable),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceUnavailable",
					Message: "Kode resource is not available due to failed statefulset creation",
				},
				{
					Type:    string(common.ConditionTypeProgressing),
					Status:  metav1.ConditionFalse,
					Reason:  "ProgressHalted",
					Message: "Progress halted due to failed failed statefulset creation",
				},
			})

			err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
			if err != nil {
				log.Error(err, "Failed to record event")
			}

			return err
		}

		// Ensure Service
		if err := r.ensureService(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure Service")
			r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
				{
					Type:    string(common.ConditionTypeError),
					Status:  metav1.ConditionTrue,
					Reason:  "ServiceCreation",
					Message: fmt.Sprintf("Failed to create service: %s", err.Error()),
				},
				{
					Type:    string(common.ConditionTypeReady),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceNotReady",
					Message: "Kode resource is not ready due to failed service creation",
				},
				{
					Type:    string(common.ConditionTypeAvailable),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceUnavailable",
					Message: "Kode resource is not available due to failed service creation",
				},
				{
					Type:    string(common.ConditionTypeProgressing),
					Status:  metav1.ConditionFalse,
					Reason:  "ProgressHalted",
					Message: "Progress halted due to failed failed service creation",
				},
			})

			err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
			if err != nil {
				log.Error(err, "Failed to record event")
			}

			return err
		}

		// Ensure PVC if storage is specified
		if !kode.Spec.Storage.IsEmpty() {
			if err := r.ensurePVC(ctx, config, kode); err != nil {
				log.Error(err, "Failed to ensure PVC")
				r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
					{
						Type:    string(common.ConditionTypeError),
						Status:  metav1.ConditionTrue,
						Reason:  "PvcCreation",
						Message: fmt.Sprintf("Failed to create pvc: %s", err.Error()),
					},
					{
						Type:    string(common.ConditionTypeReady),
						Status:  metav1.ConditionFalse,
						Reason:  "ResourceNotReady",
						Message: "Kode resource is not ready due to failed pvc creation",
					},
					{
						Type:    string(common.ConditionTypeAvailable),
						Status:  metav1.ConditionFalse,
						Reason:  "ResourceUnavailable",
						Message: "Kode resource is not available due to failed pvc creation",
					},
					{
						Type:    string(common.ConditionTypeProgressing),
						Status:  metav1.ConditionFalse,
						Reason:  "ProgressHalted",
						Message: "Progress halted due to failed failed pvc creation",
					},
				})

				err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
				if err != nil {
					log.Error(err, "Failed to record event")
				}

				return err
			}
		}

		// Update status depending on the current phase
		if kode.Status.Phase != kodev1alpha2.KodePhasePending {
			// If the Kode is Active, update the status to Pending
			if kode.Status.Phase == kodev1alpha2.KodePhaseActive {
				if err := r.updatePhasePending(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else { // If the Kode is not Active, update the status to Created
				if err := r.updatePhaseCreated(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Created")
					return err
				}
			}
		}
	}

	return nil
}
