/*
Copyright 2024.

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

// EnvoyProxyTemplateSpec defines the desired state of EnvoyProxyTemplate
type SharedEnvoyProxyTemplateSpec struct {
	// Image is the Docker image for the Envoy proxy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="envoyproxy/envoy:v1.30-latest"
	Image string `json:"image"`

	// HTTPFilters is a list of Envoy HTTP filters to be applied
	// +kubebuilder:validation:MinItems=1
	HTTPFilters []HTTPFilter `json:"httpFilters"`
}

// EnvoyProxyTemplateStatus defines the observed state of EnvoyProxyTemplate
type SharedEnvoyProxyTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// EnvoyProxyTemplateReference is a reference to an EnvoyProxyTemplate or ClusterEnvoyProxyTemplate
type EnvoyProxyTemplateReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=EnvoyProxyTemplate;ClusterEnvoyProxyTemplate
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name is the name of the EnvoyProxyTemplate or ClusterEnvoyProxyTemplate
	// +kubebuilder:validation:Description="Name of the template"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}
