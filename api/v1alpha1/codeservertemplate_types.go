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

// CodeServerTemplateSpec defines the desired state of CodeServerTemplate
type CodeServerTemplateSpec struct {
	// EnvoyProxyRef is the reference to the EnvoyProxy configuration
	EnvoyProxyTemplateRef EnvoyProxyTemplateReference `json:"envoyProxyTemplateRef,omitempty"`

	// Image is the container image for code-server
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="lscr.io/linuxserver/code-server:latest"
	Image string `json:"image"`

	// Port is the port for the code-server process. Used by EnvoyProxy to expose the service.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3000
	Port int32 `json:"port,omitempty"`

	// Specifies the envs
	Envs []string `json:"envs,omitempty" protobuf:"bytes,9,opt,name=envs"`

	// Specifies the envs
	Args []string `json:"args,omitempty" protobuf:"bytes,10,opt,name=args"`

	// TZ is the timezone for the code-server process
	// +kubebuilder:default="Europe/Stockholm"
	TZ string `json:"tz,omitempty"`

	// PUID is the user ID for the code-server process
	// +kubebuilder:default=1000
	PUID int64 `json:"puid,omitempty"`

	// PGID is the group ID for the code-server process
	// +kubebuilder:default=1000
	PGID int64 `json:"pgid,omitempty"`

	// ProxyDomain specifies the domain to proxy for code-server
	// +kubebuilder:validation:Description="If this optional variable is set, this domain will be proxied for subdomain proxying."
	// +kubebuilder:validation:MinLength=1
	ProxyDomain string `json:"proxyDomain,omitempty"`

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
}

// CodeServerTemplateStatus defines the observed state of CodeServerTemplate
type CodeServerTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CodeServerTemplate is the Schema for the codeservertemplates API
type CodeServerTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeServerTemplateSpec   `json:"spec,omitempty"`
	Status CodeServerTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CodeServerTemplateList contains a list of CodeServerTemplate
type CodeServerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeServerTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeServerTemplate{}, &CodeServerTemplateList{})
}
