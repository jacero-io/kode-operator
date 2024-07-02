/*
Copyright 2024.

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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
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
	var err error

	log.Info("Ensuring PVC exists")

	// If ExistingVolumeClaim is specified, log and skip creation
	if config.Kode.Spec.Storage.ExistingVolumeClaim != "" {
		log.Info("Using existing PVC", "ExistingVolumeClaim", config.Kode.Spec.Storage.ExistingVolumeClaim)
		return nil
	}

	// Construct PVC only if ExistingVolumeClaim is not specified
	pvc := r.constructPVC(config)
	if pvc == nil {
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

	if err := controllerutil.SetControllerReference(&config.Kode, pvc, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	err = r.ResourceManager.Ensure(ctx, pvc)
	if err != nil {
		log.Error(err, "Failed to ensure PVC")
		return fmt.Errorf("failed to ensure PVC: %w", err)
	}

	log.Info("Successfully ensured PVC")
	return nil
}

// constructPVC constructs a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) constructPVC(config *common.KodeResourcesConfig) *corev1.PersistentVolumeClaim {
	// If ExistingVolumeClaim is specified, return nil
	if config.Kode.Spec.Storage.ExistingVolumeClaim != "" {
		return nil
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.GetPVCName(&config.Kode),
			Namespace: config.Kode.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&config.Kode, kodev1alpha1.GroupVersion.WithKind("Kode")),
			},
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

	return pvc
}
