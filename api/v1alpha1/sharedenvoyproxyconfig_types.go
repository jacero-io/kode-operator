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

import runtime "k8s.io/apimachinery/pkg/runtime"

// SharedEnvoyProxyStatus defines the observed state of EnvoyProxy
type SharedEnvoyProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// SharedEnvoyProxyConfigSpec defines the desired state of EnvoyProxy
type SharedEnvoyProxyConfigSpec struct {
	// Image is the Docker image for the Envoy proxy
	// +kubebuilder:validation:Description="Docker image for the Envoy proxy"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="envoyproxy/envoy:v1.30-latest"
	Image string `json:"image"`

	// AuthConfig is custom authentication configuration. Can be used to configure basic auth, JWT, etc.
	// +kubebuilder:validation:Description="Custom authentication configuration. Can be used to configure basic auth, JWT, etc."
	// +kubebuilder:validation:Optional
	AuthConfig AuthConfig `json:"authConfig"`

	// HTTPFilters is a list of Envoy HTTP filters to be applied
	// +kubebuilder:validation:Description="HTTP filters to be applied"
	// +kubebuilder:validation:Optional
	HTTPFilters []HTTPFilter `json:"httpFilters"`

	// Clusters is a list of Envoy clusters
	// +kubebuilder:validation:Description="Envoy clusters"
	// +kubebuilder:validation:Optional
	Clusters []Cluster `json:"clusters,omitempty"`
}

type AuthConfig struct {
	// AuthType is the type of authentication. Can be set to "basic" or "jwt"
	// +kubebuilder:validation:Description="Type of authentication. Can be set to 'basic' or 'jwt'"
	// +kubebuilder:validation:enum="basic";"jwt"
	// +kubebuilder:validation:Required
	AuthType string `json:"authType"`
}

// EnvoyProxyReference is a reference to an EnvoyProxyConfig or EnvoyProxyClusterConfig
type EnvoyProxyReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=EnvoyProxyConfig;EnvoyProxyClusterConfig
	// +kubebuilder:validation:Optional
	Kind string `json:"kind,omitempty"`

	// Name is the name of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Name of the config"
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// Namespace is the namespace of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Namespace of the Envoy Proxy config"
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

type SocketAddress struct {
	// Address is the address of the socket
	// +kubebuilder:validation:Required
	Address string `json:"address"`

	// PortValue is the port of the socket
	// +kubebuilder:validation:Required
	PortValue int `json:"port_value"`
}

type Address struct {
	// SocketAddress is the socket address
	// +kubebuilder:validation:Required
	SocketAddress SocketAddress `json:"socket_address"`
}

type Endpoint struct {
	// Address is the address of the load balancer endpoint
	// +kubebuilder:validation:Required
	Address Address `json:"address"`
}

type LbEndpoint struct {
	// Endpoints is a list of endpoints
	// +kubebuilder:validation:Required
	Endpoint Endpoint `json:"endpoint"`
}

type Endpoints struct {
	// LbEndpoints is the load balancer endpoints
	// +kubebuilder:validation:Required
	LbEndpoints []LbEndpoint `json:"lb_endpoints"`
}

type LoadAssignment struct {
	// ClusterName is the name of the cluster
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClusterName string `json:"cluster_name"`

	// Endpoints is a list of endpoints
	// +kubebuilder:validation:Required
	Endpoints []Endpoints `json:"endpoints"`
}

type Cluster struct {
	// Name is the name of the cluster
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ConnectTimeout is the timeout for connecting to the cluster
	// +kubebuilder:validation:Required
	ConnectTimeout string `json:"connect_timeout"`

	// Type is the type of the cluster
	// +kubebuilder:validation:Required
	// +kube:validation:Enum=STRICT_DNS;LOGICAL_DNS;STATIC;EDS;ORIGINAL_DST;ENVIRONMENT_VARIABLE
	// +kube:validation:default=STRICT_DNS
	Type string `json:"type"`

	// LbPolicy is the load balancing policy for the cluster
	// +kubebuilder:validation:Required
	// +kube:validation:Enum=ROUND_ROBIN;LEAST_REQUEST;RANDOM;RING_HASH;MAGLEV;ORIGINAL_DST_LB;CLUSTER_PROVIDED
	LbPolicy string `json:"lb_policy"`

	// TypedExtensionProtocolOptions is a map of typed extension protocol options
	// +kubebuilder:validation:Optional
	TypedExtensionProtocolOptions runtime.RawExtension `json:"typed_extension_protocol_options,omitempty"`

	// LoadAssignment is the load assignment for the cluster
	// +kubebuilder:validation:Required
	LoadAssignment LoadAssignment `json:"load_assignment"`
}
