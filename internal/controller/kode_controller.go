// internal/controller/kode_controller.go

/*
Copyright 2024.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	// Initialize Kode resources config
	config := InitKodeResourcesConfig(kode, templates)

	// Ensure resources
	if err := r.ensureResources(ctx, config); err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.UpdateStatusWithError(ctx, config, err)
	}

	// Check if all resources are ready
	ready, err := r.checkResourcesReady(ctx, config)
	if err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.UpdateStatusWithError(ctx, config, fmt.Errorf("failed to check resource readiness: %w", err))
	}
	if !ready {
		log.Info("Resources not ready, requeuing")
		currentTime := r.GetCurrentTime()
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatus(ctx, config, kodev1alpha1.KodePhaseActive, nil, "resources not ready", &currentTime)
	}

	// Update status to success
	if err := r.UpdateStatusWithSuccess(ctx, config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Kode reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) ensureResources(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("ResourceEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	// Ensure StatefulSet
	if err := r.ensureStatefulSet(ctx, config); err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		return err
	}

	// Ensure Service
	if err := r.ensureService(ctx, config); err != nil {
		log.Error(err, "Failed to ensure Service")
		return err
	}

	// Ensure PVC if storage is specified
	if !config.Kode.Spec.DeepCopy().Storage.IsEmpty() {
		if err := r.ensurePVC(ctx, config); err != nil {
			log.Error(err, "Failed to ensure PVC")
			return err
		}
	}

	log.Info("All resources ensured successfully")
	return nil
}

func (r *KodeReconciler) checkResourcesReady(ctx context.Context, config *common.KodeResourcesConfig) (bool, error) {
	log := r.Log.WithName("ResourceReadyChecker").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))
	ctx, cancel := common.ContextWithTimeout(ctx, 20) // 20 seconds timeout
	defer cancel()

	// Check StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: config.KodeName, Namespace: config.KodeNamespace}, statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("StatefulSet not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	if statefulSet.Status.ReadyReplicas != statefulSet.Status.Replicas {
		log.Info("StatefulSet not ready", "ReadyReplicas", statefulSet.Status.ReadyReplicas, "DesiredReplicas", statefulSet.Status.Replicas)
		return false, nil
	}

	// Check Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: config.KodeNamespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Service not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Service: %w", err)
	}

	// Check PersistentVolumeClaim if storage is specified
	if !config.Kode.Spec.Storage.IsEmpty() {
		pvc := &corev1.PersistentVolumeClaim{}
		err = r.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: config.KodeNamespace}, pvc)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("PersistentVolumeClaim not found")
				return false, nil
			}
			return false, fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			log.Info("PersistentVolumeClaim not bound", "Phase", pvc.Status.Phase)
			return false, nil
		}
	}

	// Check if Envoy sidecar is ready (if applicable)
	if config.Templates.KodeTemplate != nil && config.Templates.KodeTemplate.EnvoyProxyRef.Name != "" {
		ready, err := r.checkEnvoySidecarReady(ctx, config)
		if err != nil {
			return false, fmt.Errorf("failed to check Envoy sidecar readiness: %w", err)
		}
		if !ready {
			log.Info("Envoy sidecar not ready")
			return false, nil
		}
	}

	log.Info("All resources are ready")
	return true, nil
}

func (r *KodeReconciler) checkEnvoySidecarReady(ctx context.Context, config *common.KodeResourcesConfig) (bool, error) {
	log := r.Log.WithName("EnvoySidecarReadyChecker").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: config.KodeName + "-0", Namespace: config.KodeNamespace}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Pod not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Pod: %w", err)
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "envoy-proxy" {
			if !containerStatus.Ready {
				log.Info("Envoy sidecar container not ready")
				return false, nil
			}
			return true, nil
		}
	}

	log.Info("Envoy sidecar container not found in pod")
	return false, nil
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

func (r *KodeReconciler) handleFinalizer(ctx context.Context, kode *kodev1alpha1.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is not being deleted, so ensure the finalizer is present
		if !controllerutil.ContainsFinalizer(kode, common.FinalizerName) {
			controllerutil.AddFinalizer(kode, common.FinalizerName)
			log.Info("Adding finalizer", "finalizer", common.FinalizerName)
			if err := r.Update(ctx, kode); err != nil {
				log.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		// Object is being deleted
		if controllerutil.ContainsFinalizer(kode, common.FinalizerName) {
			// Run finalization logic
			if err := r.finalize(ctx, kode); err != nil {
				log.Error(err, "Failed to run finalizer")
				// Don't return here, continue to remove the finalizer
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(kode, common.FinalizerName)
			log.Info("Removing finalizer", "finalizer", common.FinalizerName)
			if err := r.Update(ctx, kode); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) finalize(ctx context.Context, kode *kodev1alpha1.Kode) error {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Running finalizer")

	// Initialize Kode resources config without templates
	config := &common.KodeResourcesConfig{
		Kode:          *kode,
		KodeName:      kode.Name,
		KodeNamespace: kode.Namespace,
		PVCName:       kode.Name + "-pvc",
		ServiceName:   kode.Name + "-svc",
	}

	// Perform cleanup
	return r.CleanupManager.Cleanup(ctx, config)
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
