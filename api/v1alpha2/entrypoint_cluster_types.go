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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterEntryPointSpec defines the desired state of ClusterEntryPoint
type ClusterEntryPointSpec struct {
	EntryPointSharedSpec `json:",inline"`
}

// ClusterEntryPointStatus defines the observed state of ClusterEntryPoint
type ClusterEntryPointStatus struct {
	EntryPointSharedStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterEntryPoint is the Schema for the clusterentrypoints API
type ClusterEntryPoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterEntryPointSpec   `json:"spec,omitempty"`
	Status ClusterEntryPointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterEntryPointList contains a list of ClusterEntryPoint
type ClusterEntryPointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterEntryPoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterEntryPoint{}, &ClusterEntryPointList{})
}
