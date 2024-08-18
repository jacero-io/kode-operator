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

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

type EntryPointReconciler struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	ResourceManager resource.ResourceManager
	TemplateManager template.TemplateManager
	CleanupManager  cleanup.CleanupManager
	StatusUpdater   status.StatusUpdater
	Validator       validation.Validator
	EventManager    events.EventManager
}

const (
	RequeueInterval = 250 * time.Millisecond
)

// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete

func (r *EntryPointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("resource", req.NamespacedName)

	var obj client.Object
	if req.Namespace == "" {
		obj = &kodev1alpha2.EntryPoint{}
	} else {
		obj = &kodev1alpha2.Kode{}
	}

	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch v := obj.(type) {
	case *kodev1alpha2.EntryPoint:
		log.V(1).Info("Reconciling EntryPoint", "namespace", v.Namespace, "name", v.Name)
		return ctrl.Result{}, nil
		// return r.reconcileEntryPoint(ctx, v)
	case *kodev1alpha2.Kode:
		log.V(1).Info("Reconciling Kode", "namespace", v.Namespace, "name", v.Name)
		return r.reconcileKode(ctx, v)
	default:
		log.Error(nil, "Unknown resource type", "type", fmt.Sprintf("%T", obj))
		return ctrl.Result{}, nil
	}
}

// func (r *EntryPointReconciler) reconcileEntryPoint(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
//     log := r.Log.WithName("EntryPointReconciler").WithValues(
//         "entryPoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace},
//     )
//
//     log.V(1).Info("Starting reconciliation")
//
// 	// Fetch the latest reconcileEntryPoint object
//     latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, kodev1alpha2.ObjectName(entryPoint.Name), kodev1alpha2.Namespace(entryPoint.Namespace))
//     if err != nil {
//         log.Error(err, "Failed to get latest EntryPoint")
//         return ctrl.Result{RequeueAfter: RequeueInterval}, err
//     }
//
//     log.V(1).Info("Retrieved latest EntryPoint", "generation", latestEntryPoint.Generation)
//
// 	config := InitEntryPointResourcesConfig(latestEntryPoint)
//
// 	// Ensure Gateway
// 	if err := r.ensureGateway(ctx, latestEntryPoint, config); err != nil {
// 		log.Error(err, "Failed to ensure Gateway", "namespace", latestEntryPoint.Namespace, "name", latestEntryPoint.Name)
// 		r.updateStatus(ctx, latestEntryPoint, kodev1alpha2.EntryPointPhaseFailed, []metav1.Condition{{
// 			Type:               "GatewayCreationFailed",
// 			Status:             metav1.ConditionTrue,
// 			Reason:             "GatewayCreationError",
// 			Message:            "Failed to create Gateway",
// 			LastTransitionTime: metav1.Now(),
// 			ObservedGeneration: entryPoint.Generation,
// 		}}, err)
// 		return ctrl.Result{RequeueAfter: RequeueInterval}, err
// 	}
//
// 	// Update status to active
// 	updateErr := r.updateStatus(ctx, latestEntryPoint, kodev1alpha2.EntryPointPhaseActive, []metav1.Condition{{
// 		Type:               "Ready",
// 		Status:             metav1.ConditionTrue,
// 		Reason:             "EntryPointActive",
// 		Message:            "EntryPoint is active",
// 		LastTransitionTime: metav1.Now(),
// 		ObservedGeneration: entryPoint.Generation,
// 	}}, nil)
// 	if updateErr != nil {
// 		log.Error(updateErr, "Failed to update EntryPoint status")
// 		return ctrl.Result{RequeueAfter: RequeueInterval}, updateErr
// 	}
//
//     log.V(1).Info("Reconciliation completed successfully")
//     return ctrl.Result{}, nil
// }

func (r *EntryPointReconciler) reconcileKode(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithName("KodeReconiler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)

	// Find the associated EntryPoint
	log.V(1).Info("Fetching EntryPoint for Kode", "namespace", kode.Namespace, "name", kode.Name)
	entryPoint, err := r.findEntryPointForKode(ctx, kode)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("EntryPoint not found, requeuing", "error", err)
			// Requeue after a short delay
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		log.Error(err, "Unable to find EntryPoint for Kode")
		return r.handleReconcileError(ctx, kode, nil, err, "Unable to find EntryPoint for Kode")
	}

	config := InitEntryPointResourcesConfig(entryPoint)

	// Construct the KodeUrl
	kodeHostname, kodeNamespace, kodeDomain, kodeUrl, err := kode.GenerateKodeUrlForEntryPoint(entryPoint.Spec.RoutingType, entryPoint.Spec.BaseDomain, kode.Name, kode.Namespace, config.Protocol)
	if err != nil {
		return r.handleReconcileError(ctx, kode, entryPoint, err, "Failed to construct Kode URL")
	}
	log.V(1).Info("Constructed Kode URL", "hostname", kodeHostname, "namespace", kodeNamespace, "domain", kodeDomain, "url", kodeUrl)

	// Check Kode Port
	kodePort := kode.GetPort()
	if kodePort == 0 {
		log.Info("Kode port is not set, requeuing")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Ensure HTTPRoute
	created, err := r.ensureHTTPRoutes(ctx, entryPoint, kode, config, kodeHostname, kodeDomain)
	if err != nil {
		log.Error(err, "Failed to ensure HTTPRoute for Kode", "namespace", kode.Namespace, "name", kode.Name)
		return r.handleReconcileError(ctx, kode, entryPoint, err, "Failed to ensure HTTPRoute")
	}

	log.Info("HTTPRoute configuration successful", "created", created, "kodeUrl", kodeUrl)
	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) findEntryPointForKode(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.EntryPoint, error) {
	// Fetch the template object
	template, err := r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	var entryPointRef kodev1alpha2.CrossNamespaceObjectReference
	switch template.Kind {
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindPodTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterPodTemplate):
		entryPointRef = template.PodTemplateSpec.BaseSharedSpec.EntryPointRef
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindTofuTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterTofuTemplate):
		entryPointRef = template.TofuTemplateSpec.BaseSharedSpec.EntryPointRef
	default:
		return nil, fmt.Errorf("unknown template kind: %s", template.Kind)
	}

	// Fetch the latest EntryPoint object
	latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, string(entryPointRef.Name), string(entryPointRef.Namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get EntryPoint: %w", err)
	}

	return latestEntryPoint, nil
}

// handleReconcileError updates the Kode status with the error and returns a Result for requeuing
func (r *EntryPointReconciler) handleReconcileError(ctx context.Context, kode *kodev1alpha2.Kode, entryPoint *kodev1alpha2.EntryPoint, err error, message string) (ctrl.Result, error) {
	log := r.Log.WithName("ErrorHandler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)
	log.Error(err, message)

	// Update EntryPoint status if it exists
	if entryPoint != nil {
		entryPointUpdateErr := r.updatePhaseFailed(ctx, entryPoint, err, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeError),
				Status:  metav1.ConditionTrue,
				Reason:  "ReconciliationFailed",
				Message: fmt.Sprintf("%s: %v", message, err),
			},
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceNotReady",
				Message: "Resource is not ready due to reconciliation failure",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceUnavailable",
				Message: "Resource is not available due to reconciliation failure",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionFalse,
				Reason:  "ProgressHalted",
				Message: "Progress halted due to resource reconciliation failure",
			},
		})

		if entryPointUpdateErr != nil {
			log.Error(entryPointUpdateErr, "Failed to update EntryPoint status")
			// Continue to update the Kode status even if EntryPoint update fails
		}
	} else {
		log.Info("EntryPoint is nil, skipping EntryPoint status update")
	}

	err = r.EventManager.Record(ctx, entryPoint, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Entrypoint: %v", err))
	if err != nil {
		log.Error(err, "Failed to record event")
	}

	// Update Kode status
	kodeUpdateErr := r.updateKodeStatus(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{{
		Type:               string(common.ConditionTypeError),
		Status:             metav1.ConditionTrue,
		Reason:             "ReconciliationFailed",
		Message:            fmt.Sprintf("%s: %v", message, err),
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: kode.Generation,
	}}, "", err)
	if kodeUpdateErr != nil {
		log.Error(kodeUpdateErr, "Failed to update Kode status")
		// If we can't update the status, return both errors
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("primary error: %v, kode status update error: %v", err, kodeUpdateErr)
	}

	err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
	if err != nil {
		log.Error(err, "Failed to record event")
	}

	// Requeue after a delay
	return ctrl.Result{RequeueAfter: 5 * time.Second}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryPointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.EntryPoint{}).
		Owns(&gwapiv1.HTTPRoute{}).
		Owns(&gwapiv1.Gateway{}).
		Watches(&kodev1alpha2.Kode{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
