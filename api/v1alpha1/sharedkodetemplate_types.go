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

// SharedKodeTemplateSpec defines the desired state of KodeClusterTemplate
type SharedKodeTemplateSpec struct {
	// Credentials specifies the credentials for the service.
	// +kubebuilder:validation:Description="Credentials for the service."
	Credentials CredentialsSpec `json:"credentials,omitempty"`

	// EntryPointSpec defines the desired state of the entrypoint.
	// +kubebuilder:validation:Description="Desired state of the entrypoint."
	EntryPointRef EntryPointReference `json:"entryPointRef,omitempty"`

	// Runtime is the runtime for the service. Can be one of 'container', 'virtualization', 'tofu'.
	// +kubebuilder:validation:Description="Runtime for the service. Can be one of 'container', 'virtualization', 'tofu'."
	// +kubebuilder:validation:Enum=container;virtualization;tofu
	// +kubebuilder:default=container
	Runtime string `json:"runtime,omitempty"`

	// ContainerSpec is the container specification. Only used if runtime is 'container'.
	// +kubebuilder:validation:Description="Container specification. Only used if runtime is 'container'."
	ContainerSpec ContainerSpec `json:"containerSpec,omitempty"`

	// VirtualMachineSpec is the virtual machine specification. Only used if runtime is 'virtualization'.
	// +kubebuilder:validation:Description="Virtual machine specification. Only used if runtime is 'virtualization'."
	VirtualMachineSpec virtinkv1alpha1.VirtualMachineSpec `json:"virtualMachineSpec,omitempty"`

	// TofuSpec is the terraform specification. Only used if runtime is 'tofu'.
	// +kubebuilder:validation:Description="Terraform specification. Only used if runtime is 'tofu'."
	// TofuSpec tofuv1alpha2.TerraformSpec `json:"tofuSpec,omitempty"`

	// Port is the port for the service process. Used by EnvoyProxy to expose the kode.
	// +kubebuilder:validation:Description="Port for the service. Used by EnvoyProxy to expose the container. Defaults to '8000'."
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=8000
	Port int32 `json:"port,omitempty"`

	// Specifies the period before controller inactive the resource (delete all resources except volume).
	// +kubebuilder:validation:Description="Period before controller inactive the resource (delete all resources except volume)."
	// +kubebuilder:default=600
	InactiveAfterSeconds *int64 `json:"inactiveAfterSeconds,omitempty"`

	// Specifies the period before controller recycle the resource (delete all resources).
	// +kubebuilder:validation:Description="Period before controller recycle the resource (delete all resources)."
	// +kubebuilder:default=28800
	RecycleAfterSeconds *int64 `json:"recycleAfterSeconds,omitempty"`

	// LogoUrl is the URL to the logo for the Kode.
	IconUrl string `json:"logoUrl,omitempty"`
}

// SharedKodeTemplateStatus defines the observed state
type SharedKodeTemplateStatus struct {
	// Conditions reflect the current state of the template
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
