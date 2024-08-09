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

package v1alpha1

import (
	// tofuv1alpha2 "github.com/flux-iac/tofu-controller/api/v1alpha2"
	virtinkv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Credentials struct {
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
}

// SharedKodeTemplateSpec defines the desired state of KodeClusterTemplate
type SharedKodeTemplateSpec struct {
	// Credentials specifies the credentials for the service.
	// +kubebuilder:validation:Description="Credentials for the service."
	Credentials   Credentials   `json:"credentials,omitempty"`

	// EntryPointSpec defines the desired state of the entrypoint.
	// +kubebuilder:validation:Description="Desired state of the entrypoint."
	EntryPointSpec EntryPointSpec `json:"entryPointSpec,omitempty"`

	// Runtime is the runtime for the service. Can be one of 'container', 'virtual', 'tofu'.
	// +kubebuilder:validation:Description="Runtime for the service. Can be one of 'container', 'virtual', 'tofu'."
	// +kubebuilder:validation:Enum=container;virtual;tofu
	Runtime string `json:"runtime,omitempty"`

	// ContainerSpec is the container specification.
	// +kubebuilder:validation:Description="Container specification."
	ContainerSpec ContainerSpec `json:"containerSpec,omitempty"`

	// VirtualMachineSpec is the virtual machine specification.
	// +kubebuilder:validation:Description="Virtual machine specification."
	VirtualMachineSpec virtinkv1alpha1.VirtualMachineSpec `json:"virtualMachineSpec,omitempty"`

	// TofuSpec is the terraform specification.
	// +kubebuilder:validation:Description="Terraform specification."
	// TofuSpec tofuv1alpha2.TerraformSpec `json:"tofuSpec,omitempty"`

	// Specifies the period before controller inactive the resource (delete all resources except volume).
	// +kubebuilder:validation:Description="Period before controller inactive the resource (delete all resources except volume)."
	// +kubebuilder:default=600
	InactiveAfterSeconds *int64 `json:"inactiveAfterSeconds,omitempty"`

	// Specifies the period before controller recycle the resource (delete all resources).
	// +kubebuilder:validation:Description="Period before controller recycle the resource (delete all resources)."
	// +kubebuilder:default=28800
	RecycleAfterSeconds *int64 `json:"recycleAfterSeconds,omitempty"`
}

type TerraformSpec struct {
	// Module is the module for the service.
	// +kubebuilder:validation:Description="Module for the service."
	// +kubebuilder:validation:MinLength=1
	Module string `json:"module,omitempty"`

	SourceRef SourceReference `json:"sourceRef,omitempty"`
}

// SourceReference is a reference to a source
type SourceReference struct {
	// Kind is the kind of the source.
	// +kubebuilder:validation:Description="Kind of the source."
	Kind string `json:"kind,omitempty"`

	// Name is the name of the source.
	// +kubebuilder:validation:Description="Name of the source."
	Name string `json:"name,omitempty"`

	// Namespace is the namespace of the source.
	// +kubebuilder:validation:Description="Namespace of the source."
	Namespace string `json:"namespace,omitempty"`
}

// SharedKodeTemplateStatus defines the observed state
type SharedKodeTemplateStatus struct {
	// Conditions reflect the current state of the template
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
