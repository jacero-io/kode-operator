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

// import (
// 	"context"
// 	"fmt"

// 	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
// 	"github.com/jacero-io/kode-operator/internal/common"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// func (r *EntryPointReconciler) ensureResources(ctx context.Context, config *common.EntryPointResourceConfig) error {
// 	log := r.Log.WithName("ResourceEnsurer").WithValues("entry", common.ObjectKeyFromConfig(config.CommonConfig))

// 	// Fetch the latest EntryPoint object
// 	entry, err := common.GetLatestEntryPoint(ctx, r.Client, config.CommonConfig.Name)
// 	if err != nil {
// 		return fmt.Errorf("failed to get EntryPoint: %w", err)
// 	}

// 	// Check if the resource has changed since last reconciliation
// 	if entry.Generation != entry.Status.ObservedGeneration {
// 		log.Info("Resource has changed, ensuring all resources", "Generation", entry.Generation, "ObservedGeneration", entry.Status.ObservedGeneration)

// 		// Update status to Pending or Creating
// 		if err := r.updateStatusForReconciliation(ctx, entry); err != nil {
// 			return err
// 		}

// 		// Ensure HTTPRoute

// 		// Update status to Active
// 		if err := r.updateStatus(ctx, entry, kodev1alpha2.EntryPointPhaseActive, []metav1.Condition{}, nil); err != nil {
// 			log.Error(err, "Failed to update status to Active")
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (r *EntryPointReconciler) updateStatusForReconciliation(ctx context.Context, entry *kodev1alpha2.ClusterEntryPoint) error {
// 	if entry.Status.Phase != kodev1alpha2.EntryPointPhasePending {
// 		if entry.Status.Phase == kodev1alpha2.EntryPointPhaseActive {
// 			if err := r.updateStatus(ctx, entry, kodev1alpha2.EntryPointPhasePending, []metav1.Condition{}, nil); err != nil {
// 				r.Log.Error(err, "Failed to update status to Pending")
// 				return err
// 			}
// 		} else {
// 			if err := r.updateStatus(ctx, entry, kodev1alpha2.EntryPointPhaseCreating, []metav1.Condition{}, nil); err != nil {
// 				r.Log.Error(err, "Failed to update status to Creating")
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }
