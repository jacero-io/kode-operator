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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvoyProxyConfigSpec defines the desired state of EnvoyProxyConfig
type EnvoyProxyConfigSpec struct {
	SharedEnvoyProxyConfigSpec `json:",inline"`
}

// EnvoyProxyConfigStatus defines the observed state of EnvoyProxyConfig
type EnvoyProxyConfigStatus struct {
	SharedEnvoyProxyStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EnvoyProxyConfig is the Schema for the envoyproxyconfigs API
type EnvoyProxyConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyProxyConfigSpec   `json:"spec,omitempty"`
	Status EnvoyProxyConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EnvoyProxyConfigList contains a list of EnvoyProxyConfig
type EnvoyProxyConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyProxyConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyProxyConfig{}, &EnvoyProxyConfigList{})
}
