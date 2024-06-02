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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WebtopTemplateKind = "WebtopTemplate"
	CodeServerTemplateKind = "CodeServerTemplate"
)

type KodeTemplateReference struct {
	// Kind is the kind of the reference (for KodeTemplate)
	// +kubebuilder:validation:Enum=WebtopTemplate;CodeServerTemplate
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate
	Name string `json:"name"`
}

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// TemplateRef is the reference to the KodeTemplate configuration
	TemplateRef KodeTemplateReference `json:"templateRef"`

	// Storage specifies the storage configuration for code-server
	Storage KodeStorageSpec `json:"storage,omitempty"`
}

// KodeStorageSpec defines the storage configuration for code-server
type KodeStorageSpec struct {
	// AccessModes specifies the access modes for the persistent volume
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// StorageClassName specifies the storage class name for the persistent volume
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Resources specifies the resource requirements for the persistent volume
	Resources corev1.VolumeResourceRequirements `json:"resources,omitempty"`
}

// KodeStatus defines the observed state of Kode
type KodeStatus struct {
    Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Kode is the Schema for the kodes API
type Kode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KodeSpec   `json:"spec,omitempty"`
	Status KodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KodeList contains a list of Kode
type KodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}
