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

// TofuSharedSpec defines the desired state of KodeTofu
type TofuSharedSpec struct {

	// BaseSharedSpec is the base shared spec.
	CommonSpec `json:",inline"`

	// Desired state of the entrypoint.
	EntryPointRef CrossNamespaceObjectReference `json:"entryPointRef,omitempty" yaml:"entryPointRef,omitempty"`

	// erraform specification.
	// TofuSpec tofuv1alpha2.TerraformSpec `json:"tofuSpec,omitempty" yaml:"tofuSpec,omitempty"`
}

// TofuSharedStatus defines the observed state for Tofu
type TofuSharedStatus struct {
	CommonStatus `json:",inline"`
}
