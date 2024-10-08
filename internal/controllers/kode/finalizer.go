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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KodeReconciler) handleFinalizer(ctx context.Context, kode *kodev1alpha2.Kode) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is not being deleted, so ensure the finalizer is present
		if !controllerutil.ContainsFinalizer(kode, common.KodeFinalizerName) {
			controllerutil.AddFinalizer(kode, common.KodeFinalizerName)
			log.Info("Adding finalizer", "finalizer", common.KodeFinalizerName)
			if err := r.Client.Update(ctx, kode); err != nil {
				log.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Object is being deleted
	if controllerutil.ContainsFinalizer(kode, common.KodeFinalizerName) {
		log.V(1).Info("Running finalization logic")
		if err := r.finalize(ctx, kode); err != nil {
			log.Error(err, "Failed to run finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Finalization logic completed successfully")

		// Remove finalizer
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			latestKode, err := common.GetLatestKode(ctx, r.Client, kode.Name, kode.Namespace)
			if err != nil {
				return err
			}

			if controllerutil.ContainsFinalizer(latestKode, common.KodeFinalizerName) {
				controllerutil.RemoveFinalizer(latestKode, common.KodeFinalizerName)
				log.Info("Removing finalizer", "finalizer", common.KodeFinalizerName)
				return r.Client.Update(ctx, latestKode)
			}
			return nil
		})

		if err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	} else {
		log.Info("Finalizer not found, skipping finalization", "finalizer", common.KodeFinalizerName)
	}

	// Stop reconciliation
	return ctrl.Result{}, nil
}

func (r *KodeReconciler) finalize(ctx context.Context, kode *kodev1alpha2.Kode) error {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))

	if kode == nil {
		log.Info("Kode is nil, skipping finalization")
		return nil
	}

	cleanupResource := NewKodeCleanupResource(kode)

	// Perform cleanup
	err := r.CleanupManager.Cleanup(ctx, cleanupResource)
	if err != nil {
		log.Error(err, "Failed to cleanup resources")
		return err
	}

	return nil
}
