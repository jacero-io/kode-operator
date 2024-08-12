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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
)

// EntryPointReconciler reconciles a EntryPoint object
type EntryPointReconciler struct {
	Client          client.Client
	Scheme          *runtime.Scheme
	Log             logr.Logger
	ResourceManager resource.ResourceManager
	TemplateManager template.TemplateManager
	CleanupManager  cleanup.CleanupManager
	StatusUpdater   status.StatusUpdater
	Validator       validation.Validator
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=entrypoints/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodeclustertemplates,verbs=get;list;watch

func (r *EntryPointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", req.NamespacedName)

	// Fetch the EntryPoint instance
	entryPoint := &kodev1alpha1.EntryPoint{}
	err := r.Client.Get(ctx, req.NamespacedName, entryPoint)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("EntryPoint resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get EntryPoint")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryPointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.EntryPoint{}).
		Complete(r)
}
