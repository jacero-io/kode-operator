// internal/controllers/kode/finalizer.go

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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
				return ctrl.Result{}, err
			}

			// Remove finalizer
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Fetch the latest version of Kode
				latestKode := &kodev1alpha1.Kode{}
				if err := r.Get(ctx, client.ObjectKeyFromObject(kode), latestKode); err != nil {
					return err
				}

				if controllerutil.ContainsFinalizer(latestKode, common.FinalizerName) {
					controllerutil.RemoveFinalizer(latestKode, common.FinalizerName)
					log.Info("Removing finalizer", "finalizer", common.FinalizerName)
					return r.Update(ctx, latestKode)
				}
				return nil
			})

			if err != nil {
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

	// Initialize Kode resources config without templates
	config := &common.KodeResourcesConfig{
		KodeSpec:      kode.Spec,
		KodeName:      kode.Name,
		KodeNamespace: kode.Namespace,
		PVCName:       common.GetPVCName(kode),
		ServiceName:   common.GetServiceName(kode),
	}

	// Perform cleanup
	return r.CleanupManager.Cleanup(ctx, config)
}