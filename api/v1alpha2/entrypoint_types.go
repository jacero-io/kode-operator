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

// EntryPointSpec defines the desired state of EntryPoint
type EntryPointSpec struct {
	EntryPointSharedSpec `json:",inline"`
}

// EntryPointStatus defines the observed state of EntryPoint
type EntryPointStatus struct {
	EntryPointSharedStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EntryPoint is the Schema for the entrypoints API
type EntryPoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EntryPointSpec   `json:"spec,omitempty"`
	Status EntryPointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EntryPointList contains a list of EntryPoint
type EntryPointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EntryPoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EntryPoint{}, &EntryPointList{})
}
