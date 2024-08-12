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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *EntryPointReconciler) ensureResources(ctx context.Context, config *common.EntryPointResourceConfig) error {
	log := r.Log.WithName("ResoruceEnsurer").WithValues("entry", common.ObjectKeyFromConfig(config.CommonConfig))

	// Fetch the latest EntryPoint object
	entry, err := common.GetLatestEntryPoint(ctx, r.Client, config.CommonConfig.Name, config.CommonConfig.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get EntryPoint: %w", err)
	}

	// Check if the resource has changed since last reconciliation
	if entry.Generation != entry.Status.ObservedGeneration {
		log.Info("Resource has changed, ensuring all resources", "Generation", entry.Generation, "ObservedGeneration", entry.Status.ObservedGeneration)
		// Update status depending on the current phase
		if entry.Status.Phase != kodev1alpha1.EntryPointPhasePending {
			// If the EntryPoint is already Active, update the status to Pending
			if entry.Status.Phase == kodev1alpha1.EntryPointPhaseActive {
				if err := r.updateStatus(ctx, entry, kodev1alpha1.EntryPointPhasePending, []metav1.Condition{}, nil); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else { // If the EntryPoint is not Active, update the status to Creating
				if err := r.updateStatus(ctx, entry, kodev1alpha1.EntryPointPhaseCreating, []metav1.Condition{}, nil); err != nil {
					log.Error(err, "Failed to update status to Creating")
					return err
				}
			}
		}

		// Update status depending on the current phase
		if entry.Status.Phase != kodev1alpha1.EntryPointPhasePending {
			// If the EntryPoint is Active, update the status to Pending
			if entry.Status.Phase == kodev1alpha1.EntryPointPhaseActive {
				if err := r.updateStatus(ctx, entry, kodev1alpha1.EntryPointPhasePending, []metav1.Condition{}, nil); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else { // If the EntryPoint is not Active, update the status to Created
				if err := r.updateStatus(ctx, entry, kodev1alpha1.EntryPointPhaseCreating, []metav1.Condition{}, nil); err != nil {
					log.Error(err, "Failed to update status to Creating")
					return err
				}
			}
		}
	}

	return nil
}
