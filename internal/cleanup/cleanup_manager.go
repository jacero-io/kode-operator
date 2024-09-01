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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resource represents a generic Kubernetes resource
type Resource struct {
	Kind      string
	Name      string
	Namespace string
	Object    client.Object
}

// CleanupableResource defines the interface for resources that can be cleaned up
type CleanupableResource interface {
	GetResources() []Resource
	ShouldDelete(resource Resource) bool
}

// CleanupManager defines the interface for cleaning up resources
type CleanupManager interface {
	Cleanup(ctx context.Context, resource CleanupableResource) error
}

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

func (m *defaultCleanupManager) Cleanup(ctx context.Context, resource CleanupableResource) error {
	log := m.log.WithValues("resource", resource)
	log.Info("Starting cleanup of resources")

	resources := resource.GetResources()
	var wg sync.WaitGroup

	for _, res := range resources {
		// Check if the resource should be deleted
		if resource.ShouldDelete(res) {
			wg.Add(1) // Increment the wait group counter
			go func(res Resource) {
				defer wg.Done()
				if err := m.deleteResourceAsync(ctx, res); err != nil {
					log.Error(err, fmt.Sprintf("Failed to delete %s", res.Kind))
				}
			}(res)
		} else {
			log.V(1).Info(fmt.Sprintf("Skipping deletion of %s due to condition not met", res.Kind))
		}
	}

	go func() {
		wg.Wait()
		log.V(1).Info("Resources cleanup initiated successfully")
	}()

	return nil
}

func (m *defaultCleanupManager) deleteResourceAsync(ctx context.Context, resource Resource) error {
	log := m.log.WithValues(resource.Kind, client.ObjectKey{Name: resource.Name, Namespace: resource.Namespace})

	log.V(1).Info("Attempting to delete resource")

	// Set the name and namespace
	resource.Object.SetName(resource.Name)
	resource.Object.SetNamespace(resource.Namespace)

	// Attempt to delete the resource
	if err := m.client.Delete(ctx, resource.Object); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s not found, considering it deleted", resource.Kind))
			return nil
		}
		log.Error(err, fmt.Sprintf("Failed to delete %s", resource.Kind))
		return fmt.Errorf("failed to delete %s: %w", resource.Kind, err)
	}

	// Start a goroutine to periodically check the deletion status
	go m.checkDeletionStatus(ctx, resource.Object, resource.Kind)

	return nil
}

func (m *defaultCleanupManager) checkDeletionStatus(ctx context.Context, obj client.Object, kind string) {
	log := m.log.WithValues(kind, client.ObjectKeyFromObject(obj))

	err := wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		err := m.client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s confirmed deleted", kind))
			return true, nil
		}
		if err != nil {
			log.Error(err, fmt.Sprintf("Error checking %s deletion status", kind))
			return false, err
		}
		log.V(1).Info(fmt.Sprintf("%s still exists, waiting for deletion", kind))
		return false, nil
	})

	if err != nil {
		if err == context.DeadlineExceeded {
			log.Error(err, fmt.Sprintf("Timeout waiting for %s deletion", kind))
		} else if err == context.Canceled {
			log.Info(fmt.Sprintf("Context canceled while waiting for %s deletion", kind))
		} else {
			log.Error(err, fmt.Sprintf("Error occurred while waiting for %s deletion", kind))
		}
	}
}
