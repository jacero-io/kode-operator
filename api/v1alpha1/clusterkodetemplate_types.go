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

// ClusterKodeTemplateSpec defines the desired state of ClusterKodeTemplate
type ClusterKodeTemplateSpec struct {
	SharedKodeTemplateSpec `json:",inline"`
}

// ClusterKodeTemplateStatus defines the observed state of ClusterKodeTemplate
type ClusterKodeTemplateStatus struct {
	SharedKodeTemplateStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterKodeTemplate is the Schema for the kodetemplates API
type ClusterKodeTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KodeTemplateSpec   `json:"spec,omitempty"`
	Status KodeTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterKodeTemplateList contains a list of ClusterKodeTemplate
type ClusterKodeTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterKodeTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterKodeTemplate{}, &ClusterKodeTemplateList{})
}
