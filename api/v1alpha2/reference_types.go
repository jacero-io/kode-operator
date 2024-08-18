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

package v1alpha2

type CrossNamespaceObjectReference struct {
	// API version of the referent.
	// +kubebuilder:validation:Description="API version of the referent."
	// +kubebuilder:validation:Optional
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Kind is the resource kind.
	// +kubebuilder:validation:Description="Resource kind"
	Kind Kind `json:"kind" yaml:"kind"`

	// Name is the name of the resource.
	// +kubebuilder:validation:Description="Name of the resource"
	Name ObjectName `json:"name" yaml:"name"`

	// Namespace is the namespace of the resource.
	// +kubebuilder:validation:Description="Namespace of the resource"
	// +kubebuilder:validation:Optional
	Namespace Namespace `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}
