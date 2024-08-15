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
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultStatusUpdater struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultStatusUpdater(client client.Client, log logr.Logger) StatusUpdater {
	return &defaultStatusUpdater{
		client: client,
		log:    log,
	}
}

func (u *defaultStatusUpdater) UpdateStatusKode(ctx context.Context,
	kode *kodev1alpha2.Kode,
	phase kodev1alpha2.KodePhase,
	newConditions []metav1.Condition,
	lastError string,
	lastErrorTime *metav1.Time) error {

	log := u.log.WithValues("kode", client.ObjectKeyFromObject(kode))

	log.V(1).Info("Updating Kode status", "Phase", phase)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of Kode
		latestKode, err := common.GetLatestKode(ctx, u.client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestKode.Status.DeepCopy()

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeConditions(updatedStatus.Conditions, newConditions)
		updatedStatus.LastError = lastError
		updatedStatus.LastErrorTime = lastErrorTime
		updatedStatus.ObservedGeneration = latestKode.Generation

		// Create a patch
		patch := client.MergeFrom(latestKode.DeepCopy())
		latestKode.Status = *updatedStatus

		// Apply the patch
		if err := u.client.Status().Patch(ctx, latestKode, patch); err != nil {
			log.Error(err, "Failed to update Kode status")
			return err
		}
		return nil
	})
}

func (u *defaultStatusUpdater) UpdateStatusEntryPoint(ctx context.Context,
	entry *kodev1alpha2.ClusterEntryPoint,
	phase kodev1alpha2.EntryPointPhase,
	conditions []metav1.Condition,
	lastError string,
	lastErrorTime *metav1.Time) error {

	log := u.log.WithValues("entrypoint", client.ObjectKeyFromObject(entry))

	log.V(1).Info("Updating EntryPoint status", "Phase", phase)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of EntryPoint
		latestEntry, err := common.GetLatestEntryPoint(ctx, u.client, entry.Name)
		if err != nil {
			return err
		}

		// Create a copy of the latest status
		updatedStatus := latestEntry.Status.DeepCopy()

		// Update only the fields we want to change
		updatedStatus.Phase = phase
		updatedStatus.Conditions = mergeConditions(updatedStatus.Conditions, conditions)
		updatedStatus.LastError = lastError
		updatedStatus.LastErrorTime = lastErrorTime
		updatedStatus.ObservedGeneration = latestEntry.Generation

		// Create a patch
		patch := client.MergeFrom(latestEntry.DeepCopy())
		latestEntry.Status = *updatedStatus

		// Apply the patch
		if err := u.client.Status().Patch(ctx, latestEntry, patch); err != nil {
			log.Error(err, "Failed to update EntryPoint status")
			return err
		}
		return nil
	})
}

func mergeConditions(existing, new []metav1.Condition) []metav1.Condition {
	merged := make([]metav1.Condition, 0, len(existing)+len(new))
	existingMap := make(map[string]metav1.Condition)

	for _, condition := range existing {
		existingMap[condition.Type] = condition
	}

	for _, condition := range new {
		if existing, ok := existingMap[condition.Type]; ok {
			if existing.Status != condition.Status ||
				existing.Reason != condition.Reason ||
				existing.Message != condition.Message {
				merged = append(merged, condition)
			} else {
				merged = append(merged, existing)
			}
			delete(existingMap, condition.Type)
		} else {
			merged = append(merged, condition)
		}
	}

	for _, condition := range existingMap {
		merged = append(merged, condition)
	}

	return merged
}
