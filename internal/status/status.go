// internal/status/status_updater.go

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

package status

import (
	"context"
	"sort"
	"time"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

// StatusUpdater defines the interface for updating the status of a Kode and entry point resources.
type StatusUpdater interface {
	UpdateStatusKode(ctx context.Context, kode *kodev1alpha2.Kode, phase kodev1alpha2.KodePhase, conditions []metav1.Condition, conditionsToRemove []string, lastError string, lastErrorTime *metav1.Time) error
	UpdateStatusEntryPoint(ctx context.Context, entry *kodev1alpha2.EntryPoint, phase kodev1alpha2.EntryPointPhase, conditions []metav1.Condition, conditionsToRemove []string, lastError string, lastErrorTime *metav1.Time) error
}

type defaultStatusUpdater struct {
	Client client.Client
	Log    logr.Logger
}

func NewDefaultStatusUpdater(client client.Client, log logr.Logger) StatusUpdater {
	return &defaultStatusUpdater{
		Client: client,
		Log:    log,
	}
}

func (u *defaultStatusUpdater) UpdateStatusKode(ctx context.Context, kode *kodev1alpha2.Kode, phase kodev1alpha2.KodePhase, conditions []metav1.Condition, conditionsToRemove []string, lastError string, lastErrorTime *metav1.Time) error {
	log := u.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	log.V(1).Info("Updating Kode status", "Phase", phase)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of Kode
		latestKode, err := common.GetLatestKode(ctx, u.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestKode.Status.DeepCopy()

		// Ensure conditions have all required fields
		conditions = ensureConditionFields(conditions, latestKode.Generation)

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeAndRemoveConditions(updatedStatus.Conditions, conditions, conditionsToRemove)
		if lastError != "" {
			updatedStatus.LastError = &lastError
		}
		if lastErrorTime != nil {
			updatedStatus.LastErrorTime = lastErrorTime
		}
		updatedStatus.Runtime = kode.SetRuntime()

		// Create a patch
		patch := client.MergeFrom(latestKode.DeepCopy())
		latestKode.Status = *updatedStatus

		// Apply the patch
		if err := u.Client.Status().Patch(ctx, latestKode, patch); err != nil {
			log.Error(err, "Failed to update Kode status")
			return err
		}
		return nil
	})
}

func (u *defaultStatusUpdater) UpdateStatusEntryPoint(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, phase kodev1alpha2.EntryPointPhase, conditions []metav1.Condition, conditionsToRemove []string, lastError string, lastErrorTime *metav1.Time) error {

	log := u.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint))

	log.V(1).Info("Updating EntryPoint status", "Phase", phase)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of EntryPoint
		latestEntryPoint, err := common.GetLatestEntryPoint(ctx, u.Client, entryPoint.Name, entryPoint.Namespace)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestEntryPoint.Status.DeepCopy()

		// Ensure conditions have all required fields
		conditions = ensureConditionFields(conditions, latestEntryPoint.Generation)

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeAndRemoveConditions(updatedStatus.Conditions, conditions, conditionsToRemove)
		if lastError != "" {
			updatedStatus.LastError = &lastError
		}
		if lastErrorTime != nil {
			updatedStatus.LastErrorTime = lastErrorTime
		}

		// Create a patch
		patch := client.MergeFrom(latestEntryPoint.DeepCopy())
		latestEntryPoint.Status = *updatedStatus

		// Apply the patch
		if err := u.Client.Status().Patch(ctx, latestEntryPoint, patch); err != nil {
			log.Error(err, "Failed to update EntryPoint status")
			return err
		}
		return nil
	})
}

func mergeAndRemoveConditions(existing, new []metav1.Condition, toRemove []string) []metav1.Condition {
	// Create a map for quick lookup of conditions to remove
	removeMap := make(map[string]bool)
	for _, condType := range toRemove {
		removeMap[condType] = true
	}

	// Create a map of existing conditions
	existingMap := make(map[string]metav1.Condition)
	for _, condition := range existing {
		if !removeMap[condition.Type] {
			existingMap[condition.Type] = condition
		}
	}

	// Merge new conditions
	now := metav1.NewTime(time.Now())
	for _, condition := range new {
		if existingCond, exists := existingMap[condition.Type]; exists {
			// Update LastTransitionTime only if the status has changed
			if existingCond.Status != condition.Status {
				condition.LastTransitionTime = now
			} else {
				condition.LastTransitionTime = existingCond.LastTransitionTime
			}
		} else {
			condition.LastTransitionTime = now
		}
		existingMap[condition.Type] = condition
	}

	// Convert map back to slice
	result := make([]metav1.Condition, 0, len(existingMap))
	for _, condition := range existingMap {
		result = append(result, condition)
	}

	// Sort conditions by type for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Type < result[j].Type
	})

	return result
}

func ensureConditionFields(conditions []metav1.Condition, generation int64) []metav1.Condition {
	now := metav1.Now()
	for i := range conditions {
		if conditions[i].LastTransitionTime.IsZero() {
			conditions[i].LastTransitionTime = now
		}
		if conditions[i].ObservedGeneration == 0 {
			conditions[i].ObservedGeneration = generation
		}
	}
	return conditions
}
