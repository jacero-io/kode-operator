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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindPodTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterPodTemplate):
		if template.PodTemplateSpec == nil || template.PodTemplateSpec.BaseSharedSpec.EntryPointRef == nil {
			return nil, fmt.Errorf("invalid PodTemplate: missing EntryPointRef")
		}
		entryPointRef = *template.PodTemplateSpec.BaseSharedSpec.EntryPointRef
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
	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint))

	if entryPoint.Status.Phase == newPhase {
		// No transition needed
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Transitioning EntryPoint state", "from", entryPoint.Status.Phase, "to", newPhase)

	// Create a deep copy of the entryPoint object to avoid modifying the cache
	entryPointCopy := entryPoint.DeepCopy()

	// Update the phase
	entryPointCopy.Status.Phase = newPhase

	// Perform any additional actions based on the new state
	var requeueAfter time.Duration
	switch newPhase {
	case kodev1alpha2.EntryPointPhasePending:
		if err := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPointCopy, kodev1alpha2.EntryPointPhasePending, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "Pending",
				Message: "EntryPoint resource is pending configuration.",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "Pending",
				Message: "EntryPoint resource is pending configuration.",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionTrue,
				Reason:  "Pending",
				Message: "EntryPoint resource is pending configuration.",
			},
		}, nil, "", nil); err != nil {
			log.Error(err, "Failed to update status for Pending state")
			return ctrl.Result{Requeue: true}, err
		}

	case kodev1alpha2.EntryPointPhaseConfiguring:
		removeConditions := []string{"Pending"}
		if err := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPointCopy, kodev1alpha2.EntryPointPhaseConfiguring, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "Configuring",
				Message: "EntryPoint resource is being configured.",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "Configuring",
				Message: "EntryPoint resource is being configured.",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionTrue,
				Reason:  "Configuring",
				Message: "EntryPoint resource is being configured.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Configuring state")
			return ctrl.Result{Requeue: true}, err
		}

		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointConfiguring, "EntryPoint is being configured"); err != nil {
			log.Error(err, "Failed to record EntryPoint configuring event")
		}

	case kodev1alpha2.EntryPointPhaseProvisioning:
		removeConditions := []string{"Configuring", "Pending"}
		if err := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPointCopy, kodev1alpha2.EntryPointPhaseProvisioning, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "Provisioning",
				Message: "EntryPoint resource is being provisioned.",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "Provisioning",
				Message: "EntryPoint resource is being provisioned.",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionTrue,
				Reason:  "Provisioning",
				Message: "EntryPoint resource is being provisioned.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Provisioning state")
			return ctrl.Result{Requeue: true}, err
		}

		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointProvisioning, "EntryPoint is being provisioned"); err != nil {
			log.Error(err, "Failed to record EntryPoint provisioning event")
		}

	case kodev1alpha2.EntryPointPhaseActive:
		removeConditions := []string{"Configuring", "Pending", "Provisioning"}
		if err := r.StatusUpdater.UpdateStatusEntryPoint(ctx, entryPointCopy, kodev1alpha2.EntryPointPhaseActive, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionTrue,
				Reason:  "EnteredActiveState",
				Message: "EntryPoint is now active and ready.",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionTrue,
				Reason:  "EnteredActiveState",
				Message: "EntryPoint is now active and available.",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionFalse,
				Reason:  "EnteredActiveState",
				Message: "EntryPoint is now active.",
			},
		}, removeConditions, "", nil); err != nil {
			log.Error(err, "Failed to update status for Active state")
			return ctrl.Result{Requeue: true}, err
		}

		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointActive, "EntryPoint is now active"); err != nil {
			log.Error(err, "Failed to record EntryPoint active event")
		}

	case kodev1alpha2.EntryPointPhaseDeleting:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeNormal, events.ReasonEntryPointDeleting, "EntryPoint is being deleted"); err != nil {
			log.Error(err, "Failed to record EntryPoint deleting event")
		}

	case kodev1alpha2.EntryPointPhaseFailed:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeWarning, events.ReasonEntryPointFailed, "EntryPoint has entered Failed state"); err != nil {
			log.Error(err, "Failed to record EntryPoint failed event")
		}
		requeueAfter = 5 * time.Minute

	case kodev1alpha2.EntryPointPhaseUnknown:
		if err := r.EventManager.Record(ctx, entryPoint, events.EventTypeWarning, events.ReasonEntryPointUnknown, "EntryPoint has entered Unknown state"); err != nil {
			log.Error(err, "Failed to record EntryPoint unknown state event")
		}
		requeueAfter = 1 * time.Minute
	}

	// Requeue to handle the new state
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{Requeue: true}, nil
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
