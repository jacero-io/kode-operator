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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

func (r *KodeReconciler) updatePortStatus(ctx context.Context, kode *kodev1alpha2.Kode, template *kodev1alpha2.Template) error {
	// Fetch the latest version of Kode
	latestKode, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
	if err != nil {
		return err
	}

	// Update the Kode port
	err = latestKode.UpdateKodePort(ctx, r.Client, template.Port)
	if err != nil {
		return err
	}
	return nil
}

func (r *KodeReconciler) fetchTemplatesWithRetry(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.Template, error) {
	var template *kodev1alpha2.Template
	var lastErr error

	backoff := wait.Backoff{
		Steps:    5,                      // Maximum number of retries
		Duration: 100 * time.Millisecond, // Initial backoff duration
		Factor:   2.0,                    // Factor to increase backoff each try
		Jitter:   0.1,                    // Jitter factor
	}

	retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
		var err error
		template, err = r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
		if err == nil {
			return true, nil // Success
		}

		if errors.IsNotFound(err) {
			r.Log.Info("Template not found, will not retry", "error", err)
			return false, err // Don't retry if not found
		}

		// For other errors, log and retry
		r.Log.Error(err, "Failed to fetch template, will retry")
		lastErr = err
		return false, nil // Retry
	})

	if retryErr != nil {
		if errors.IsNotFound(retryErr) {
			return nil, fmt.Errorf("template not found after retries: %w", retryErr)
		}
		return nil, fmt.Errorf("failed to fetch template after retries: %w", lastErr)
	}

	return template, nil
}

func (r *KodeReconciler) updateObservedGeneration(ctx context.Context, kode *kodev1alpha2.Kode) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		if latest.Status.ObservedGeneration != latest.Generation {
			r.Log.Info("Updating ObservedGeneration",
				"Name", latest.Name,
				"Namespace", latest.Namespace,
				"OldObservedGeneration", latest.Status.ObservedGeneration,
				"NewObservedGeneration", latest.Generation)

			latest.Status.ObservedGeneration = latest.Generation
			return r.Client.Status().Update(ctx, latest)
		}

		r.Log.V(1).Info("ObservedGeneration already up to date",
			"Name", latest.Name,
			"Namespace", latest.Namespace,
			"Generation", latest.Generation)

		return nil
	})
}

func (r *KodeReconciler) checkResourcesExist(ctx context.Context, kode *kodev1alpha2.Kode) (bool, error) {
	// Check for StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: kode.Namespace, Name: kode.Name}, statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check for Service
	service := &corev1.Service{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: kode.Namespace, Name: kode.GetServiceName()}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check for PVC if applicable
	if kode.Spec.Storage != nil && kode.Spec.Storage.ExistingVolumeClaim == nil {
		pvc := &corev1.PersistentVolumeClaim{}
		err = r.Client.Get(ctx, client.ObjectKey{Namespace: kode.Namespace, Name: kode.GetPVCName()}, pvc)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
	}

	return true, nil
}

func (r *KodeReconciler) determineCurrentState(ctx context.Context, kode *kodev1alpha2.Kode) (kodev1alpha2.KodePhase, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	// Check if the Kode is being deleted
	if !kode.DeletionTimestamp.IsZero() {
		return kodev1alpha2.KodePhaseDeleting, nil
	}

	// Fetch the template
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return kodev1alpha2.KodePhaseFailed, err
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if resources exist and are ready
	resourcesExist, err := r.checkResourcesExist(ctx, kode)
	if err != nil {
		return kodev1alpha2.KodePhaseFailed, err
	}

	if !resourcesExist {
		return kodev1alpha2.KodePhasePending, nil
	}

	resourcesReady, err := r.checkPodResources(ctx, kode, config)
	if err != nil {
		return kodev1alpha2.KodePhaseFailed, err
	}

	if !resourcesReady {
		return kodev1alpha2.KodePhaseConfiguring, nil
	}

	// If everything is set up and ready, consider it active
	return kodev1alpha2.KodePhaseActive, nil
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
