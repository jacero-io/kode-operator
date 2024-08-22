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
