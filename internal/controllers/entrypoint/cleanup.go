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
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1"
)

type EntryPointCleanupResource struct {
	EntryPoint *kodev1alpha2.EntryPoint
}

func (e *EntryPointCleanupResource) GetResources() []cleanup.Resource {
	resources := []cleanup.Resource{}

	// Add HTTPRoute resource
	resources = append(resources, cleanup.Resource{
		Name:      e.EntryPoint.Name,
		Namespace: e.EntryPoint.Namespace,
		Kind:      "HTTPRoute",
		Object:    &gatewayv1beta1.HTTPRoute{},
	})

	return resources
}

func (e *EntryPointCleanupResource) ShouldDelete(resource cleanup.Resource) bool {
	// For EntryPoint, we typically want to delete all associated resources
	return true
}

func NewEntryPointCleanupResource(entryPoint *kodev1alpha2.EntryPoint) cleanup.CleanupableResource {
	return &EntryPointCleanupResource{EntryPoint: entryPoint}
}
