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
	"reflect"
	"time"

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
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/pkg/constant"
)

func (r *KodeReconciler) transitionTo(ctx context.Context, kode *kodev1alpha2.Kode, newPhase kodev1alpha2.KodePhase) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode.Status.Phase == newPhase {
		// TODO: Implement level-triggered updates instead of edge-triggered with requeue delay
		// No transition needed, requeue after a delay
		// return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Create a copy of the Kode to avoid mutating the original
	kodeCopy := kode.DeepCopy()

	// Update the phase
	kodeCopy.SetPhase(newPhase)

	log.V(1).Info("Transitioning Kode state", "from", kode.Status.Phase, "to", newPhase)

	// Perform any additional actions based on the new state
	var requeueAfter time.Duration
	switch newPhase {
	case kodev1alpha2.KodePhasePending:
		// Empty case, requeue immediately

	case kodev1alpha2.KodePhaseConfiguring:
		err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeConfiguring, "Kode is being configured")
		if err != nil {
			log.Error(err, "Failed to record Kode configuring event")
		}

	case kodev1alpha2.KodePhaseProvisioning:
		err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeProvisioning, "Kode is being provisioned")
		if err != nil {
			log.Error(err, "Failed to record Kode provisioning event")
		}

	case kodev1alpha2.KodePhaseActive:
		if err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeActive, "Kode is now active"); err != nil {
			log.Error(err, "Failed to record Kode active event")
		}

	case kodev1alpha2.KodePhaseSuspending:
		if err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeSuspended, "Kode is being suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}

	case kodev1alpha2.KodePhaseSuspended:
		if err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeSuspended, "Kode has been suspended"); err != nil {
			log.Error(err, "Failed to record Kode suspended event")
		}

	case kodev1alpha2.KodePhaseResuming:
		err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeResuming, "Kode is resuming")
		if err != nil {
			log.Error(err, "Failed to record Kode resuming event")
		}

	case kodev1alpha2.KodePhaseDeleting:
		err := r.Event.Record(ctx, kode, event.EventTypeNormal, event.ReasonKodeDeleting, "Kode is being deleted")
		if err != nil {
			log.Error(err, "Failed to record Kode deleting event")
		}

	case kodev1alpha2.KodePhaseFailed:
		if err := r.Event.Record(ctx, kode, event.EventTypeWarning, event.ReasonKodeFailed, "Kode has entered Failed state"); err != nil {
			log.Error(err, "Failed to record Kode failed event")
		}

		// Requeue with a delay to avoid spinning
		requeueAfter = 1 * time.Minute

	case kodev1alpha2.KodePhaseUnknown:
		// Record unknown state event
		if err := r.Event.Record(ctx, kode, event.EventTypeWarning, event.ReasonKodeUnknown, "Kode has entered Unknown state"); err != nil {
			log.Error(err, "Failed to record Kode unknown state event")
		}

		// Requeue with a delay to avoid spinning
		requeueAfter = 1 * time.Minute

	}

	// Update the phase in the status if it has changed
	if !reflect.DeepEqual(kode.Status, kodeCopy.Status) {
		log.V(1).Info("Updating Kode status", "from", kode.Status.Phase, "to", newPhase)
		err := kodeCopy.UpdateStatus(ctx, r.Client)
		if err != nil {
			log.Error(err, "Failed to update status")
			// If we fail to update the status, requeue
			return ctrl.Result{Requeue: true}, err
		}
		log.V(1).Info("Kode status updated", "Phase", kodeCopy.Status.Phase)
	}

	// Update the Kode resource
	kode.Status = kodeCopy.Status

	// Requeue to handle the new state
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *KodeReconciler) updateRetryCount(ctx context.Context, kode *kodev1alpha2.Kode, count int) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latestKode := &kodev1alpha2.Kode{}
		err := r.Client.Get(ctx, client.ObjectKey{Name: kode.Name, Namespace: kode.Namespace}, latestKode)
		if err != nil {
			return err
		}

		latestKode.Status.RetryCount = count
		return r.Client.Status().Update(ctx, latestKode)
	})
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
		template, err = r.Template.Fetch(ctx, kode.Spec.TemplateRef)
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

	// Update Runtime
	if kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindContainer || kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindClusterContainer {
		runtime := kodev1alpha2.Runtime{
			Runtime: kodev1alpha2.RuntimeContainer,
			Type:    template.ContainerTemplateSpec.Runtime,
		}
		kode.SetRuntime(runtime, ctx, r.Client)
	}

	return template, nil
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
	resourcesExist, err := r.checkResourcesDeleted(ctx, kode)
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

func (r *KodeReconciler) handleGenerationMismatch(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Generation mismatch detected", "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration)

	// Set the condition
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "Reconfiguring", "Kode resource is being reconfigured")

	// Update the ObservedGeneration
	kode.Status.ObservedGeneration = kode.Generation

	// Always transition to Configuring state, except when deleting
	if kode.Status.Phase != kodev1alpha2.KodePhaseDeleting {
		return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseConfiguring)
	}

	// If already in Deleting state, just update the status
	err := kode.UpdateStatus(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to update status")
		// If we fail to update the status, requeue
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *KodeReconciler) handleReconcileError(ctx context.Context, kode *kodev1alpha2.Kode, err error, message string) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Error(err, message)
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ReconciliationFailed", fmt.Sprintf("%s: %v", message, err))
	kode.Status.LastError = common.StringPtr(err.Error())
	kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
}
