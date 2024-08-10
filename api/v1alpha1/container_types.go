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
	corev1 "k8s.io/api/core/v1"
)

type ContainerSpec struct {
	// Type is the type of container to use. Can be one of 'code-server', 'webtop', 'devcontainers', 'jupyter'.
	// +kubebuilder:validation:Description="Type of container to use. Can be one of 'code-server', 'webtop', 'jupyter'."
	// +kubebuilder:validation:Enum=code-server;webtop
	Type string `json:"type,omitempty"`

	// Image is the container image for the service.
	// +kubebuilder:validation:Description="Container image for the service."
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// Command is the command to run in the container.
	// +kubebuilder:validation:Description="Command to run in the container."
	Command []string `json:"command,omitempty"`

	// Specifies the environment variables.
	// +kubebuilder:validation:Description="Environment variables for the container"
	Env []corev1.EnvVar `json:"envs,omitempty"`

	// EnvFrom are the environment variables taken from a Secret or ConfigMap.
	// +kubebuilder:validation:Description="Environment variables taken from a Secret or ConfigMap."
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Specifies the args.
	// +kubebuilder:validation:Description="Arguments for the container."
	Args []string `json:"args,omitempty"`

	// TZ is the timezone for the service process.
	// +kubebuilder:validation:Description="Timezone for the service process. Defaults to 'UTC'."
	// +kubebuilder:default="UTC"
	TZ string `json:"tz,omitempty"`

	// PUID is the user ID for the service process.
	// +kubebuilder:validation:Description="User ID for the service process. Defaults to '1000'."
	// +kubebuilder:default=1000
	PUID int64 `json:"puid,omitempty"`

	// PGID is the group ID for the service process.
	// +kubebuilder:validation:Description="Group ID for the service process. Defaults to '1000'."
	// +kubebuilder:default=1000
	PGID int64 `json:"pgid,omitempty"`

	// DefaultHome is the path to the directory for the user data.
	// +kubebuilder:validation:Description="Default home is the path to the directory for the user data. Defaults to '/config'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	DefaultHome string `json:"defaultHome,omitempty"`

	// DefaultWorkspace is the default directory for the kode instance on first launch. Defaults to 'workspace'.
	// +kubebuilder:validation:Description="Default workspace is the default directory for the kode instance on first launch. Defaults to 'workspace'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	// +kubebuilder:default=workspace
	DefaultWorkspace string `json:"defaultWorkspace,omitempty"`

	// AllowPrivileged is a flag to allow privileged execution of the container.
	// +kubebuilder:validation:Description="Flag to allow privileged execution of the container."
	// +kubebuilder:default=false
	AllowPrivileged *bool `json:"allowPrivileged,omitempty"`

	// InitPlugins specifies the OCI containers to be run before the main container. It is an ordered list.
	// +kubebuilder:validation:Description="OCI containers to be run before the main container. It is an ordered list."
	InitPlugins []InitPluginSpec `json:"initPlugins,omitempty"`
}
