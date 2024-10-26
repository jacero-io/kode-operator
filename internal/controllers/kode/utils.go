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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

func fetchTemplatesWithRetry(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (*kodev1alpha2.Template, error) {
	log := r.GetLog().WithValues(
		"kode", client.ObjectKeyFromObject(kode),
		"templateRef", kode.Spec.TemplateRef,
	)

	log.V(1).Info("Starting template fetch",
		"templateKind", kode.Spec.TemplateRef.Kind,
		"templateName", kode.Spec.TemplateRef.Name,
		"templateNamespace", kode.Spec.TemplateRef.Namespace)

	template, err := r.GetTemplateManager().Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Template not found",
				"templateKind", kode.Spec.TemplateRef.Kind,
				"templateName", kode.Spec.TemplateRef.Name,
				"templateNamespace", kode.Spec.TemplateRef.Namespace)
			return nil, fmt.Errorf("template not found: %w", err)
		}
		log.Error(err, "Failed to fetch template")
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	log.V(1).Info("Successfully fetched template",
		"templateKind", template.Kind,
		"port", template.Port)

	// Log runtime update attempt
	log.V(1).Info("Checking if runtime update is needed",
		"templateKind", kode.Spec.TemplateRef.Kind,
		"isContainer", kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindContainer,
		"isClusterContainer", kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindClusterContainer)

	// Update Runtime
	if kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindContainer ||
		kodev1alpha2.TemplateKind(kode.Spec.TemplateRef.Kind) == kodev1alpha2.TemplateKindClusterContainer {
		runtime := kodev1alpha2.Runtime{
			Runtime: kodev1alpha2.RuntimeContainer,
			Type:    template.ContainerTemplateSpec.Runtime,
		}
		log.V(1).Info("Updating runtime",
			"runtime", runtime.Runtime,
			"type", runtime.Type)

		if err := kode.SetRuntime(runtime, ctx, r.GetClient()); err != nil {
			log.Error(err, "Failed to update runtime",
				"runtime", runtime.Runtime,
				"type", runtime.Type)
			return template, fmt.Errorf("failed to update runtime: %w", err)
		}
		log.V(1).Info("Successfully updated runtime")
	}

	log.V(1).Info("Template fetch completed successfully",
		"templateKind", template.Kind,
		"templateName", template.Name,
		"port", template.Port)

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
