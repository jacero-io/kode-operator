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

// TofuTemplateSpec defines the desired state of KodeTofu
type TofuTemplateSpec struct {
	TofuSharedSpec `json:",inline"`
}

// TofuTemplateStatus defines the observed state of KodeTofu
type TofuTemplateStatus struct {
	TofuSharedStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TofuTemplate is the Schema for the tofutemplates API
type TofuTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TofuTemplateSpec   `json:"spec,omitempty"`
	Status TofuTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TofuTemplateList contains a list of TofuTemplate
type TofuTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TofuTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TofuTemplate{}, &TofuTemplateList{})
}
