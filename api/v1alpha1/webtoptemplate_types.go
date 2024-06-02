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

// WebtopTemplateSpec defines the desired state of WebtopTemplate
type WebtopTemplateSpec struct {
	// EnvoyProxyRef is the reference to the EnvoyProxy configuration
	EnvoyProxyTemplateRef EnvoyProxyTemplateReference `json:"envoyProxyTemplateRef,omitempty"`

	// Image is the container image for webtop
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="lscr.io/linuxserver/webtop:latest"
	Image string `json:"image"`

	// Port is the port for the webtop process. Used by EnvoyProxy to expose the service.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3000
	Port int32 `json:"port,omitempty"`

	// Specifies the envs
	Envs []string `json:"envs,omitempty" protobuf:"bytes,9,opt,name=envs"`

	// Specifies the args
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

	// CustomUser HTTP Basic auth username, abc is default.
	// +kubebuilder:default="abc"
	CustomUser string `json:"customUser,omitempty"`

	// Password HTTP Basic auth password, abc is default. If unset there will be no auth
	Password string `json:"password,omitempty"`

	// Subfolder Subfolder for the application if running a subfolder reverse proxy, need both slashes IE /subfolder/
	Subfolder string `json:"subfolder,omitempty"`

	// Title The page title displayed on the web browser, default "KasmVNC Client".
	// +kubebuilder:default="KasmVNC Client"
	Title string `json:"title,omitempty"`

	// FmHome This is the home directory (landing) for the file manager, default "/config".
	// +kubebuilder:default="/config"
	FmHome string `json:"fmHome,omitempty"`

	// StartDocker If set to false a container with privilege will not automatically start the DinD Docker setup.
	// +kubebuilder:default=true
	StartDocker bool `json:"startDocker,omitempty"`

	// Drinode If mounting in /dev/dri for DRI3 GPU Acceleration allows you to specify the device to use IE /dev/dri/renderD128
	Drinode string `json:"drinode,omitempty"`

	// DisableIpv6 If set to true or any value this will disable IPv6
	DisableIpv6 bool `json:"disableIpv6,omitempty"`

	// LcAll Set the Language for the container to run as IE fr_FR.UTF-8 ar_AE.UTF-8
	LcAll string `json:"lcAll,omitempty"`

	// NoDecor If set the application will run without window borders in openbox for use as a PWA.
	NoDecor bool `json:"noDecor,omitempty"`

	// NoFull Do not autmatically fullscreen applications when using openbox.
	NoFull bool `json:"noFull,omitempty"`
}

// WebtopTemplateStatus defines the observed state of WebtopTemplate
type WebtopTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WebtopTemplate is the Schema for the webtoptemplates API
type WebtopTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebtopTemplateSpec   `json:"spec,omitempty"`
	Status WebtopTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebtopTemplateList contains a list of WebtopTemplate
type WebtopTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebtopTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebtopTemplate{}, &WebtopTemplateList{})
}
