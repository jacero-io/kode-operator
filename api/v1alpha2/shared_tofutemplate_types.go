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
	tofuv1alpha2 "github.com/flux-iac/tofu-controller/api/v1alpha2"
)

// TofuSharedSpec defines the desired state of KodeTofu
type TofuSharedSpec struct {

	// BaseSharedSpec is the base shared spec.
	BaseSharedSpec `json:",inline"`

	// EntryPointSpec defines the desired state of the entrypoint.
	// +kubebuilder:validation:Description="Desired state of the entrypoint."
	EntryPointRef EntryPointReference `json:"entryPointRef,omitempty" yaml:"entryPointRef,omitempty"`

	// TofuSpec is the terraform specification.
	// +kubebuilder:validation:Description="Terraform specification."
	TofuSpec tofuv1alpha2.TerraformSpec `json:"tofuSpec,omitempty" yaml:"tofuSpec,omitempty"`
}

// TofuSharedStatus defines the observed state for Tofu
type TofuSharedStatus struct {
	BaseSharedStatus `json:",inline"`
}