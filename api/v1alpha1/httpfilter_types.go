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
	"k8s.io/apimachinery/pkg/runtime"
)

// TypedConfig represents the typed configuration for an Envoy filter
type TypedConfig struct {
	// Type is the type of the configuration
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Description=Type of the configuration
	// +kubebuilder:validation:Required
	Type string `json:"@type"`

	// Config is the configuration for the filter
	// +kubebuilder:validation:Description=Configuration for the filter
	Config runtime.RawExtension `json:"config,omitempty"`
}

// HTTPFilter represents an individual HTTP filter configuration
type HTTPFilter struct {
	// Name is the name of the HTTP filter
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Description=Name of the HTTP filter
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// TypedConfig is the typed configuration for the HTTP filter
	// It is intentionally the same as the Envoy filter's typed_config field to make it easier to copy-paste
	// +kubebuilder:validation:Description=Typed configuration for the HTTP filter. It is intentionally the same as the Envoy filter's typed_config field to make it easier to copy-paste
	// +kubebuilder:validation:Required
	TypedConfig TypedConfig `json:"typed_config"`
}
