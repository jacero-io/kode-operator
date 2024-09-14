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

// ContainerTemplateSpec defines the desired state of ClusterKodeContainer
type ContainerTemplateSpec struct {
	ContainerTemplateSharedSpec `json:",inline" yaml:",inline"`
}

// ContainerTemplateStatus defines the observed state of ClusterKodeContainer
type ContainerTemplateStatus struct {
	ContainerTemplateSharedStatus `json:",inline" yaml:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ContainerTemplate is the Schema for the ContainerTemplates API
type ContainerTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerTemplateSpec   `json:"spec,omitempty"`
	Status ContainerTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContainerTemplateList contains a list of ContainerTemplate
type ContainerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerTemplate{}, &ContainerTemplateList{})
}
