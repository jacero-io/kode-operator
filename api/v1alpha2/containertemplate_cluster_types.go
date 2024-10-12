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

// ClusterContainerTemplateSpec defines the desired state of ClusterKodeContainer
type ClusterContainerTemplateSpec struct {
	ContainerTemplateSharedSpec `json:",inline" yaml:",inline"`
}

// ClusterContainerTemplateStatus defines the observed state of ClusterKodeContainer
type ClusterContainerTemplateStatus struct {
	ContainerTemplateSharedStatus `json:",inline" yaml:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterContainerTemplate is the Schema for the clusterContainerTemplates API
type ClusterContainerTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterContainerTemplateSpec   `json:"spec,omitempty"`
	Status ClusterContainerTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterContainerTemplateList contains a list of ClusterContainerTemplate
type ClusterContainerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterContainerTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterContainerTemplate{}, &ClusterContainerTemplateList{})
}
