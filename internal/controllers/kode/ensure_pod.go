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
	"k8s.io/apimachinery/pkg/types"
)

func (r *KodeReconciler) ensurePodResources(ctx context.Context, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.Log.WithValues("kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace})
	log.Info("Ensuring resources")

	// Fetch the latest Kode object
	latestKode, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get Kode: %w", err)
	}

	// Ensure the Secret
	if config.Credentials != nil {
		if err := r.ensureSecret(ctx, latestKode, config); err != nil {
			log.Error(err, "Failed to ensure Secret")
			return err
		}
	}

	// Ensure the Service
	if err := r.ensureService(ctx, latestKode, config); err != nil {
		log.Error(err, "Failed to ensure Service")
		return err
	}

	// Ensure the PersistentVolumeClaim
	if err := r.ensurePersistentVolumeClaim(ctx, latestKode, config); err != nil {
		log.Error(err, "Failed to ensure PersistentVolumeClaim")
		return err
	}

	// Ensure the StatefulSet
	if err := r.ensureStatefulSet(ctx, latestKode, config); err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		return err
	}

	return nil
}
