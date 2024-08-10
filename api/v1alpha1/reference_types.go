/*
Copyright emil@jacero.se 2024.

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

package v1alpha1

type CrossNamespaceObjectReference struct {
	// Kind is the resource kind.
	// +kubebuilder:validation:Description="Resource kind"
	Kind string `json:"kind"`

	// Name is the name of the resource.
	// +kubebuilder:validation:Description="Name of the resource"
	Name string `json:"name"`

	// Namespace is the namespace of the resource.
	// +kubebuilder:validation:Description="Namespace of the resource"
	Namespace string `json:"namespace,omitempty"`
}

// TODO: Add in code validation for the kind field
// KodeTemplateReference is a reference to a KodeTemplate or KodeClusterTemplate
type KodeTemplateReference CrossNamespaceObjectReference

// TODO: Add in code validation for the kind field
// EnvoyConfigReference is a reference to an EnvoyConfig or EnvoyProxyClusterConfig
type EnvoyConfigReference CrossNamespaceObjectReference

// EntryPointReference is a reference to an EntryPoint
type EntryPointReference CrossNamespaceObjectReference
