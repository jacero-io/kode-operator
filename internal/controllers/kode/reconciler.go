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

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
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
		return ctrl.Result{}, client.IgnoreNotFound(err)
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

	// Fetch templates for normal reconciliation
	templates, err := r.fetchTemplatesWithRetry(ctx, kode)
	if err != nil {
		log.Error(err, "Failed to fetch templates after retries")
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, err
	}

	// Validate Kode instance and resources
	// if err := r.Validator.Validate(kode, templates); err != nil {
	// 	return ctrl.Result{}, r.UpdateStatusWithError(ctx, config, fmt.Errorf("failed to validate Kode instance: %w", err))
	// }

	// Initialize Kode resources config
	config := InitKodeResourcesConfig(kode, templates)

	// Ensure resources
	if err := r.ensureResources(ctx, config); err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, err
	}

	// Check if all resources are ready
	ready, err := r.checkResourcesReady(ctx, config)
	if err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.updateKodePhaseFailed(ctx, config, fmt.Errorf("failed to check resource readiness: %w", err))
	}
	if !ready {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.updateKodePhasePending(ctx, config)
	}

	// Update status to active
	if err := r.updateKodePhaseActive(ctx, config); err != nil {
		return ctrl.Result{Requeue: true}, err
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

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
