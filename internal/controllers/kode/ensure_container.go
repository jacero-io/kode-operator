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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
)

func (r *KodeReconciler) ensurePodResources(ctx context.Context, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.Log.WithValues("kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, "phase", kode.Status.Phase)
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

	// Check CSI resize capability before ensuring PVC
	var resizeSupported bool
	if config.KodeSpec.Storage != nil && config.KodeSpec.Storage.ExistingVolumeClaim == nil {
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: latestKode.Namespace}, pvc)
		if err == nil {
			// PVC exists, check resize capability
			resizeSupported, err = r.checkCSIResizeCapability(ctx, pvc)
			if err != nil {
				log.Error(err, "Failed to check CSI resize capability")
				eventMessage := "Failed to check CSI resize capability"
				if recordErr := r.EventManager.Record(ctx, latestKode, events.EventTypeWarning, events.ReasonKodeCSIResizeCapabilityCheckFailed, eventMessage); recordErr != nil {
					log.Error(recordErr, "Failed to record CSI resize check failure event")
				}
				return err
			}
		} else {
			// PVC doesn't exist yet, we'll check after creation
			resizeSupported = false
		}
	}

	// Ensure the PersistentVolumeClaim
	if err := r.ensurePersistentVolumeClaim(ctx, latestKode, config, resizeSupported); err != nil {
		log.Error(err, "Failed to ensure PersistentVolumeClaim")
		return err
	}

	// If PVC was just created, check resize capability now
	if config.KodeSpec.Storage != nil && config.KodeSpec.Storage.ExistingVolumeClaim == nil && !resizeSupported {
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: latestKode.Namespace}, pvc)
		if err != nil {
			log.Error(err, "Failed to get PVC for CSI resize check")
			eventMessage := "Failed to check CSI resize capability after PVC creation"
			if recordErr := r.EventManager.Record(ctx, latestKode, events.EventTypeWarning, events.ReasonKodeCSIResizeCapabilityCheckFailed, eventMessage); recordErr != nil {
				log.Error(recordErr, "Failed to record CSI resize check failure event")
			}
			return err
		}
	}

	// Ensure the StatefulSet
	if err := r.ensureStatefulSet(ctx, latestKode, config); err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		return err
	}

	log.Info("Resources ensured")
	return nil
}
