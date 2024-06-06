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

// KodeTemplateSpec defines the desired state of KodeTemplate
type KodeTemplateSpec struct {
	// EnvoyProxyTemplateRef is the reference to the EnvoyProxy configuration
	// +kubebuilder:validation:Description="Reference to the EnvoyProxy configuration."
	EnvoyProxyTemplateRef EnvoyProxyTemplateReference `json:"envoyProxyTemplateRef,omitempty"`

	// Type is the type of container to use. Can be one of 'code-server', 'webtop', 'devcontainers', 'jupyter'.
	// +kubebuilder:validation:Description="Type of container to use. Can be one of 'code-server', 'webtop', 'devcontainers', 'jupyter'."
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=code-server;webtop;devcontainers;jupyter
	Type string `json:"type"`

	// Image is the container image for the service
	// +kubebuilder:validation:Description="Container image for the service."
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Port is the port for the service process. Used by EnvoyProxy to expose the service.
	// +kubebuilder:validation:Description="Port for the service process. Used by EnvoyProxy to expose the service. Defaults to '3000'."
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3000
	Port int32 `json:"port,omitempty"`

	// Specifies the envs
	// +kubebuilder:validation:Description="Environment variables for the service."
	Envs []corev1.EnvVar `json:"envs,omitempty"`

	// Specifies the args
	// +kubebuilder:validation:Description="Arguments for the service."
	Args []string `json:"args,omitempty"`

	// TZ is the timezone for the service process
	// +kubebuilder:validation:Description="Timezone for the service process. Defaults to 'UTC'."
	// +kubebuilder:default="UTC"
	TZ string `json:"tz,omitempty"`

	// PUID is the user ID for the service process
	// +kubebuilder:validation:Description="User ID for the service process. Defaults to '1000'."
	// +kubebuilder:default=1000
	PUID int64 `json:"puid,omitempty"`

	// PGID is the group ID for the service process
	// +kubebuilder:validation:Description="Group ID for the service process. Defaults to '1000'."
	// +kubebuilder:default=1000
	PGID int64 `json:"pgid,omitempty"`

	// Home is the path to the directory for the user data
	// +kubebuilder:validation:Description="Home is the path to the directory for the user data. Defaults to '/config'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	Home string `json:"home,omitempty"`

	// DefaultWorkspace is the default workspace directory. Defaults to 'workspace'.
	// +kubebuilder:validation:Description="Default workspace directory. Defaults to 'workspace'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	// +kubebuilder:default=workspace
	DefaultWorkspace string `json:"defaultWorkspace,omitempty"`
}

// KodeTemplateStatus defines the observed state of KodeTemplate
type KodeTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// KodeTemplate is the Schema for the kodetemplates API
type KodeTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KodeTemplateSpec   `json:"spec,omitempty"`
	Status KodeTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KodeTemplateList contains a list of KodeTemplate
type KodeTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KodeTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KodeTemplate{}, &KodeTemplateList{})
}
