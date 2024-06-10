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

type KodeTemplateReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=KodeTemplate;ClusterKodeTemplate
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate
	// +kubebuilder:validation:Description="Name of the KodeTemplate"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the KodeTemplate
	// +kubebuilder:validation:Description="Namespace of the KodeTemplate"
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace,omitempty"`
}

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// TemplateRef is the reference to the KodeTemplate configuration
	// +kubebuilder:validation:Description="Reference to the KodeTemplate configuration."
	// +kubebuilder:validation:Required
	TemplateRef KodeTemplateReference `json:"templateRef"`

	// User is the HTTP Basic auth username or the user the container should run as, abc is default.
	// +kubebuilder:validation:Description="User is the HTTP Basic auth username or the user the container should run as. Defaults to 'abc'."
	// +kubebuilder:default="abc"
	User string `json:"user,omitempty"`

	// Password HTTP Basic auth password. If unset there will be no auth
	// +kubebuilder:validation:Description="HTTP Basic auth password. If unset, there will be no authentication."
	Password string `json:"password,omitempty"`

	// ExistingSecret is a reference to an existing secret containing user and password.
	// +kubebuilder:validation:Description="ExistingSecret is a reference to an existing secret containing user and password. If set, User and Password fields are ignored."
	ExistingSecret string `json:"existingSecret,omitempty"`

	// Home is the path to the directory for the user data
	// +kubebuilder:validation:Description="Home is the path to the directory for the user data. Defaults to '/config'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	Home string `json:"home,omitempty"`

	// Workspace is the user specified workspace directory (e.g. my-workspace)
	// +kubebuilder:validation:Description="User specified workspace directory (e.g. my-workspace)."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	Workspace string `json:"workspace,omitempty"`

	// Storage specifies the storage configuration
	// +kubebuilder:validation:Description="Storage configuration."
	Storage KodeStorageSpec `json:"storage,omitempty"`
}

// KodeStorageSpec defines the storage configuration
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
