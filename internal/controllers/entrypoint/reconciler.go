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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
	"github.com/jacero-io/kode-operator/internal/template"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

type EntryPointReconciler struct {
	Client                client.Client
	Scheme                *runtime.Scheme
	Log                   logr.Logger
	Resource              resource.ResourceManager
	Template              template.TemplateManager
	CleanupManager        cleanup.CleanupManager
	EventManager          event.EventManager
	IsTestEnvironment     bool
	ReconcileInterval     time.Duration
	LongReconcileInterval time.Duration
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=event,verbs=create;patch;update

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

func (r *EntryPointReconciler) reconcileEntryPoint(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", req.NamespacedName)
	log.V(1).Info("Starting reconciliation")

	entryPoint := &kodev1alpha2.EntryPoint{}
	if err := r.Resource.Get(ctx, req.NamespacedName, entryPoint); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Unable to fetch Kode")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Fetched Kode resource", "Name", entryPoint.Name, "Namespace", entryPoint.Namespace, "Generation", entryPoint.Generation, "ObservedGeneration", entryPoint.Status.ObservedGeneration, "Phase", entryPoint.Status.Phase)

	sm := statemachine.NewStateMachine(r.Client, r.Log)
	// sm.RegisterHandler(kodev1alpha2.PhasePending, handlePendingState)
	// sm.RegisterHandler(kodev1alpha2.PhaseConfiguring, handleConfiguringState)
	// sm.RegisterHandler(kodev1alpha2.PhaseProvisioning, handleProvisioningState)
	// sm.RegisterHandler(kodev1alpha2.PhaseActive, handleActiveState)
	// sm.RegisterHandler(kodev1alpha2.PhaseUpdating, handleUpdatingState)
	// sm.RegisterHandler(kodev1alpha2.PhaseDeleting, handleDeletingState)
	// sm.RegisterHandler(kodev1alpha2.PhaseFailed, handleFailedState)
	// sm.RegisterHandler(kodev1alpha2.PhaseUnknown, handleUnknownState)

	// Handle finalizer
	if !entryPoint.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(entryPoint, constant.EntryPointFinalizerName) {
			// Consider the resource deleted, do nothing
			return ctrl.Result{}, nil
		}
		// Change to deleting state
		entryPoint.Status.Phase = kodev1alpha2.PhaseDeleting
		if err := entryPoint.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		// When the EntryPoint resource is completely new, add the finalizer and transition to Pending state
	} else if !controllerutil.ContainsFinalizer(entryPoint, constant.EntryPointFinalizerName) {
		controllerutil.AddFinalizer(entryPoint, constant.EntryPointFinalizerName)
		log.Info("Added finalizer to EntryPoint resource")

		// Transition to Pending state to begin initialization
		log.Info("Transitioning to Pending state to begin initialization")
		entryPoint.Status.Phase = kodev1alpha2.PhasePending
		if err := entryPoint.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle generation mismatch
	if entryPoint.Generation != entryPoint.Status.ObservedGeneration {
		return r.handleGenerationMismatch(ctx, entryPoint)
	}

	// Reset retry count if not in failed state
	if entryPoint.Status.Phase != kodev1alpha2.PhaseFailed && entryPoint.Status.RetryCount > 0 {
		entryPoint.Status.RetryCount = 0
		if err := entryPoint.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Run state machine
	result, err := sm.HandleState(ctx, r, entryPoint)
	if err != nil {
		log.Error(err, "Error handling state", "phase", entryPoint.Status.Phase)
		return ctrl.Result{Requeue: true}, err
	}

	log.V(1).Info("Reconciliation completed", "phase", entryPoint.Status.Phase)
	return result, nil
}

func (r *EntryPointReconciler) reconcileKode(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithName("KodeReconciler").WithValues(
		"kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace},
	)

	// Only reconcile if Kode is in Active phase
	if kode.Status.Phase != kodev1alpha2.PhaseActive {
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

		// Update Kode status with error
		kode.SetCondition(constant.ConditionTypeError, metav1.ConditionTrue, "ReconciliationFailed", fmt.Sprintf("Unable to find EntryPoint for Kode: %v", err))
		kode.Status.Phase = kodev1alpha2.PhaseFailed
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		if updateErr := kode.UpdateStatus(ctx, r.Client); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("primary error: %v, kode status update error: %v", err, updateErr)
		}

		r.GetEventRecorder().Record(ctx, kode, event.EventTypeWarning, event.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	config := InitEntryPointResourcesConfig(entryPoint)

	// Construct Kode URL
	kodeHostname, kodeDomain, kodeUrl, kodePath, err := kode.GenerateKodeUrlForEntryPoint(entryPoint.Spec.RoutingType, entryPoint.Spec.BaseDomain, kode.Name, config.Protocol)
	if err != nil {
		log.Error(err, "Failed to construct Kode URL")

		// Update Kode status with error
		kode.SetCondition(constant.ConditionTypeError, metav1.ConditionTrue, "ReconciliationFailed", fmt.Sprintf("Failed to construct Kode URL: %v", err))
		kode.Status.Phase = kodev1alpha2.PhaseFailed
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		if updateErr := kode.UpdateStatus(ctx, r.Client); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("primary error: %v, kode status update error: %v", err, updateErr)
		}

		r.GetEventRecorder().Record(ctx, kode, event.EventTypeWarning, event.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
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

		// Update Kode status with error
		kode.SetCondition(constant.ConditionTypeError, metav1.ConditionTrue, "ReconciliationFailed", fmt.Sprintf("Failed to ensure HTTPRoute: %v", err))
		kode.Status.Phase = kodev1alpha2.PhaseFailed
		kode.Status.LastError = common.StringPtr(err.Error())
		kode.Status.LastErrorTime = &metav1.Time{Time: time.Now()}
		if updateErr := kode.UpdateStatus(ctx, r.Client); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("primary error: %v, kode status update error: %v", err, updateErr)
		}

		r.GetEventRecorder().Record(ctx, kode, event.EventTypeWarning, event.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// Update kode URL
	if kodeUrl != "" {
		kode.Status.KodeUrl = kodeUrl
		kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "ResourcesReady", "Kode is available and ready to use")
		if err := kode.UpdateStatus(ctx, r.Client); err != nil {
			log.Error(err, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("failed to update Kode URL: %w", err)
		}
	}

	log.Info("HTTPRoute configuration successful", "created", created, "kodeUrl", kodeUrl)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryPointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.EntryPoint{}).
		// WithEventFilter(predicate.GenerationChangedPredicate{}).
		Owns(&gwapiv1.HTTPRoute{}).
		Owns(&gwapiv1.Gateway{}).
		Watches(&kodev1alpha2.Kode{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *EntryPointReconciler) GetClient() client.Client {
	return r.Client
}

func (r *EntryPointReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

func (r *EntryPointReconciler) GetLog() logr.Logger {
	return r.Log
}

func (r *EntryPointReconciler) GetResourceManager() resource.ResourceManager {
	return r.Resource
}

func (r *EntryPointReconciler) GetEventRecorder() event.EventManager {
	return r.EventManager
}

func (r *EntryPointReconciler) GetTemplateManager() template.TemplateManager {
	return r.Template
}

func (r *EntryPointReconciler) GetCleanupManager() cleanup.CleanupManager {
	return r.CleanupManager
}

func (r *EntryPointReconciler) GetIsTestEnvironment() bool {
	return r.IsTestEnvironment
}

func (r *EntryPointReconciler) GetReconcileInterval() time.Duration {
	return r.ReconcileInterval
}

func (r *EntryPointReconciler) GetLongReconcileInterval() time.Duration {
	return r.LongReconcileInterval
}
