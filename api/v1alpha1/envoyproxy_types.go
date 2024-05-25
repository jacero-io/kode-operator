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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvoyProxySpec defines the desired state of EnvoyProxy
type EnvoyProxySpec struct {
	// Image is the Docker image for the Envoy proxy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="envoyproxy/envoy:v1.30-latest"
	Image string `json:"image"`

	// HTTPFilters is a list of Envoy HTTP filters to be applied
	// +kubebuilder:validation:MinItems=1
	HTTPFilters []HTTPFilter `json:"httpFilters"`

	// Clusters is a list of references to EnvoyCluster objects
	// +kubebuilder:validation:MinItems=1
	Clusters []EnvoyClusterRef `json:"clusters"`
}

// EnvoyClusterRef represents a reference to an EnvoyCluster
type EnvoyClusterRef struct {
	// Name of the EnvoyCluster
	Name string `json:"name"`

	// Namespace of the EnvoyCluster
	Namespace string `json:"namespace"`
}

// EnvoyProxyStatus defines the observed state of EnvoyProxy
type EnvoyProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EnvoyProxy is the Schema for the envoyproxies API
type EnvoyProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyProxySpec   `json:"spec,omitempty"`
	Status EnvoyProxyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyProxyList contains a list of EnvoyProxy
type EnvoyProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyProxy{}, &EnvoyProxyList{})
}
