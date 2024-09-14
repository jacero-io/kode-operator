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

// import (
// 	"context"

// 	"k8s.io/client-go/util/retry"
// 	ctrl "sigs.k8s.io/controller-runtime"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

// 	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
// 	"github.com/jacero-io/kode-operator/internal/common"
// 	"github.com/jacero-io/kode-operator/internal/constants"
// )

// func (r *EntryPointReconciler) handleFinalizer(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (ctrl.Result, error) {
// 	log := r.Log.WithValues("entryPoint", client.ObjectKeyFromObject(entryPoint))

// 	if entryPoint.ObjectMeta.DeletionTimestamp.IsZero() {
// 		// Object is not being deleted, so ensure the finalizer is present
// 		if !controllerutil.ContainsFinalizer(entryPoint, constants.EntryPointFinalizerName) {
// 			controllerutil.AddFinalizer(entryPoint, constants.EntryPointFinalizerName)
// 			log.Info("Adding finalizer", "finalizer", constants.EntryPointFinalizerName)
// 			if err := r.Client.Update(ctx, entryPoint); err != nil {
// 				log.Error(err, "Failed to add finalizer")
// 				return ctrl.Result{}, err
// 			}
// 		}
// 		return ctrl.Result{}, nil
// 	}

// 	// Object is being deleted
// 	if controllerutil.ContainsFinalizer(entryPoint, constants.EntryPointFinalizerName) {
// 		// Run finalization logic
// 		if err := r.finalize(ctx, entryPoint); err != nil {
// 			log.Error(err, "Failed to run finalizer")
// 			return ctrl.Result{}, err
// 		}

// 		// Remove finalizer
// 		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
// 			// Fetch the latest version of EntryPoint
// 			latestEntrypoint, err := common.GetLatestEntryPoint(ctx, r.Client, entryPoint.Name, entryPoint.Namespace)
// 			if err != nil {
// 				return err
// 			}

// 			if controllerutil.ContainsFinalizer(latestEntrypoint, constants.EntryPointFinalizerName) {
// 				controllerutil.RemoveFinalizer(latestEntrypoint, constants.EntryPointFinalizerName)
// 				log.Info("Removing finalizer", "finalizer", constants.EntryPointFinalizerName)
// 				return r.Client.Update(ctx, latestEntrypoint)
// 			}
// 			return nil
// 		})

// 		if err != nil {
// 			log.Error(err, "Failed to remove finalizer")
// 			return ctrl.Result{}, err
// 		}
// 	}

// 	return ctrl.Result{}, nil
// }

// func (r *EntryPointReconciler) finalize(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) error {
// 	log := r.Log.WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint))

// 	cleanupResource := NewEntryPointCleanupResource(entryPoint)

// 	// Perform cleanup
// 	err := r.CleanupManager.Cleanup(ctx, cleanupResource)
// 	if err != nil {
// 		log.Error(err, "Failed to cleanup resources")
// 		return err
// 	}

// 	log.Info("Finalization completed successfully")
// 	return nil
// }
