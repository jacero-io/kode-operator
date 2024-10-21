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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

func fetchTemplatesWithRetry(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (*kodev1alpha2.Template, error) {
	var template *kodev1alpha2.Template
	var lastErr error

	backoff := wait.Backoff{
		Steps:    5,                      // Maximum number of retries
		Duration: 100 * time.Millisecond, // Initial backoff duration
		Factor:   2.0,                    // Factor to increase backoff each try
		Jitter:   0.1,                    // Jitter factor
	}

	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
		var err error
		template, err = r.GetTemplateManager().Fetch(ctx, kode.Spec.TemplateRef)
		if err == nil {
			return true, nil // Success
		}

		if errors.IsNotFound(err) {
			log.Info("Template not found, will not retry", "error", err)
			return false, err // Don't retry if not found
		}

		// For other errors, log and retry
		log.Error(err, "Failed to fetch template, will retry")
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
		kode.SetRuntime(runtime, ctx, r.GetClient())
	}

	return template, nil
}

func handleReconcileError(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, err error, message string) (kodev1alpha2.Phase, ctrl.Result, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Error(err, message)

	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ReconciliationFailed", fmt.Sprintf("%s: %v", message, err))
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ReconciliationFailed", "Kode resource is not available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "ReconciliationFailed", "Kode resource is not progressing")

	kode.Status.LastError = common.StringPtr(err.Error())
	kode.Status.LastErrorTime = &metav1.Time{Time: metav1.Now().Time}

	if updateErr := kode.UpdateStatus(ctx, r.GetClient()); updateErr != nil {
		log.Error(updateErr, "Failed to update Kode status after error")
	}

	return kodev1alpha2.PhaseFailed, ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, err
}

func determineCurrentState(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (kodev1alpha2.Phase, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	// Check if the Kode is being deleted
	if !kode.DeletionTimestamp.IsZero() {
		return kodev1alpha2.PhaseDeleting, nil
	}

	// Fetch the template
	template, err := fetchTemplatesWithRetry(ctx, r, kode)
	if err != nil {
		log.Error(err, "Failed to fetch template")
		return kodev1alpha2.PhaseFailed, err
	}
	config := InitKodeResourcesConfig(kode, template)

	// Check if resources exist and are ready
	// resourcesExist, err := r.checkResourcesDeleted(ctx, kode)
	// if err != nil {
	// 	return kodev1alpha2.PhaseFailed, err
	// }

	// if !resourcesExist {
	// 	return kodev1alpha2.PhasePending, nil
	// }

	// Check if all resources are ready
	resourcesReady, err := checkResourcesReady(ctx, r, kode, config)
	if err != nil {
		return kodev1alpha2.PhaseFailed, err
	}

	if !resourcesReady {
		return kodev1alpha2.PhaseConfiguring, nil
	}

	// If everything is set up and ready, consider it active
	log.Info("All resources are ready")
	return kodev1alpha2.PhaseActive, nil
}

func (r *KodeReconciler) updateRetryCount(ctx context.Context, kode *kodev1alpha2.Kode, count int) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latestKode := &kodev1alpha2.Kode{}
		err := r.Client.Get(ctx, client.ObjectKey{Name: kode.Name, Namespace: kode.Namespace}, latestKode)
		if err != nil {
			return err
		}

		latestKode.Status.RetryCount = count

		return latestKode.UpdateStatus(ctx, r.Client)
	})
}

func (r *KodeReconciler) handleGenerationMismatch(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Generation mismatch detected", "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration)

	// Update the ObservedGeneration
	kode.Status.ObservedGeneration = kode.Generation

	// Always transition to Updating phase, except when deleting
	if kode.Status.Phase != kodev1alpha2.PhaseDeleting {
		kode.Status.Phase = kodev1alpha2.PhaseUpdating

		// Set the condition
		kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "GenerationMismatch", "Generation mismatch detected, resource is being updated")
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "GenerationMismatch", "Generation mismatch detected, resource is being updated")
		kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionTrue, "GenerationMismatch", "Generation mismatch detected, resource is being updated")
	}

	// If already in Deleting state, just update the status
	if err := kode.UpdateStatus(ctx, r.Client); err != nil {
		log.Error(err, "Unable to update Kode status")
		// If we fail to update the status, requeue immediately
		return ctrl.Result{Requeue: true}, err
	}

	// If everything is fine, requeue immediately
	return ctrl.Result{Requeue: true}, nil
}
