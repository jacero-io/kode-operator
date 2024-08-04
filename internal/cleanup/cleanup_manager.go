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

package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultCleanupManager struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultCleanupManager(client client.Client, log logr.Logger) CleanupManager {
	return &defaultCleanupManager{
		client: client,
		log:    log,
	}
}

func (m *defaultCleanupManager) Cleanup(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := m.log.WithValues("kode", common.ObjectKeyFromConfig(config))
	log.Info("Starting cleanup of Kode resources", "Config", config)

	resourcesForCleanup := []struct {
		name      string
		obj       client.Object
		condition func() bool
	}{
		{
			name: "StatefulSet",
			obj: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.StatefulSetName,
					Namespace: config.KodeNamespace,
				},
			},
			condition: func() bool {
				shouldCleanup := true
				log.V(1).Info("StatefulSet cleanup condition", "ShouldCleanup", shouldCleanup, "StatefulSetName", config.StatefulSetName)
				return shouldCleanup
			},
		},
		{
			name: "Service",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.ServiceName,
					Namespace: config.KodeNamespace,
				},
			},
			condition: func() bool {
				shouldCleanup := true
				log.V(1).Info("Service cleanup condition", "ShouldCleanup", shouldCleanup, "ServiceName", config.ServiceName)
				return shouldCleanup
			},
		},
		{
			name: "PersistentVolumeClaim",
			obj: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.PVCName,
					Namespace: config.KodeNamespace,
				},
			},
			condition: func() bool {
				// When KodeSpec.Storage is not empty and KeepVolume is not set or set to false, it should be cleaned up
				shouldCleanup := !config.KodeSpec.DeepCopy().Storage.IsEmpty() &&
					(config.KodeSpec.Storage.KeepVolume == nil || !*config.KodeSpec.Storage.KeepVolume)
				log.V(1).Info("PVC cleanup condition", "ShouldCleanup", shouldCleanup, "PVCName", config.PVCName)
				return shouldCleanup
			},
		},
	}

	var wg sync.WaitGroup
	for _, res := range resourcesForCleanup {
		if res.condition() {
			wg.Add(1)
			go func(res struct {
				name      string
				obj       client.Object
				condition func() bool
			}) {
				defer wg.Done()
				if err := m.deleteResourceAsync(ctx, res.obj, res.name); err != nil {
					log.Error(err, fmt.Sprintf("Failed to delete %s", res.name))
				}
			}(res)
		} else {
			log.V(1).Info(fmt.Sprintf("Skipping deletion of %s due to condition not met", res.name))
		}
	}

	// Start a goroutine to wait for all deletions to complete
	go func() {
		wg.Wait()
		log.V(1).Info("Kode resources cleanup initiated successfully")
	}()

	return nil
}

func (m *defaultCleanupManager) Recycle(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := m.log.WithValues("kode", common.ObjectKeyFromConfig(config))
	log.Info("Starting recycle of Kode resources")

	log.Info("Kode resources recycled successfully")
	return nil
}

func (m *defaultCleanupManager) deleteResourceAsync(ctx context.Context, obj client.Object, resourceType string) error {
	log := m.log.WithValues(resourceType, client.ObjectKeyFromObject(obj))

	log.V(1).Info("Attempting to delete resource", "Name", obj.GetName())

	// Attempt to delete the resource
	if err := m.client.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s not found, considering it deleted", resourceType))
			return nil
		}
		log.Error(err, fmt.Sprintf("Failed to delete %s", resourceType))
		return fmt.Errorf("failed to delete %s: %w", resourceType, err)
	}

	// Start a goroutine to periodically check the deletion status
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.V(1).Info(fmt.Sprintf("Context cancelled, stopping %s deletion check", resourceType))
				return
			case <-ticker.C:
				err := m.client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
				if errors.IsNotFound(err) {
					log.Info(fmt.Sprintf("%s confirmed deleted", resourceType))
					return
				}
				if err != nil {
					log.Error(err, fmt.Sprintf("Error checking %s deletion status", resourceType))
					return
				}
				log.V(1).Info(fmt.Sprintf("%s still exists, waiting for deletion", resourceType))
			}
		}
	}()

	return nil
}
