// internal/controllers/kode/reconciler.go

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

package controller

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type KodeReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	ResourceManager resource.ResourceManager
	TemplateManager template.TemplateManager
	CleanupManager  cleanup.CleanupManager
	StatusUpdater   status.StatusUpdater
	Validator       validation.Validator
}

// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodeclustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyclusterconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("Reconcile").WithValues("kode", req.NamespacedName)

	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to fetch Kode")
		return ctrl.Result{RequeueAfter: r.calculateBackoff(1)}, nil
	}

	// Handle finalizer
	if result, err := r.handleFinalizer(ctx, kode); err != nil {
		return result, err
	} else if !result.IsZero() {
		return result, nil
	}

	// If the object is being deleted, stop here
	if !kode.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Increment attempts at the start of reconciliation
	kode.Status.ReconcileAttempts++
	if err := r.Status().Update(ctx, kode); err != nil {
		log.Error(err, "Failed to update Kode ReconcileAttempts")
		return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, err
	}

	// Refresh the Kode resource to get the updated ReconcileAttempts
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		log.Error(err, "Unable to fetch Kode")
		return ctrl.Result{RequeueAfter: r.calculateBackoff(1)}, nil
	}

	// Fetch templates and initialize config
	templates, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch templates")
		if updateErr := r.updateKodePhaseFailed(ctx, InitKodeResourcesConfig(kode, nil), fmt.Errorf("failed to fetch templates: %w", err)); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
		}
		return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, err
	}

	config := InitKodeResourcesConfig(kode, templates)

	// Ensure resources
	if err := r.ensureResources(ctx, config); err != nil {
		log.Error(err, "Failed to ensure resources")
		if updateErr := r.updateKodePhaseFailed(ctx, config, err); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
		}
		return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, err
	}

	// Check if all resources are ready
	ready, err := r.checkResourcesReady(ctx, config)
	if err != nil {
		log.Error(err, "Failed to check resource readiness")
		if updateErr := r.updateKodePhaseFailed(ctx, config, fmt.Errorf("failed to check resource readiness: %w", err)); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode status")
			return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, nil
		}
		return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, nil
	}
	if !ready {
		log.Info("Resources not ready, requeuing")
		// Refresh the Kode resource to get the updated ReconcileAttempts
		if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
			log.Error(err, "Unable to fetch Kode")
			return ctrl.Result{RequeueAfter: r.calculateBackoff(1)}, nil
		}
		kode.Status.ReconcileAttempts++
		if updateErr := r.Status().Update(ctx, kode); updateErr != nil {
			log.Error(updateErr, "Failed to update Kode ReconcileAttempts")
			// If we fail to update, we'll still requeue with backoff but won't increment the attempts
			return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, nil
		}
		return ctrl.Result{RequeueAfter: r.calculateBackoff(kode.Status.ReconcileAttempts)}, nil
	}

	// Resources are ready, reset attempts and update status to active
	kode.Status.ReconcileAttempts = 0
	if err := r.Update(ctx, kode); err != nil {
		log.Error(err, "Failed to update Kode ReconcileAttempts")
		return ctrl.Result{RequeueAfter: r.calculateBackoff(1)}, err
	}
	if err := r.updateKodePhaseActive(ctx, config); err != nil {
		log.Error(err, "Failed to update Kode status to Active")
		return ctrl.Result{RequeueAfter: r.calculateBackoff(1)}, nil
	}

	log.Info("Kode reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) fetchTemplatesWithRetry(ctx context.Context, kode *kodev1alpha1.Kode) (*common.Templates, error) {
	var templates *common.Templates
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		templates, err = r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
		return err
	})
	return templates, err
}

func (r *KodeReconciler) calculateBackoff(attempts int) time.Duration {
	baseDelay := time.Second * 5
	maxDelay := time.Minute * 5
	multiplier := math.Pow(2, float64(attempts))
	delay := time.Duration(float64(baseDelay) * multiplier)
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
