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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// TemplateRef is the reference to the KodeTemplate configuration.
	// +kubebuilder:validation:Description="Reference to the KodeTemplate configuration."
	// +kubebuilder:validation:Required
	TemplateRef KodeTemplateReference `json:"templateRef"`

	// Credentials specifies the credentials for the service.
	// +kubebuilder:validation:Description="Credentials for the service."
	Credentials CredentialsSpec `json:"credentials,omitempty"`

	// Home is the path to the directory for the user data
	// +kubebuilder:validation:Description="Home is the path to the directory for the user data. Defaults to '/config'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	Home string `json:"home,omitempty"`

	// Workspace is the user specified workspace directory (e.g. my-workspace).
	// +kubebuilder:validation:Description="User specified workspace directory (e.g. my-workspace)."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	Workspace string `json:"workspace,omitempty"`

	// Storage specifies the storage configuration.
	// +kubebuilder:validation:Description="Storage configuration."
	Storage KodeStorageSpec `json:"storage,omitempty"`

	// UserConfig specifies a git repository URL to get user configuration from.
	// +kubebuilder:validation:Description="Git repository URL to get user configuration from."
	UserConfig string `json:"userConfig,omitempty"`

	// Privileged specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing.
	// +kubebuilder:validation:Description="Specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing."
	// +kubebuilder:default=false
	Privileged *bool `json:"privileged,omitempty"`

	// InitPlugins specifies the OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list.
	// +kubebuilder:validation:Description="OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list."
	InitPlugins []InitPluginSpec `json:"initPlugins,omitempty"`
}

// KodeStorageSpec defines the storage configuration
type KodeStorageSpec struct {
	// AccessModes specifies the access modes for the persistent volume.
	// +kubebuilder:validation:Description="Access modes for the persistent volume."
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// StorageClassName specifies the storage class name for the persistent volume.
	// +kubebuilder:validation:Description="Storage class name for the persistent volume."
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Resources specifies the resource requirements for the persistent volume.
	// +kubebuilder:validation:Description="Resource requirements for the persistent volume."
	Resources corev1.VolumeResourceRequirements `json:"resources,omitempty"`

	// KeepVolume specifies if the volume should be kept when the kode is recycled. Defaults to false.
	// +kubebuilder:validation:Description="Specifies if the volume should be kept when the kode is recycled. Defaults to false."
	// +kubebuilder:default=false
	KeepVolume *bool `json:"keepVolume,omitempty"`

	// ExistingVolumeClaim specifies an existing PersistentVolumeClaim to use.
	// +kubebuilder:validation:Description="Specifies an existing PersistentVolumeClaim to use."
	ExistingVolumeClaim string `json:"existingVolumeClaim,omitempty"`
}

// KodePhase represents the current phase of the Kode resource.
type KodePhase string

const (
	// KodePhaseCreating indicates that the Kode resource is in the process of being created.
	// This includes the initial setup and creation of associated Kubernetes resources.
	KodePhaseCreating KodePhase = "Creating"

	// KodePhaseCreated indicates that all the necessary Kubernetes resources for the Kode
	// have been successfully created, but may not yet be fully operational.
	KodePhaseCreated KodePhase = "Created"

	// KodePhaseFailed indicates that an error occurred during the creation, updating,
	// or management of the Kode resource or its associated Kubernetes resources.
	KodePhaseFailed KodePhase = "Failed"

	// KodePhasePending indicates that the Kode resource has been created, but is waiting
	// for its associated resources to become ready or for external dependencies to be met.
	KodePhasePending KodePhase = "Pending"

	// KodePhaseActive indicates that the Kode resource and all its associated Kubernetes
	// resources are fully operational and ready to serve requests.
	KodePhaseActive KodePhase = "Active"

	// KodePhaseInactive indicates that the Kode resource has been marked for deletion
	// due to inactivity. Resources may be partially or fully removed in this state.
	KodePhaseInactive KodePhase = "Inactive"

	// KodePhaseRecycling indicates that the Kode resource is in the process of being
	// cleaned up and its resources are being partially or fully deleted.
	KodePhaseRecycling KodePhase = "Recycling"

	// KodePhaseRecycled indicates that the Kode resource has been fully recycled,
	// with all associated resources either partially or fully deleted.
	KodePhaseRecycled KodePhase = "Recycled"
)

// KodeStatus defines the observed state of Kode
type KodeStatus struct {
	// ObservedGeneration is the last observed generation of the Kode resource.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the Kode resource.
	Phase KodePhase `json:"phase"`

	// Conditions represent the latest available observations of a Kode's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// KodeUrl is the URL to access the Kode.
	KodeUrl string `json:"kodeUrl,omitempty"`

	// IconUrl is the URL to the icon for the Kode.
	IconUrl string `json:"iconUrl,omitempty"`

	// Runtime is the runtime for the kode. Can be one of 'container', 'virtual', 'tofu'.
	Runtime string `json:"runtime,omitempty"`

	// LastActivityTime is the timestamp when the last activity occurred.
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// LastError contains the last error message encountered during reconciliation.
	LastError string `json:"lastError,omitempty"`

	// LastErrorTime is the timestamp when the last error occurred.
	LastErrorTime *metav1.Time `json:"lastErrorTime,omitempty"`
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

func (s KodeStorageSpec) IsEmpty() bool {
	return len(s.AccessModes) == 0 &&
		s.StorageClassName == nil &&
		(s.Resources.Requests == nil || s.Resources.Requests.Storage().IsZero())
}

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}
