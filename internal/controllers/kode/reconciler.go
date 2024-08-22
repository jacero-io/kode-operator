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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

type KodeReconciler struct {
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

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.jacero.io,resources=podtemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clusterpodtemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=tofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=clustertofutemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", req.NamespacedName)

	// Fetch the Kode instance
	kode, err := common.GetLatestKode(ctx, r.Client, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Kode resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Kode")
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	// Handle finalizer
	result, err := r.handleFinalizer(ctx, kode)
	if err != nil {
		return result, err
	}
	if !result.IsZero() {
		return result, nil
	}

	// If the object is being deleted, stop here
	if !kode.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Check if reconciliation is needed
	if kode.Generation == kode.Status.ObservedGeneration && kode.Status.Phase == kodev1alpha2.KodePhaseActive {
		log.V(1).Info("Resource is stable and active, no reconciliation needed")
		return ctrl.Result{}, nil // No requeue, controller will be triggered by changes
	}

	// Fetch templates and initialize config
	template, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch templates")
		if errors.IsNotFound(err) {
			if statusErr := r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
				{
					Type:    string(common.ConditionTypeError),
					Status:  metav1.ConditionTrue,
					Reason:  "TemplateMissing",
					Message: fmt.Sprintf("Failed to fetch templates: %s", err.Error()),
				},
				{
					Type:    string(common.ConditionTypeReady),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceNotReady",
					Message: "Kode resource is not ready due to Template was not found",
				},
				{
					Type:    string(common.ConditionTypeAvailable),
					Status:  metav1.ConditionFalse,
					Reason:  "ResourceUnavailable",
					Message: "Kode resource is not available due to Template was not found",
				},
				{
					Type:    string(common.ConditionTypeProgressing),
					Status:  metav1.ConditionFalse,
					Reason:  "ProgressHalted",
					Message: "Progress halted due to Template was not found",
				},
			}); statusErr != nil {
				log.Error(statusErr, "Failed to update Kode status")
				return ctrl.Result{RequeueAfter: RequeueInterval}, err
			}

			err = r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
			if err != nil {
				log.Error(err, "Failed to record event")
			}

			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
		// For other errors, requeue after a shorter interval
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}
	config := InitKodeResourcesConfig(kode, template)

	// Ensure resources
	if err := r.ensureResources(ctx, config); err != nil {
		log.Error(err, "Failed to ensure resources")
		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	// Check resource readiness
	ready, err := r.checkResourcesReady(ctx, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	if !ready {
		if kode.Status.Phase != kodev1alpha2.KodePhasePending {
			log.Info("Resources not ready, updating status to Pending")
			if statusErr := r.updatePhasePending(ctx, kode); statusErr != nil {
				log.Error(statusErr, "Failed to update Kode status to Pending")
				return ctrl.Result{RequeueAfter: RequeueInterval}, statusErr
			}
		}
		return ctrl.Result{RequeueAfter: RequeueInterval}, nil // Requeue while not ready
	}

	// Update Port Status
	if err := r.updatePortStatus(ctx, kode, config.Template); err != nil {
		log.Error(err, "Failed to update Port status")
		if statusErr := r.updatePhaseFailed(ctx, kode, err, []metav1.Condition{
			{
				Type:    string(common.ConditionTypeError),
				Status:  metav1.ConditionTrue,
				Reason:  "PortUpdate",
				Message: fmt.Sprintf("Failed to update Port status: %s", err.Error()),
			},
			{
				Type:    string(common.ConditionTypeReady),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceNotReady",
				Message: "Kode resource is not ready due to failed port update",
			},
			{
				Type:    string(common.ConditionTypeAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "ResourceUnavailable",
				Message: "Kode resource is not available due to failed port update",
			},
			{
				Type:    string(common.ConditionTypeProgressing),
				Status:  metav1.ConditionFalse,
				Reason:  "ProgressHalted",
				Message: "Progress halted due to failed port update",
			},
		}); statusErr != nil {
			log.Error(statusErr, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: RequeueInterval}, err
		}

		err := r.EventManager.Record(ctx, kode, events.EventTypeWarning, events.ReasonFailed, fmt.Sprintf("Failed to reconcile Kode: %v", err))
		if err != nil {
			log.Error(err, "Failed to record event")
			return ctrl.Result{RequeueAfter: RequeueInterval}, err
		}

		return ctrl.Result{RequeueAfter: RequeueInterval}, err
	}

	// Resources are ready, update status to Active if it's not already
	if kode.Status.Phase != kodev1alpha2.KodePhaseActive {
		log.Info("Resources ready, updating status to Active")
		if statusErr := r.updatePhaseActive(ctx, kode); statusErr != nil {
			log.Error(statusErr, "Failed to update Kode status to Active")
			return ctrl.Result{RequeueAfter: RequeueInterval}, statusErr
		}

		err := r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonReconciled, "Kode resource reconciled successfully")
		if err != nil {
			log.Error(err, "Failed to record event")
			return ctrl.Result{RequeueAfter: RequeueInterval}, err
		}
	}

	// Update ObservedGeneration
	if kode.Status.ObservedGeneration != kode.Generation {
		if err := r.updateObservedGeneration(ctx, kode); err != nil {
			log.Error(err, "Failed to update ObservedGeneration")
			return ctrl.Result{RequeueAfter: RequeueInterval}, err
		}
	}
	log.Info("Kode reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) fetchTemplatesWithRetry(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.Template, error) {
	var template *kodev1alpha2.Template
	var lastErr error

	backoff := wait.Backoff{
		Steps:    5,                      // Maximum number of retries
		Duration: 100 * time.Millisecond, // Initial backoff duration
		Factor:   2.0,                    // Factor to increase backoff each try
		Jitter:   0.1,                    // Jitter factor
	}

	retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
		var err error
		template, err = r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
		if err == nil {
			return true, nil // Success
		}

		if errors.IsNotFound(err) {
			r.Log.Info("Template not found, will not retry", "error", err)
			return false, err // Don't retry if not found
		}

		// For other errors, log and retry
		r.Log.Error(err, "Failed to fetch template, will retry")
		lastErr = err
		return false, nil // Retry
	})

	if retryErr != nil {
		if errors.IsNotFound(retryErr) {
			return nil, fmt.Errorf("template not found after retries: %w", retryErr)
		}
		return nil, fmt.Errorf("failed to fetch template after retries: %w", lastErr)
	}

	return template, nil
}

func (r *KodeReconciler) updateObservedGeneration(ctx context.Context, kode *kodev1alpha2.Kode) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
		if err != nil {
			return err
		}

		if latest.Status.ObservedGeneration != latest.Generation {
			r.Log.Info("Updating ObservedGeneration",
				"Name", latest.Name,
				"Namespace", latest.Namespace,
				"OldObservedGeneration", latest.Status.ObservedGeneration,
				"NewObservedGeneration", latest.Generation)

			latest.Status.ObservedGeneration = latest.Generation
			return r.Client.Status().Update(ctx, latest)
		}

		r.Log.V(1).Info("ObservedGeneration already up to date",
			"Name", latest.Name,
			"Namespace", latest.Namespace,
			"Generation", latest.Generation)
		return nil
	})
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha2.Kode{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(&kodev1alpha2.PodTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.ClusterPodTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.TofuTemplate{}, &handler.EnqueueRequestForObject{}).
		Watches(&kodev1alpha2.ClusterTofuTemplate{}, &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
