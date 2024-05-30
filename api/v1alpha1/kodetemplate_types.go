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

type KodeTemplateReference struct {
	// Kind is the kind of the reference (for KodeTemplate)
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate
	Name string `json:"name"`
}

// KodeTemplateSpec defines the desired state of KodeTemplate
type KodeTemplateSpec struct {
	// Image is the Docker image for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="lscr.io/linuxserver/code-server:latest"
	Image string `json:"image"`

	// TZ is the timezone for the code-server process
	// +kubebuilder:default="Europe/Stockholm"
	TZ string `json:"tz,omitempty"`

	// PUID is the user ID for the code-server process
	// +kubebuilder:default=1000
	PUID int64 `json:"puid,omitempty"`

	// PGID is the group ID for the code-server process
	// +kubebuilder:default=1000
	PGID int64 `json:"pgid,omitempty"`

	// URL specifies the url for used to access the code-server
	URL string `json:"url,omitempty" protobuf:"bytes,7,opt,name=url"`

	// ServicePort is the port for the code-server service
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=8443
	ServicePort int32 `json:"servicePort,omitempty"`

	// Specifies the envs
	Envs []string `json:"envs,omitempty" protobuf:"bytes,9,opt,name=envs"`

	// Specifies the envs
	Args []string `json:"args,omitempty" protobuf:"bytes,10,opt,name=args"`

	// Password is the password for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Password string `json:"password"`

	// HashedPassword is the hashed password for code-server
	HashedPassword string `json:"hashedPassword,omitempty"`

	// SudoPassword is the sudo password for code-server
	SudoPassword string `json:"sudoPassword,omitempty"`

	// SudoPasswordHash is the hashed sudo password for code-server
	SudoPasswordHash string `json:"sudoPasswordHash,omitempty"`

	// ConfigPath is the path to the config directory for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default=/config
	ConfigPath string `json:"configPath,omitempty"`

	// DefaultWorkspace is the default workspace directory for code-server (eg. /config/workspace)
	// +kubebuilder:validation:MinLength=1
	DefaultWorkspace string `json:"defaultWorkspace,omitempty"`

	// EnvoyProxyRef is the reference to the EnvoyProxy configuration
	EnvoyProxyTemplateRef EnvoyProxyTemplateReference `json:"envoyProxyTemplateRef"`
}

// KodeTemplateStatus defines the observed state of KodeTemplate
type KodeTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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
