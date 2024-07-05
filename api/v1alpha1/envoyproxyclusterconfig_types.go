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

// EnvoyProxyClusterConfigSpec defines the desired state of EnvoyProxyClusterConfig
type EnvoyProxyClusterConfigSpec struct {
	SharedEnvoyProxyConfigSpec `json:",inline"`
}

// EnvoyProxyClusterConfigStatus defines the observed state of EnvoyProxyClusterConfig
type EnvoyProxyClusterConfigStatus struct {
	SharedEnvoyProxyStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// EnvoyProxyClusterConfig is the Schema for the envoyproxyclusterconfigs API
type EnvoyProxyClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyProxyClusterConfigSpec   `json:"spec,omitempty"`
	Status EnvoyProxyClusterConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EnvoyProxyClusterConfigList contains a list of EnvoyProxyClusterConfig
type EnvoyProxyClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyProxyClusterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyProxyClusterConfig{}, &EnvoyProxyClusterConfigList{})
}
