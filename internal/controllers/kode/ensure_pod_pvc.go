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
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
)

// ensurePVC ensures that the PersistentVolumeClaim exists for the Kode instance
func (r *KodeReconciler) ensurePersistentVolumeClaim(ctx context.Context, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, resizeSupported bool) error {
	log := r.Log.WithName("PVCEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring PVC")

	// If the storage spec is nil, skip PVC creation
	if config.KodeSpec.Storage == nil {
		log.V(1).Info("Storage spec is nil, skipping PVC creation")
		return nil
	}
	// If ExistingVolumeClaim is specified, skip PVC creation
	if config.KodeSpec.Storage.ExistingVolumeClaim != nil {
		log.V(1).Info("ExistingVolumeClaim specified, skipping PVC creation", "ExistingVolumeClaim", config.KodeSpec.Storage.ExistingVolumeClaim)
		eventMessage := fmt.Sprintf("Using existing PVC %s for Kode %s", *config.KodeSpec.Storage.ExistingVolumeClaim, kode.Name)
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeExistingPVCUsed, eventMessage)
		if err != nil {
			log.Error(err, "Failed to record event")
		}
		return nil
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.PVCName,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
	}

	err := r.ResourceManager.CreateOrPatch(ctx, pvc, func() error {
		// Construct the desired PVC spec
		constructedPVC, err := r.constructPVCSpec(config)
		if err != nil {
			return fmt.Errorf("failed to construct PVC spec: %v", err)
		}

		// Get the existing PVC
		existing := &corev1.PersistentVolumeClaim{}
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(pvc), existing)
        if err == nil {
            // PVC exists, update only if resize is supported
            if resizeSupported {
                if !pvc.Spec.Resources.Requests.Storage().Equal(*constructedPVC.Spec.Resources.Requests.Storage()) {
                    pvc.Spec.Resources = constructedPVC.Spec.Resources
                    log.V(1).Info("Updating PVC resources", "NewSize", constructedPVC.Spec.Resources.Requests.Storage())
                    
                    // Record event for resize attempt
					eventMessage := fmt.Sprintf("Attempting to resize PVC %s to %s", pvc.Name, constructedPVC.Spec.Resources.Requests.Storage().String())
					err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodePVCResizeAttempted, eventMessage)
					if err != nil {
						log.Error(err, "Failed to record event")
					}
                }
            } else {
                log.V(1).Info("Resize not supported, skipping PVC resource update")
                
                // Record event for skipped resize
				eventMessage := fmt.Sprintf("Skipped resizing PVC %s, CSI driver does not support volume expansion", pvc.Name)
                err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonKodePVCResizeSkipped, eventMessage)
				if err != nil {
					log.Error(err, "Failed to record event")
				}
            }
            pvc.ObjectMeta.Labels = constructedPVC.ObjectMeta.Labels
            pvc.ObjectMeta.Annotations = constructedPVC.ObjectMeta.Annotations
            // Preserve immutable fields
            pvc.Spec.AccessModes = existing.Spec.AccessModes
            pvc.Spec.VolumeName = existing.Spec.VolumeName
            pvc.Spec.VolumeMode = existing.Spec.VolumeMode
            pvc.Spec.StorageClassName = existing.Spec.StorageClassName
            pvc.Spec.Selector = existing.Spec.Selector
        } else if errors.IsNotFound(err) {
            // PVC doesn't exist, use the entire constructed spec
            pvc.Spec = constructedPVC.Spec
            
            // Record event for PVC creation
			eventMessage := fmt.Sprintf("Created new PVC %s with size %s", pvc.Name, constructedPVC.Spec.Resources.Requests.Storage().String())
			err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodePVCCreated, eventMessage)
			if err != nil {
				log.Error(err, "Failed to record event")
			}
        } else {
            return fmt.Errorf("failed to get existing PVC: %v", err)
        }

		// Update metadata for both new and existing PVCs
		pvc.ObjectMeta.Labels = constructedPVC.ObjectMeta.Labels
		pvc.ObjectMeta.Annotations = constructedPVC.ObjectMeta.Annotations

		return controllerutil.SetControllerReference(kode, pvc, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to create or patch PVC: %v", err)
	}

	return nil
}

// constructPVCSpec constructs a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) constructPVCSpec(config *common.KodeResourceConfig) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("PvcConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	pvc := &corev1.PersistentVolumeClaim{
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: config.KodeSpec.Storage.AccessModes,
			Resources:   *config.KodeSpec.Storage.Resources,
		},
	}

	if config.KodeSpec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = config.KodeSpec.Storage.StorageClassName
	}

	log.V(1).Info("PVC object constructed", "PVC", pvc, "Spec", pvc.Spec)

	return pvc, nil
}

func (r *KodeReconciler) checkCSIResizeCapability(ctx context.Context, pvc *corev1.PersistentVolumeClaim) (bool, error) {
	if pvc.Spec.StorageClassName == nil {
		return false, fmt.Errorf("PVC does not have a StorageClassName specified")
	}

	// Get the StorageClass
	sc := &storagev1.StorageClass{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: *pvc.Spec.StorageClassName}, sc)
	if err != nil {
		return false, fmt.Errorf("failed to get StorageClass: %v", err)
	}

	// Check if volume expansion is allowed
	if sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion {
		return true, nil
	}

	// If AllowVolumeExpansion is not set, we can check for any CSI-specific annotations
	// This is an example and might vary depending on the CSI driver
	if value, exists := sc.Annotations["csi.storage.k8s.io/resizable"]; exists && value == "true" {
		return true, nil
	}

	// If no resize capability is detected, return false
	return false, nil
}
