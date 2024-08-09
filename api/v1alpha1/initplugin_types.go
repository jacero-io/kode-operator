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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type InitPluginSpec struct {
	// Name is the name of the container.
	// +kubebuilder:validation:Description="Name of the container."
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Image is the OCI image for the container.
	// +kubebuilder:validation:Description="OCI image for the container."
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Command is the command to run in the container.
	// +kubebuilder:validation:Description="Command to run in the container."
	Command []string `json:"command,omitempty"`

	// Args are the arguments to the container.
	// +kubebuilder:validation:Description="Arguments to the container."
	Args []string `json:"args,omitempty"`

	// Env are the environment variables to the container.
	// +kubebuilder:validation:Description="Environment variables to the container."
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom are the environment variables taken from a Secret or ConfigMap.
	// +kubebuilder:validation:Description="Environment variables taken from a Secret or ConfigMap."
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// VolumeMounts are the volume mounts for the container. Can be used to mount a ConfigMap or Secret.
	// +kubebuilder:validation:Description="Volume mounts for the container. Can be used to mount a ConfigMap or Secret."
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}
