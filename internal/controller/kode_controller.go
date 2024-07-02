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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/cleanup"
	"github.com/emil-jacero/kode-operator/internal/common"
	"github.com/emil-jacero/kode-operator/internal/repository"
	"github.com/emil-jacero/kode-operator/internal/resource"
	"github.com/emil-jacero/kode-operator/internal/status"
	"github.com/emil-jacero/kode-operator/internal/template"
	"github.com/emil-jacero/kode-operator/internal/validation"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type KodeReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	Repo            repository.Repository
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
	ctx, cancel := common.ContextWithTimeout(ctx, 60) // 60 seconds timeout
	defer cancel()

	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Kode resource not found.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Kode")
		return ctrl.Result{}, err
	}

	// Check if the Kode instance is being deleted
	if !kode.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleCleanup(ctx, kode)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(kode, common.FinalizerName) {
		controllerutil.AddFinalizer(kode, common.FinalizerName)
		if err := r.Update(ctx, kode); err != nil {
			currentTime := r.GetCurrentTime()
			return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatus(ctx, kode, kodev1alpha1.KodePhaseActive, nil, err.Error(), &currentTime)
		}
	}

	// // Validate Kode spec
	// if err := r.Validator.Validate(kode); err != nil {
	// 	return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatusWithError(ctx, kode, err)
	// }

	// Fetch templates
	templates := &common.Templates{}
	templates, err := r.TemplateManager.Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		currentTime := r.GetCurrentTime()
		if _, ok := err.(*common.TemplateNotFoundError); ok {
			log.Info("Template not found, requeuing", "error", err)
			return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatus(ctx, kode, kodev1alpha1.KodePhaseActive, nil, err.Error(), &currentTime)
		}
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatus(ctx, kode, kodev1alpha1.KodePhaseError, nil, err.Error(), &currentTime)
	}

	// Initialize Kode resources config
	config := InitKodeResourcesConfig(kode, templates)

	// Ensure resources
	if err := r.ensureResources(ctx, config); err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.UpdateStatusWithError(ctx, kode, err)
	}

	// Check if all resources are ready
	ready, err := r.checkResourcesReady(ctx, kode, templates)
	if err != nil {
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.UpdateStatusWithError(ctx, kode, fmt.Errorf("failed to check resource readiness: %w", err))
	}
	if !ready {
		log.Info("Resources not ready, requeuing")
		currentTime := r.GetCurrentTime()
		return ctrl.Result{RequeueAfter: common.RequeueInterval}, r.StatusUpdater.UpdateStatus(ctx, kode, kodev1alpha1.KodePhaseActive, nil, "resources not ready", &currentTime)
	}

	// Update status to success
	if err := r.UpdateStatusWithSuccess(ctx, kode); err != nil {
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

func (r *KodeReconciler) checkResourcesReady(ctx context.Context, kode *kodev1alpha1.Kode, templates *common.Templates) (bool, error) {
	ctx, cancel := common.ContextWithTimeout(ctx, 20) // 20 seconds timeout
	defer cancel()
	log := r.Log.WithName("ResourceReadyChecker").WithValues("kode", client.ObjectKeyFromObject(kode))

	// Check StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, statefulSet)
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
	err = r.Get(ctx, types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Service not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Service: %w", err)
	}

	// Check PersistentVolumeClaim if storage is specified
	if !kode.Spec.Storage.IsEmpty() {
		pvc := &corev1.PersistentVolumeClaim{}
		err = r.Get(ctx, types.NamespacedName{Name: common.GetPVCName(kode), Namespace: kode.Namespace}, pvc)
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
	if templates.KodeTemplate != nil && templates.KodeTemplate.EnvoyProxyRef.Name != "" {
		ready, err := r.checkEnvoySidecarReady(ctx, kode)
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

func (r *KodeReconciler) checkEnvoySidecarReady(ctx context.Context, kode *kodev1alpha1.Kode) (bool, error) {
	log := r.Log.WithName("EnvoySidecarReadyChecker").WithValues("kode", client.ObjectKeyFromObject(kode))

	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: kode.Name + "-0", Namespace: kode.Namespace}, pod)
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

func (r *KodeReconciler) handleCleanup(ctx context.Context, kode *kodev1alpha1.Kode) (ctrl.Result, error) {
	log := r.Log.WithName("CleanupHandler").WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Handling Kode resource deletion")

	if controllerutil.ContainsFinalizer(kode, common.FinalizerName) {
		// Perform cleanup
		if err := r.CleanupManager.Cleanup(ctx, kode); err != nil {
			return ctrl.Result{Requeue: true}, r.UpdateStatusWithError(ctx, kode, err)
		}
	}

	// // Update status to recycled
	// if err := r.UpdateStatusRecycled(ctx, kode); err != nil {
	// 	log.Error(err, "Failed to update status to recycled")
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	// // Clear error status
	// if err := r.clearErrorStatus(ctx, kode); err != nil {
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	log.Info("Kode resource deletion handled successfully")
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
