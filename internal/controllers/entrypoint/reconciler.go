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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
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
		log.V(1).Info("Reconciling EntryPoint", "namespace", v.Namespace, "name", v.Name)
		return r.reconcileEntryPoint(ctx, v)
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

	// Handle deletion
	if !entryPoint.DeletionTimestamp.IsZero() {
		return r.handleFinalizer(ctx, entryPoint)
	}

	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.ExistingGatewayRef.Name != "" {
		log.Info("EntryPoint has existing Gateway, skipping reconciliation")
		return ctrl.Result{}, nil
	} else {
		log.Info("EntryPoint does not have existing Gateway but this controller is not fully implemented yet, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// // Ensure finalizer is present
	// if !controllerutil.ContainsFinalizer(entryPoint, common.EntryPointFinalizerName) {
	// 	controllerutil.AddFinalizer(entryPoint, common.EntryPointFinalizerName)
	// 	if err := r.Client.Update(ctx, entryPoint); err != nil {
	// 		log.Error(err, "Failed to add finalizer")
	// 		return ctrl.Result{Requeue: true}, err
	// 	}
	// 	log.Info("Added finalizer to EntryPoint resource")
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	// // Initialize status if it's a new resource
	// if entryPoint.Status.Phase == "" {
	// 	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhasePending)
	// }

	// // Handle state transition
	// var result ctrl.Result
	// var err error

	// // Handle state transition
	// switch entryPoint.Status.Phase {
	// case kodev1alpha2.EntryPointPhasePending:
	// 	result, err = r.handlePendingState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseConfiguring:
	// 	result, err = r.handleConfiguringState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseProvisioning:
	// 	result, err = r.handleProvisioningState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseActive:
	// 	result, err = r.handleActiveState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseDeleting:
	// 	result, err = r.handleDeletingState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseFailed:
	// 	result, err = r.handleFailedState(ctx, entryPoint)
	// case kodev1alpha2.EntryPointPhaseUnknown:
	// 	result, err = r.handleUnknownState(ctx, entryPoint)
	// default:
	// 	log.Info("Unknown phase, transitioning to Unknown", "phase", entryPoint.Status.Phase)
	// 	return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseUnknown)
	// }

	// // Handle errors from state handlers
	// if err != nil {
	// 	log.Error(err, "Error handling state", "phase", entryPoint.Status.Phase)
	// 	if entryPoint.Status.Phase != kodev1alpha2.EntryPointPhaseFailed {
	// 		// Transition to failed state if not already there
	// 		return r.transitionTo(ctx, entryPoint, kodev1alpha2.EntryPointPhaseFailed)
	// 	}
	// 	// If already in failed state, just requeue
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	// log.V(1).Info("State transition successful", "phase", entryPoint.Status.Phase)
	// return result, nil
}

func (r *EntryPointReconciler) reconcileKode(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithName("KodeReconciler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)

	// Only reconcile if Kode is in Active phase
	if kode.Status.Phase != kodev1alpha2.KodePhaseActive {
		log.Info("Kode is not in Active phase, skipping reconciliation", "phase", kode.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Find the associated EntryPoint
	entryPoint, err := r.findEntryPointForKode(ctx, kode)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("EntryPoint not found, requeuing", "error", err)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		log.Error(err, "Unable to find EntryPoint for Kode")
		return r.handleReconcileError(ctx, kode, nil, err, "Unable to find EntryPoint for Kode")
	}

	config := InitEntryPointResourcesConfig(entryPoint)

	// Construct the KodeUrl
	kodeHostname, kodeDomain, kodeUrl, kodePath, err := kode.GenerateKodeUrlForEntryPoint(entryPoint.Spec.RoutingType, entryPoint.Spec.BaseDomain, kode.Name, config.Protocol)
	if err != nil {
		return r.handleReconcileError(ctx, kode, entryPoint, err, "Failed to construct Kode URL")
	}
	log.V(1).Info("Constructed Kode URL", "hostname", kodeHostname, "domain", kodeDomain, "url", kodeUrl, "path", kodePath, "protocol", config.Protocol)

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
		kode.UpdateKodeUrl(ctx, r.Client, kodeUrl)
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
