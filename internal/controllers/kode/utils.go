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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
)

func (r *KodeReconciler) transitionTo(ctx context.Context, kode *kodev1alpha2.Kode, newPhase kodev1alpha2.KodePhase) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode.Status.Phase == newPhase {
		// No transition needed
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Transitioning Kode state", "from", kode.Status.Phase, "to", newPhase)

	// Create a deep copy of the kode object to avoid modifying the cache
	kodeCopy := kode.DeepCopy()

	// Update the phase
	kodeCopy.Status.Phase = newPhase

	// Perform any additional actions based on the new state
	var requeueAfter time.Duration
	switch newPhase {

	case kodev1alpha2.KodePhasePending:
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kodeCopy, kodev1alpha2.KodePhasePending, []metav1.Condition{
			{
				Type:    common.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "Pending",
				Message: "Kode resource is pending configuration.",
			},
			{
				Type:    common.ConditionTypeAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "Pending",
				Message: "Kode resource is pending configuration.",
			},
			{
				Type:    common.ConditionTypeProgressing,
				Status:  metav1.ConditionFalse,
				Reason:  "Pending",
				Message: "Kode resource is pending configuration.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for Pending state")
			return ctrl.Result{Requeue: true}, err
		}

		// Trigger immediate reconciliation to start configuration
		// requeueAfter = 1 * time.Millisecond

	case kodev1alpha2.KodePhaseConfiguring:
		removeConditions := []string{"Pending"}
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseConfiguring, []metav1.Condition{
			{
				Type:    common.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "Configuring",
				Message: "Kode resource is being configured.",
			},
			{
				Type:    common.ConditionTypeAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "Configuring",
				Message: "Kode resource is being configured.",
			},
			{
				Type:    common.ConditionTypeProgressing,
				Status:  metav1.ConditionFalse,
				Reason:  "Configuring",
				Message: "Kode resource is being configured.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Configuring state")
			return ctrl.Result{Requeue: true}, err
		}

		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeConfiguring, "Kode is being configured")
		if err != nil {
			log.Error(err, "Failed to record Kode configuring event")
		}

		// Trigger immediate reconciliation to start provisioning
		// requeueAfter = 1 * time.Millisecond

	case kodev1alpha2.KodePhaseProvisioning:

		removeConditions := []string{"Configuring", "Pending"}
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseProvisioning, []metav1.Condition{
			{
				Type:    common.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "Provisioning",
				Message: "Kode resource is being provisioned.",
			},
			{
				Type:    common.ConditionTypeAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "Provisioning",
				Message: "Kode resource is being provisioned.",
			},
			{
				Type:    common.ConditionTypeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "Provisioning",
				Message: "Kode resource is being provisioned.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Provisioning state")
			return ctrl.Result{Requeue: true}, err
		}

		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeProvisioning, "Kode is being provisioned")
		if err != nil {
			log.Error(err, "Failed to record Kode provisioning event")
		}

		// Trigger immediate reconciliation to transition to active state
		// requeueAfter = 1 * time.Millisecond

	case kodev1alpha2.KodePhaseActive:
		removeConditions := []string{"Configuring", "Pending", "ResourcesNotReady"}
		if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseActive, []metav1.Condition{
			{
				Type:    common.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "EnteredActiveState",
				Message: "Kode is now active and ready.",
			},
			{
				Type:    common.ConditionTypeAvailable,
				Status:  metav1.ConditionTrue,
				Reason:  "EnteredActiveState",
				Message: "Kode is now active and available.",
			},
			{
				Type:    common.ConditionTypeProgressing,
				Status:  metav1.ConditionFalse,
				Reason:  "EnteredActiveState",
				Message: "Kode is now active.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Active state")
			return ctrl.Result{Requeue: true}, err
		}

		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeActive, "Kode is now active"); err != nil {
			log.Error(err, "Failed to record Kode active event")
		}

	case kodev1alpha2.KodePhaseSuspending:
		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeSuspended, "Kode is being suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}

	case kodev1alpha2.KodePhaseSuspended:
		if err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeSuspended, "Kode has been suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}

	case kodev1alpha2.KodePhaseResuming:
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeResuming, "Kode is resuming")
		if err != nil {
			log.Error(err, "Failed to record Kode resuming event")
		}

	case kodev1alpha2.KodePhaseDeleting:
		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonKodeDeleting, "Kode is being deleted")
		if err != nil {
			log.Error(err, "Failed to record Kode deleting event")
		}

	case kodev1alpha2.KodePhaseFailed:
		// Record failure event
		if err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonKodeFailed, "Kode has entered Failed state"); err != nil {
			log.Error(err, "Failed to record Kode failed event")
		}
		// Requeue with a delay to avoid spinning
		requeueAfter = time.Minute

	case kodev1alpha2.KodePhaseUnknown:
		// Record unknown state event
		if err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonKodeUnknown, "Kode has entered Unknown state"); err != nil {
			log.Error(err, "Failed to record Kode unknown state event")
		}
		// Requeue with a delay to avoid spinning
		requeueAfter = 10 * time.Second

	}

	// Requeue to handle the new state
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *KodeReconciler) updateRetryCount(ctx context.Context, kode *kodev1alpha2.Kode, count int) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		latest.Status.RetryCount = count
		return r.Client.Status().Update(ctx, latest)
	})
}

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
