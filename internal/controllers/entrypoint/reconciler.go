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
	"reflect"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/constants"
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
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update

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
		// Skip reconciliation of EntryPoints because it is not yet implemented
		log.V(1).Info("Skipping reconciliation of EntryPoint", "namespace", v.Namespace, "name", v.Name)
		return ctrl.Result{}, nil
		// log.V(1).Info("Reconciling EntryPoint", "namespace", v.Namespace, "name", v.Name)
		// return r.reconcileEntryPoint(ctx, v)
	case *kodev1alpha2.Kode:
		log.V(1).Info("Reconciling Kode", "namespace", v.Namespace, "name", v.Name)
		return r.reconcileKode(ctx, v)
	default:
		log.Error(nil, "Unknown resource type", "type", fmt.Sprintf("%T", obj))
		return ctrl.Result{}, nil
	}
}

func (r *EntryPointReconciler) reconcileEntryPoint(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})

	log.V(1).Info("Fetched EntryPoint resource", "Name", entryPoint.Name, "Namespace", entryPoint.Namespace, "Generation", entryPoint.Generation, "ObservedGeneration", entryPoint.Status.ObservedGeneration, "Phase", entryPoint.Status.Phase)

	// **Add finalizer if not present**
	if !controllerutil.ContainsFinalizer(entryPoint, constants.EntryPointFinalizerName) {
		controllerutil.AddFinalizer(entryPoint, constants.EntryPointFinalizerName)
		if err := r.Client.Update(ctx, entryPoint); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		log.Info("Added finalizer to EntryPoint resource")
		// Requeue to ensure the updated resource is processed
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle state transition
	var result ctrl.Result
	var err error

	// Transition to Deleting state if deletion timestamp is set and not already in deleting state
	if !entryPoint.DeletionTimestamp.IsZero() && entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseDeleting {
		result, err = r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseDeleting)
		return result, err // Early return after transition
	} else if entryPoint.Generation != entryPoint.Status.ObservedGeneration && entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseDeleting && entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseConfiguring { // Transition to Configuring state if generation mismatch
		log.Info("Generation mismatch, transitioning to Configuring state")
		result, err = r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseConfiguring)
		return result, err // Early return after transition
	}

	// Transition to Pending state if no phase is set
	if entryPoint.Status.Phase == "" {
		log.Info("Transitioning to Pending state")
		result, err := r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhasePending)
		return result, err // Early return after transition
	}

	// Reset retry count if we're not in a failed state
	if entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseFailed && entryPoint.Status.RetryCount > 0 {
		if err := r.updateRetryCount(ctx, entryPoint, 0); err != nil {
			log.Error(err, "Failed to reset retry count")
			return ctrl.Result{Requeue: true}, err
		}
	}

	switch entryPoint.Status.Phase {
	case kodev1alpha2.EntryPointPhasePending:
		result, err = r.handlePendingState(ctx, entryPoint)
	case kodev1alpha2.EntryPointPhaseConfiguring:
		result, err = r.handleConfiguringState(ctx, entryPoint)
	case kodev1alpha2.EntryPointPhaseProvisioning:
		result, err = r.handleProvisioningState(ctx, entryPoint)
	case kodev1alpha2.EntryPointPhaseActive:
		result, err = r.handleActiveState(ctx, entryPoint)
	case kodev1alpha2.EntryPointPhaseDeleting:
		result, err = r.handleDeletingState(ctx, entryPoint)
	case kodev1alpha2.EntryPointPhaseFailed:
		result, err = r.handleFailedState(ctx, entryPoint)
	default:
		log.Info("Unknown phase, transitioning to Failed", "phase", entryPoint.Status.Phase)
		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseFailed)
	}

	// Handle errors from state handlers
	if err != nil {
		log.Error(err, "Error handling state", "phase", entryPoint.Status.Phase)
		if entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseFailed {
			// Transition to failed state if not already there
			result, err = r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseFailed)
			return result, err // Early return after transition
		}
		// If already in failed state, just requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Kode resource still exists before updating status
	latestEntryPoint := &kodev1alpha2.Kode{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: entryPoint.Name, Namespace: entryPoint.Namespace}, latestEntryPoint); err != nil {
		if errors.IsNotFound(err) {
			// Kode resource has been deleted, nothing to update
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get latest Kode")
		return ctrl.Result{Requeue: true}, err
	}

	// Update the whole status if it has changed
	if !reflect.DeepEqual(latestEntryPoint.Status, entryPoint.Status) {
		if err := r.updateStatus(ctx, entryPoint); err != nil {
			// If we fail to update the status, requeue
			return ctrl.Result{Requeue: true}, err
		}
	}

	log.V(1).Info("Reconciliation completed", "Phase", entryPoint.Status.Phase, "result", result)
	return result, nil
}

func (r *EntryPointReconciler) reconcileKode(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithName("KodeReconciler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)

	// Only reconcile if Kode is in Active phase
	if kode.Status.Phase != kodev1alpha2.KodePhaseActive {
		log.V(1).Info("Kode is not in Active phase, skipping reconciliation", "phase", kode.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Find the associated EntryPoint
	entryPoint, err := r.findEntryPointForKode(ctx, kode)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("EntryPoint not found, requeuing", "error", err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		log.Error(err, "Unable to find EntryPoint for Kode")
		return r.handleReconcileError(ctx, kode, nil, err, "Unable to find EntryPoint for Kode")
	}

	config := InitEntryPointResourcesConfig(entryPoint)

	// Construct Kode URL
	kodeHostname, kodeDomain, kodeUrl, kodePath, err := kode.GenerateKodeUrlForEntryPoint(entryPoint.Spec.RoutingType, entryPoint.Spec.BaseDomain, kode.Name, config.Protocol)
	if err != nil {
		return r.handleReconcileError(ctx, kode, entryPoint, err, "Failed to construct Kode URL")
	}
	log.V(1).Info("Constructed Kode URL", "hostname", kodeHostname, "domain", kodeDomain, "url", kodeUrl, "path", kodePath, "protocol", config.Protocol)

	// Check if Kode URL has changed
	if kode.Status.KodeUrl == kodeUrl {
		log.Info("Kode URL has not changed, skipping reconciliation", "KodeUrl", kode.Status.KodeUrl)
		return ctrl.Result{}, nil
	}

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

	// Update kode URL
	if kodeUrl != "" {
		kode.UpdateUrl(ctx, r.Client, kodeUrl)
		// TODO: Rewrite status handling to work with SetCondition
		// kode.SetCondition(constants.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "Kode is available and ready to use")
	}

	log.Info("HTTPRoute configuration successful", "created", created, "kodeUrl", kodeUrl)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryPointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.EntryPoint{}).
		Owns(&gwapiv1.HTTPRoute{}).
		Owns(&gwapiv1.Gateway{}).
		Watches(&kodev1alpha2.Kode{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
