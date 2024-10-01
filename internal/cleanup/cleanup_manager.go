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
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
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
	Cleanup(ctx context.Context, resource CleanupableResource) (ctrl.Result, error)
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

func (m *defaultCleanupManager) Cleanup(ctx context.Context, resource CleanupableResource) (ctrl.Result, error) {
	log := m.log.WithValues("cleanup", "manager")
	log.Info("Starting cleanup of resources")

	resources := resource.GetResources()
	deletionInitiated := false

	for _, res := range resources {
		if resource.ShouldDelete(res) {
			if err := m.deleteResource(ctx, res); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, fmt.Sprintf("Failed to delete %s", res.Kind))
					return ctrl.Result{}, err
				}
			} else {
				deletionInitiated = true
			}
		} else {
			log.V(1).Info(fmt.Sprintf("Skipping deletion of %s due to condition not met", res.Kind))
		}
	}

	if deletionInitiated {
		log.Info("Resource deletion initiated, requeuing")
		return ctrl.Result{RequeueAfter: time.Second * 1}, nil
	}

	log.V(1).Info("No resources to delete or all resources already deleted")
	return ctrl.Result{}, nil
}

func (m *defaultCleanupManager) deleteResource(ctx context.Context, resource Resource) error {
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

	log.Info(fmt.Sprintf("%s deletion initiated", resource.Kind))
	return nil
}
