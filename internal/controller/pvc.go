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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensurePVC ensures that the PersistentVolumeClaim exists for the Kode instance
func (r *KodeReconciler) ensurePVC(ctx context.Context, kode *kodev1alpha1.Kode) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("ensurePVC")

	log.Info("Ensuring PVC exists", "Namespace", kode.Namespace, "Name", kode.Name)

	pvc, err := r.getOrCreatePVC(ctx, kode)
	if err != nil {
		return pvc, err
	}

	return pvc, r.updatePVCIfNecessary(ctx, kode, pvc)
}

// getOrCreatePVC gets or creates a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) getOrCreatePVC(ctx context.Context, kode *kodev1alpha1.Kode) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("getOrCreatePVC")

	pvc := r.constructPVC(kode)
	if err := controllerutil.SetControllerReference(kode, pvc, r.Scheme); err != nil {
		return nil, err
	}

	found := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new PVC", "Namespace", pvc.Namespace, "Name", pvc.Name)
			if err := r.Create(ctx, pvc); err != nil {
				return nil, err
			}
			return pvc, nil
		}
		return nil, err
	}

	return found, nil
}

// updatePVCIfNecessary updates the PVC if the desired state is different from the existing state
func (r *KodeReconciler) updatePVCIfNecessary(ctx context.Context, kode *kodev1alpha1.Kode, existingPVC *corev1.PersistentVolumeClaim) error {
	log := r.Log.WithName("updatePVCIfNecessary")
	desiredPVC := r.constructPVC(kode)

	// Only update mutable fields: Resources.Requests
	if !equality.Semantic.DeepEqual(existingPVC.Spec.Resources.Requests, desiredPVC.Spec.Resources.Requests) {
		existingPVC.Spec.Resources.Requests = desiredPVC.Spec.Resources.Requests
		log.Info("Updating existing PVC resources", "Namespace", existingPVC.Namespace, "Name", existingPVC.Name)
		return r.Update(ctx, existingPVC)
	}

	log.Info("PVC is up-to-date", "Namespace", existingPVC.Namespace, "Name", existingPVC.Name)
	return nil
}

// constructPVC constructs a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) constructPVC(kode *kodev1alpha1.Kode) *corev1.PersistentVolumeClaim {
	log := r.Log.WithName("constructPVC")

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PersistentVolumeClaimName,
			Namespace: kode.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: kode.Spec.Storage.AccessModes,
			Resources:   kode.Spec.Storage.Resources,
		},
	}
	if kode.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = kode.Spec.Storage.StorageClassName
	}

	logPVCManifest(log, pvc)

	return pvc
}
