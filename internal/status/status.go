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

func (u *defaultStatusUpdater) UpdateStatusKode(ctx context.Context,
	kode *kodev1alpha2.Kode,
	phase kodev1alpha2.KodePhase,
	conditions []metav1.Condition,
	conditionsToRemove []string,
	lastError string,
	lastErrorTime *metav1.Time,
) error {
	// ... existing code ...

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of Kode
		latestKode, err := common.GetLatestKode(ctx, u.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestKode.Status.DeepCopy()

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeAndRemoveConditions(updatedStatus.Conditions, conditions, conditionsToRemove)
		updatedStatus.LastError = &lastError
		updatedStatus.LastErrorTime = lastErrorTime
		updatedStatus.ObservedGeneration = latestKode.Generation
		updatedStatus.Runtime = kode.SetRuntime()

		// Create a patch
		patch := client.MergeFrom(latestKode.DeepCopy())
		latestKode.Status = *updatedStatus

		// Apply the patch
		if err := u.Client.Status().Patch(ctx, latestKode, patch); err != nil {
			u.Log.Error(err, "Failed to update Kode status")
			return err
		}
		return nil
	})
}

func (u *defaultStatusUpdater) UpdateStatusEntryPoint(ctx context.Context,
	entryPoint *kodev1alpha2.EntryPoint,
	phase kodev1alpha2.EntryPointPhase,
	conditions []metav1.Condition,
	conditionsToRemove []string,
	lastError string,
	lastErrorTime *metav1.Time) error {

	log := u.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint))

	log.V(1).Info("Updating EntryPoint status", "Phase", phase)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of EntryPoint
		latestEntry, err := common.GetLatestEntryPoint(ctx, u.Client, entryPoint.Name, entryPoint.Namespace)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestEntry.Status.DeepCopy()

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeAndRemoveConditions(updatedStatus.Conditions, conditions, conditionsToRemove)
		updatedStatus.LastError = &lastError
		updatedStatus.LastErrorTime = lastErrorTime
		updatedStatus.ObservedGeneration = latestEntry.Generation

		// Create a patch
		patch := client.MergeFrom(latestEntry.DeepCopy())
		latestEntry.Status = *updatedStatus

		// Apply the patch
		if err := u.Client.Status().Patch(ctx, latestEntry, patch); err != nil {
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
	for _, condition := range new {
		existingMap[condition.Type] = condition
	}

	// Convert map back to slice
	result := make([]metav1.Condition, 0, len(existingMap))
	for _, condition := range existingMap {
		result = append(result, condition)
	}

	return result
}
