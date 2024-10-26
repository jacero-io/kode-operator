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
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resourcev1"
	"github.com/jacero-io/kode-operator/internal/template"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

type EntryPointReconciler struct {
	Client                client.Client
	Scheme                *runtime.Scheme
	Log                   logr.Logger
	Resource              resourcev1.ResourceManager
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
	case *kodev1alpha2.Kode:
		log.V(1).Info("Reconciling Kode", "namespace", v.Namespace, "name", v.Name)
		return r.reconcileKode(ctx, v)
	default:
		log.Error(nil, "Unknown resource type", "type", fmt.Sprintf("%T", obj))
		return ctrl.Result{}, nil
	}
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

func (r *EntryPointReconciler) GetResourceManager() resourcev1.ResourceManager {
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
