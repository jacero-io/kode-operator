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

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
	"github.com/jacero-io/kode-operator/internal/template"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

type KodeReconciler struct {
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

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=containertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clustercontainertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=tofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clustertofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;watch;create;update;patch;delete,namespace=kode-system
// +kubebuilder:rbac:groups="storage.k8s.io",resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=event,verbs=create;patch;update

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", req.NamespacedName)
	log.V(1).Info("Starting reconciliation")

	kode := &kodev1alpha2.Kode{}
	if err := r.Resource.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Unable to fetch Kode")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Fetched Kode resource", "Name", kode.Name, "Namespace", kode.Namespace, "Generation", kode.Generation, "ObservedGeneration", kode.Status.ObservedGeneration, "Phase", kode.Status.Phase)

	sm := statemachine.NewStateMachine(r.Client, r.Log)
	sm.RegisterHandler(kodev1alpha2.PhasePending, handlePendingState)
	sm.RegisterHandler(kodev1alpha2.PhaseConfiguring, handleConfiguringState)
	sm.RegisterHandler(kodev1alpha2.PhaseProvisioning, handleProvisioningState)
	sm.RegisterHandler(kodev1alpha2.PhaseActive, handleActiveState)
	sm.RegisterHandler(kodev1alpha2.PhaseUpdating, handleUpdatingState)
	sm.RegisterHandler(kodev1alpha2.PhaseDeleting, handleDeletingState)
	sm.RegisterHandler(kodev1alpha2.PhaseFailed, handleFailedState)
	sm.RegisterHandler(kodev1alpha2.PhaseUnknown, handleUnknownState)

	// Handle finalizer
	if !kode.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(kode, constant.KodeFinalizerName) {
			// Consider the resource deleted, do nothing
			return ctrl.Result{}, nil
		}
		// Change to deleting state
		kode.Status.Phase = kodev1alpha2.PhaseDeleting
		if err := kode.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		// When the Kode resource is completely new, add the finalizer and transition to Pending state
	} else if !controllerutil.ContainsFinalizer(kode, constant.KodeFinalizerName) {
		controllerutil.AddFinalizer(kode, constant.KodeFinalizerName)
		log.Info("Added finalizer to Kode resource")

		// Transition to Pending state to begin initialization
		log.Info("Transitioning to Pending state to begin initialization")
		kode.Status.Phase = kodev1alpha2.PhasePending
		if err := kode.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle generation mismatch
	if kode.Generation != kode.Status.ObservedGeneration {
		return r.handleGenerationMismatch(ctx, kode)
	}

	// Reset retry count if not in failed state
	if kode.Status.Phase != kodev1alpha2.PhaseFailed && kode.Status.RetryCount > 0 {
		kode.Status.RetryCount = 0
		if err := kode.UpdateStatus(ctx, r.Client); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Run state machine
	result, err := sm.HandleState(ctx, r, kode)
	if err != nil {
		log.Error(err, "Error handling state", "phase", kode.Status.Phase)
		return ctrl.Result{Requeue: true}, err
	}

	log.V(1).Info("Reconciliation completed", "phase", kode.Status.Phase)
	return result, nil
}

func (r *KodeReconciler) HandleReconcileError(ctx context.Context, resource statemachine.StateManagedResource, err error, message string) (kodev1alpha2.Phase, ctrl.Result, error) {
	kode, ok := resource.(*kodev1alpha2.Kode)
	if !ok {
		return kodev1alpha2.PhaseFailed, ctrl.Result{}, fmt.Errorf("resource is not a Kode")
	}

	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Error(err, message)

	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "ReconciliationFailed", fmt.Sprintf("%s: %v", message, err))
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionFalse, "ReconciliationFailed", "Kode resource is not available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "ReconciliationFailed", "Kode resource is not progressing")

	kode.Status.LastError = common.StringPtr(err.Error())
	kode.Status.LastErrorTime = &metav1.Time{Time: metav1.Now().Time}

	if updateErr := kode.UpdateStatus(ctx, r.Client); updateErr != nil {
		log.Error(updateErr, "Failed to update Kode status after error")
	}

	return kodev1alpha2.PhaseFailed, ctrl.Result{}, err
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.Kode{}).
		// WithEventFilter(predicate.GenerationChangedPredicate{}).
		// Watches(&kodev1alpha2.ContainerTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.ClusterContainerTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.TofuTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha2.ClusterTofuTemplate{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}

func (r *KodeReconciler) GetClient() client.Client {
	return r.Client
}

func (r *KodeReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

func (r *KodeReconciler) GetLog() logr.Logger {
	return r.Log
}

func (r *KodeReconciler) GetResourceManager() resource.ResourceManager {
	return r.Resource
}

func (r *KodeReconciler) GetEventRecorder() event.EventManager {
	return r.EventManager
}

func (r *KodeReconciler) GetTemplateManager() template.TemplateManager {
	return r.Template
}

func (r *KodeReconciler) GetCleanupManager() cleanup.CleanupManager {
	return r.CleanupManager
}

func (r *KodeReconciler) GetIsTestEnvironment() bool {
	return r.IsTestEnvironment
}

func (r *KodeReconciler) GetReconcileInterval() time.Duration {
	return r.ReconcileInterval
}

func (r *KodeReconciler) GetLongReconcileInterval() time.Duration {
	return r.LongReconcileInterval
}
