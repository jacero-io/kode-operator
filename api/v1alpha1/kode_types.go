/*
Copyright emil@jacero.se 2024.

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

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// TemplateRef is the reference to the KodeTemplate configuration.
	// +kubebuilder:validation:Description="Reference to the KodeTemplate configuration."
	// +kubebuilder:validation:Required
	TemplateRef KodeTemplateReference `json:"templateRef"`

	// Username is both the the HTTP Basic auth username (when used) and the user the container should run as, abc is default.
	// +kubebuilder:validation:Description="Is both the the HTTP Basic auth username (when used) and the user the container should run as. Defaults to 'abc'."
	// +kubebuilder:default="abc"
	Username string `json:"username,omitempty"`

	// Password HTTP Basic auth password. If unset there will be no auth.
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
	// ReconcileAttempts is the number of times the Kode resource has been reconciled.
	ReconcileAttempts int `json:"reconcileAttempts,omitempty"`

	// Phase represents the current phase of the Kode resource.
	Phase KodePhase `json:"phase"`

	// Conditions represent the latest available observations of a Kode's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

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

type KodeTemplateReference struct {
	// Kind is the resource kind.
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=KodeTemplate;KodeClusterTemplate
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate.
	// +kubebuilder:validation:Description="Name of the KodeTemplate"
	Name string `json:"name"`

	// Namespace is the namespace of the KodeTemplate.
	// +kubebuilder:validation:Description="Namespace of the KodeTemplate"
	Namespace string `json:"namespace,omitempty"`
}

type InitPluginSpec struct {
	// Name is the name of the container.
	// +kubebuilder:validation:Description="Name of the container."
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Image is the OCI image for the container.
	// +kubebuilder:validation:Description="OCI image for the container."
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Command is the command to run in the container.
	// +kubebuilder:validation:Description="Command to run in the container."
	Command []string `json:"command,omitempty"`

	// Args are the arguments to the container.
	// +kubebuilder:validation:Description="Arguments to the container."
	Args []string `json:"args,omitempty"`

	// Env are the environment variables to the container.
	// +kubebuilder:validation:Description="Environment variables to the container."
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom are the environment variables taken from a Secret or ConfigMap.
	// +kubebuilder:validation:Description="Environment variables taken from a Secret or ConfigMap."
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// VolumeMounts are the volume mounts for the container. Can be used to mount a ConfigMap or Secret.
	// +kubebuilder:validation:Description="Volume mounts for the container. Can be used to mount a ConfigMap or Secret."
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

func (s KodeStorageSpec) IsEmpty() bool {
	return len(s.AccessModes) == 0 &&
		s.StorageClassName == nil &&
		(s.Resources.Requests == nil || s.Resources.Requests.Storage().IsZero())
}

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}
