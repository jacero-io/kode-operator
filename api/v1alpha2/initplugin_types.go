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

import (
	corev1 "k8s.io/api/core/v1"
)

type InitPluginSpec struct {
	// The name of the container.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// The OCI image for the container.
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// The command to run in the container.
	Command []string `json:"command,omitempty"`

	// The arguments that will be passed to the command in the main container.
	Args []string `json:"args,omitempty"`

	// The environment variables for the main container.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// The environment variables taken from a Secret or ConfigMap for the main container.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// The volume mounts for the container. Can be used to mount a ConfigMap or Secret.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}
