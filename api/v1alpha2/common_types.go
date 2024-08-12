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

package v1alpha2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type CredentialsSpec struct {
	// Username is both the the HTTP Basic auth username (when used) and the user the container should run as, abc is default.
	// +kubebuilder:validation:Description="Is both the the HTTP Basic auth username (when used) and the user the container should run as. Defaults to 'abc'."
	// +kubebuilder:default="abc"
	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	// Password HTTP Basic auth password. If unset there will be no auth.
	// +kubebuilder:validation:Description="HTTP Basic auth password. If unset, there will be no authentication."
	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	// ExistingSecret is a reference to an existing secret containing user and password.
	// +kubebuilder:validation:Description="ExistingSecret is a reference to an existing secret containing user and password. If set, User and Password fields are ignored."
	ExistingSecret string `json:"existingSecret,omitempty" yaml:"existingSecret,omitempty"`

	// DisableBuiltinAuth disables the built-in HTTP Basic auth.
	// +kubebuilder:validation:Description="Disables the built-in HTTP Basic auth."
	// +kubebuilder:default=false
	DisableBuiltinAuth bool `json:"disableBuiltinAuth,omitempty" yaml:"disableBuiltinAuth,omitempty"`
}

// BaseSharedSpec defines the common fields for both Tofu and Container specs
type BaseSharedSpec struct {
	// Credentials specifies the credentials for the service.
	// +kubebuilder:validation:Description="Credentials for the service."
	Credentials CredentialsSpec `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	// EntryPointSpec defines the desired state of the entrypoint.
	// +kubebuilder:validation:Description="Desired state of the entrypoint."
	EntryPointRef EntryPointReference `json:"entryPointRef,omitempty" yaml:"entryPointRef,omitempty"`

	// Specifies the period before controller inactive the resource (delete all resources except volume).
	// +kubebuilder:validation:Description="Period before controller inactive the resource (delete all resources except volume)."
	// +kubebuilder:default=600
	InactiveAfterSeconds *int64 `json:"inactiveAfterSeconds,omitempty" yaml:"inactiveAfterSeconds,omitempty"`

	// Specifies the period before controller recycle the resource (delete all resources).
	// +kubebuilder:validation:Description="Period before controller recycle the resource (delete all resources)."
	// +kubebuilder:default=28800
	RecycleAfterSeconds *int64 `json:"recycleAfterSeconds,omitempty" yaml:"recycleAfterSeconds,omitempty"`

	// Port is the port for the service process. Used by EnvoyProxy to expose the kode.
	// +kubebuilder:validation:Description="Port for the service. Used by EnvoyProxy to expose the container. Defaults to '8000'."
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=8000
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`
}

// SharedStatus defines the common observed state
type BaseSharedStatus struct {
	// ObservedGeneration is the last observed generation of the Kode resource.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`

	// Conditions reflect the current state of the template
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Template represents a unified structure for different types of Kode templates
type Template struct {
	// Kind specifies the type of template (e.g., "KodeContainer", "ClusterKodeContainer", "KodeTofu", "ClusterKodeTofu")
	Kind string `json:"kind" yaml:"kind"`

	// Name is the name of the template resource
	Name string `json:"name" yaml:"name"`

	// Namespace is the namespace of the template resource
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Port is the port to expose the kode instance
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	// PodTemplateSpec is a reference to a PodTemplate or ClusterPodTemplate
	PodTemplateSpec *ContainerSharedSpec `json:"container,omitempty" yaml:"container,omitempty"`

	// TofuTemplateSpec is a reference to a TofuTemplate or ClusterTofuTemplate
	TofuTemplateSpec *TofuSharedSpec `json:"tofu,omitempty" yaml:"tofu,omitempty"`
}
