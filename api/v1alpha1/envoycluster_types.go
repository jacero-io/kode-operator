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

// ClusterLoadAssignment specifies the cluster load assignment configuration
type ClusterLoadAssignment struct {
	// ClusterName is the name of the cluster load assignment
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClusterName string `json:"clusterName"`

	// Endpoints is a list of endpoints for the cluster
	// +kubebuilder:validation:MinItems=1
	Endpoints []Endpoint `json:"endpoints"`
}

// Endpoint specifies a single endpoint configuration
type Endpoint struct {
	// Address specifies the address of the endpoint
	Address string `json:"address"`

	// Port specifies the port of the endpoint
	Port int32 `json:"port"`
}

// EnvoyClusterSpec defines the desired state of EnvoyCluster
type EnvoyClusterSpec struct {
	// Name of the cluster
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ConnectTimeout specifies the timeout for establishing new connections
	// +kubebuilder:validation:Pattern=`^([0-9]+(s|ms|us|ns))$`
	// +kubebuilder:validation:Required
	ConnectTimeout string `json:"connectTimeout"`

	// Type specifies the type of the cluster (e.g., STATIC, STRICT_DNS)
	// +kubebuilder:validation:Enum=STATIC;STRICT_DNS;LOGICAL_DNS;EDS;ORIGINAL_DST;RELOADABLE
	Type string `json:"type"`

	// LoadAssignment specifies the cluster load assignment
	LoadAssignment ClusterLoadAssignment `json:"loadAssignment"`
}

// EnvoyClusterStatus defines the observed state of EnvoyCluster
type EnvoyClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EnvoyCluster is the Schema for the envoyclusters API
type EnvoyCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyClusterSpec   `json:"spec,omitempty"`
	Status EnvoyClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyClusterList contains a list of EnvoyCluster
type EnvoyClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyCluster{}, &EnvoyClusterList{})
}
