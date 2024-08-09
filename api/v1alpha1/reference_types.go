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


type KodeTemplateReference struct {
	// Kind is the resource kind.
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=KodeTemplate;KodeClusterTemplate
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate.
	// +kubebuilder:validation:Description="Name of the KodeTemplate"
	Name string `json:"name"`

	// Namespace is the namespace of the KodeTemplate.
	// +kubebuilder:validation:Description="Namespace of the KodeTemplate"
	Namespace string `json:"namespace,omitempty"`
}

// EnvoyConfigReference is a reference to an EnvoyProxyConfig or EnvoyProxyClusterConfig
type EnvoyConfigReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=EnvoyProxyConfig;EnvoyProxyClusterConfig
	// +kubebuilder:validation:Optional
	Kind string `json:"kind,omitempty"`

	// Name is the name of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Name of the config"
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// Namespace is the namespace of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Namespace of the Envoy Proxy config"
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}
