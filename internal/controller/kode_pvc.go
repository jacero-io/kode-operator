// internal/controller/kode_pvc.go

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
	"fmt"

	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TODO: Investigate if "if pvc == nil" is really required
// ensurePVC ensures that the PersistentVolumeClaim exists for the Kode instance
func (r *KodeReconciler) ensurePVC(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("PVCEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring PVC")

	// If ExistingVolumeClaim is specified, return nil
	if config.Kode.Spec.Storage.ExistingVolumeClaim != "" {
		log.Info("ExistingVolumeClaim specified, skipping PVC creation", "ExistingVolumeClaim", config.Kode.Spec.Storage.ExistingVolumeClaim)
		return nil
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.PVCName,
			Namespace: config.KodeNamespace,
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
			// PVC exists, update only mutable fields
			pvc.Spec.Resources = constructedPVC.Spec.Resources
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
		} else {
			return fmt.Errorf("failed to get existing PVC: %v", err)
		}

		// Update metadata for both new and existing PVCs
		pvc.ObjectMeta.Labels = constructedPVC.ObjectMeta.Labels
		pvc.ObjectMeta.Annotations = constructedPVC.ObjectMeta.Annotations

		return controllerutil.SetControllerReference(&config.Kode, pvc, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to create or patch PVC: %v", err)
	}

	return nil
}

// // Check if PVC already exists
// existingPVC := &corev1.PersistentVolumeClaim{}
// err = r.Get(ctx, client.ObjectKeyFromObject(pvc), existingPVC)
// if err == nil {
//     // PVC exists, ensure finalizer is present
// 	if !controllerutil.ContainsFinalizer(existingPVC, common.FinalizerName) {
// 		controllerutil.AddFinalizer(existingPVC, common.PVCFinalizerName)
//         if err := r.Update(ctx, existingPVC); err != nil {
//             return fmt.Errorf("failed to update PVC finalizers: %w", err)
//         }
//     }
//     return nil
// }

// constructPVCSpec constructs a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) constructPVCSpec(config *common.KodeResourcesConfig) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("PvcConstructor").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.PVCName,
			Namespace: config.KodeNamespace,
			Labels:    config.Labels,
			// Finalizers: []string{common.PVCFinalizerName},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: config.Kode.Spec.Storage.AccessModes,
			Resources:   config.Kode.Spec.Storage.Resources,
		},
	}

	if config.Kode.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = config.Kode.Spec.Storage.StorageClassName
	}

	log.V(1).Info("PVC object constructed", "PVC", pvc, "Spec", pvc.Spec)

	return pvc, nil
}
