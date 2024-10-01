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
	corev1 "k8s.io/api/core/v1"
)

// ContainerTemplateSharedSpec defines the desired state of KodeContainer
type ContainerTemplateSharedSpec struct {

	// BaseSharedSpec is the base shared spec.
	BaseSharedSpec `json:",inline"`

	// Type is the type of container to use. Can be one of 'code-server', 'webtop', 'devcontainers', 'jupyter'.
	// +kubebuilder:validation:Enum=code-server;webtop
	Type string `json:"type,omitempty"`

	// Image is the container image for the service.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// Command is the command to run in the container.
	Command []string `json:"command,omitempty"`

	// Specifies the environment variables.
	Env []corev1.EnvVar `json:"envs,omitempty"`

	// EnvFrom are the environment variables taken from a Secret or ConfigMap.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Specifies the args.
	Args []string `json:"args,omitempty"`

	// Specifies the resources for the pod.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Timezone for the service process. Defaults to 'UTC'.
	// +kubebuilder:default="UTC"
	TZ string `json:"tz,omitempty"`

	// User ID for the service process. Defaults to '1000'.
	// +kubebuilder:default=1000
	PUID int64 `json:"puid,omitempty"`

	// Group ID for the service process. Defaults to '1000'.
	// +kubebuilder:default=1000
	PGID int64 `json:"pgid,omitempty"`

	// Default home is the path to the directory for the user data. Defaults to '/config'.
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	DefaultHome string `json:"defaultHome,omitempty"`

	// Default workspace is the default directory for the kode instance on first launch. Defaults to 'workspace'.
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	// +kubebuilder:default=workspace
	DefaultWorkspace string `json:"defaultWorkspace,omitempty"`

	// Flag to allow privileged execution of the container.
	// +kubebuilder:default=false
	AllowPrivileged *bool `json:"allowPrivileged,omitempty"`

	// OCI containers to be run before the main container. It is an ordered list.
	InitPlugins []InitPluginSpec `json:"initPlugins,omitempty"`
}

// ContainerTemplateSharedStatus defines the observed state for Container
type ContainerTemplateSharedStatus struct {
	BaseSharedStatus `json:",inline"`
}
