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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// Image is the Docker image for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="lscr.io/linuxserver/code-server:latest"
	Image string `json:"image"`

	// ServicePort is the port for the code-server service
	// +kubebuilder:validation:Minimum=1
	ServicePort int32 `json:"servicePort"`

	// Password is the password for code-server
	// +kubebuilder:validation:MinLength=1
	Password string `json:"password"`

	// PUID is the user ID for the code-server process
	// +kubebuilder:default=1000
	PUID int `json:"puid,omitempty"`

	// PGID is the group ID for the code-server process
	// +kubebuilder:default=1000
	PGID int `json:"pgid,omitempty"`

	// TZ is the timezone for the code-server process
	// +kubebuilder:default="Etc/UTC"
	TZ string `json:"tz,omitempty"`

	// HashedPassword is the hashed password for code-server
	HashedPassword string `json:"hashedPassword,omitempty"`

	// SudoPassword is the sudo password for code-server
	SudoPassword string `json:"sudoPassword,omitempty"`

	// SudoPasswordHash is the hashed sudo password for code-server
	SudoPasswordHash string `json:"sudoPasswordHash,omitempty"`

	// ProxyDomain is the proxy domain for code-server
	ProxyDomain string `json:"proxyDomain,omitempty"`

	// DefaultWorkspace is the default workspace directory for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default=/config/workspace
	DefaultWorkspace string `json:"defaultWorkspace,omitempty"`

	// ConfigPath is the path to the config directory for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default=/config
	ConfigPath string `json:"configPath,omitempty"`

	// Storage specifies the storage configuration for code-server
	Storage *StorageSpec `json:"storage,omitempty"`
}

// StorageSpec defines the desired state of storage for Kode
type StorageSpec struct {
	// Name is the name of the PersistentVolumeClaim for code-server
	Name string `json:"name"`

	// StorageClassName is the storage class name for the PVC
	StorageClassName string `json:"storageClassName,omitempty"`

	// Size is the storage request size for the PVC
	// +kubebuilder:validation:MinLength=1
	Size string `json:"size,omitempty"`

	// ExistingPV is the name of an existing PersistentVolume to use
	ExistingPV string `json:"existingPV,omitempty"`
}

// KodeStatus defines the observed state of Kode
type KodeStatus struct {
	// +kubebuilder:validation:Minimum=0
	AvailableReplicas int32 `json:"availableReplicas"`
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
