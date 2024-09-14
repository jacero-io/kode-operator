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

package entrypoint

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
)

func (r *EntryPointReconciler) findEntryPointForKode(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.EntryPoint, error) {
	// Fetch the template object
	template, err := r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	var entryPointRef kodev1alpha2.CrossNamespaceObjectReference
	switch template.Kind {
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindContainerTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterContainerTemplate):
		if template.ContainerTemplateSpec == nil || template.ContainerTemplateSpec.BaseSharedSpec.EntryPointRef == nil {
			return nil, fmt.Errorf("invalid ContainerTemplate: missing EntryPointRef")
		}
		entryPointRef = *template.ContainerTemplateSpec.BaseSharedSpec.EntryPointRef
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindTofuTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterTofuTemplate):
		if template.TofuTemplateSpec == nil || template.TofuTemplateSpec.BaseSharedSpec.EntryPointRef == nil {
			return nil, fmt.Errorf("invalid TofuTemplate: missing EntryPointRef")
		}
		entryPointRef = *template.TofuTemplateSpec.BaseSharedSpec.EntryPointRef
	default:
		return nil, fmt.Errorf("unknown template kind: %s", template.Kind)
	}

	// Fetch the latest EntryPoint object
	latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, string(entryPointRef.Name), string(*entryPointRef.Namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get EntryPoint: %w", err)
	}

	return latestEntryPoint, nil
}

func (r *EntryPointReconciler) transitionTo(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, newPhase kodev1alpha2.EntryPointPhase) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})

	if entryPoint.Status.Phase == newPhase {
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
	}

	log.V(1).Info("Transitioning EntryPoint state", "from", entryPoint.Status.Phase, "to", newPhase)

	entryPoint.Status.Phase = newPhase

	switch newPhase {
	case kodev1alpha2.EntryPointPhasePending:
		// Empty case - does nothing

	case kodev1alpha2.EntryPointPhaseConfiguring:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointConfiguring, "EntryPoint is being configured"); err != nil {
			log.Error(err, "Failed to record EntryPoint configuring event")
		}

	case kodev1alpha2.EntryPointPhaseProvisioning:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointProvisioning, "EntryPoint is being provisioned"); err != nil {
			log.Error(err, "Failed to record EntryPoint provisioning event")
		}

	case kodev1alpha2.EntryPointPhaseActive:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointActive, "EntryPoint is now active"); err != nil {
			log.Error(err, "Failed to record EntryPoint active event")
		}

	case kodev1alpha2.EntryPointPhaseFailed:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeWarning, events.ReasonEntryPointFailed, "EntryPoint has entered Failed state"); err != nil {
			log.Error(err, "Failed to record EntryPoint failed event")
		}

	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *EntryPointReconciler) updateStatus(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the EntryPoint
		latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, entryPoint.Name, entryPoint.Namespace)
		if err != nil {
			return err
		}

		// Create a patch
		patch := client.MergeFrom(latestEntryPoint.DeepCopy())

		// Update the status
		latestEntryPoint.Status = entryPoint.Status

		// Apply the patch
		if err := r.Client.Status().Patch(ctx, latestEntryPoint, patch); err != nil {
			r.Log.Error(err, "Failed to update EntryPoint status")
			return err
		}
		return nil
	})
}

func (r *EntryPointReconciler) checkGatewayClassExists(ctx context.Context, gatewayClassName string) error {
	gatewayClass := &gwapiv1.GatewayClass{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: gatewayClassName}, gatewayClass); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("specified GatewayClass '%s' not found", gatewayClassName)
		}
		return fmt.Errorf("failed to check GatewayClass: %w", err)
	}
	return nil
}

func (r *EntryPointReconciler) checkCertificatesExist(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) error {
	for _, certRef := range *entryPoint.Spec.GatewaySpec.CertificateRefs {
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: string(certRef.Name), Namespace: entryPoint.Namespace}, secret); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("certificate Secret '%s' not found", certRef.Name)
			}
			return fmt.Errorf("failed to check certificate Secret: %w", err)
		}
	}
	return nil
}

func (r *EntryPointReconciler) checkBaseDomainAvailability(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) error {
	entryPointList := &kodev1alpha2.EntryPointList{}
	if err := r.Client.List(ctx, entryPointList); err != nil {
		return fmt.Errorf("failed to list EntryPoints: %w", err)
	}

	for _, ep := range entryPointList.Items {
		if ep.Name != entryPoint.Name && ep.Spec.BaseDomain == entryPoint.Spec.BaseDomain {
			return fmt.Errorf("BaseDomain '%s' is already in use by EntryPoint '%s'", entryPoint.Spec.BaseDomain, ep.Name)
		}
	}
	return nil
}
