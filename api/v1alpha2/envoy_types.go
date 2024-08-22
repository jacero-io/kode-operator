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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type PatchPolicy struct {
	// List of Envoy HTTP filters to be applied
	// +kubebuilder:validation:Optional
	HTTPFilters []HTTPFilter `json:"httpFilters"`

	// List of Envoy clusters
	// +kubebuilder:validation:Optional
	Clusters []Cluster `json:"clusters,omitempty"`
}

// // ExtAuthFilter represents an individual HTTP filter configuration
// type ExtAuthFilter struct {
// 	// Name of the HTTP filter
// 	// +kubebuilder:validation:MinLength=1
// 	// +kubebuilder:validation:Required
// 	Name string `json:"name"`

// 	// The typed configuration for the HTTP filter
// 	// It is intentionally the same as the Envoy filter's typed_config field to make it easier to copy-paste
// 	// +kubebuilder:validation:Required
// 	TypedConfig envoy_filter_ext_auth_v3.ExtAuthz `json:"typed_config"`
// }

// type RouterFilter struct {
// 	// Name of the HTTP filter
// 	// +kubebuilder:validation:MinLength=1
// 	// +kubebuilder:validation:Required
// 	Name string `json:"name"`

// 	// The typed configuration for the HTTP filter
// 	// It is intentionally the same as the Envoy filter's typed_config field to make it easier to copy-paste
// 	// +kubebuilder:validation:Required
// 	TypedConfig envoy_filter_router_v3.Router `json:"typed_config"`
// }

// HTTPFilter represents an individual HTTP filter configuration
type HTTPFilter struct {
	// Name of the HTTP filter
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// The typed configuration for the HTTP filter
	// It is intentionally the same as the Envoy filter's typed_config field to make it easier to copy-paste
	// +kubebuilder:validation:Required
	TypedConfig runtime.RawExtension `json:"typed_config"`
}

type SocketAddress struct {
	// The address of the socket
	// +kubebuilder:validation:Required
	Address string `json:"address"`

	// PortValue is the port of the socket
	// +kubebuilder:validation:Required
	PortValue int `json:"port_value"`
}

type Address struct {
	// The socket address
	// +kubebuilder:validation:Required
	SocketAddress SocketAddress `json:"socket_address"`
}

type Endpoint struct {
	// The address of the load balancer endpoint
	// +kubebuilder:validation:Required
	Address Address `json:"address"`
}

type LbEndpoint struct {
	// List of endpoints
	// +kubebuilder:validation:Required
	Endpoint Endpoint `json:"endpoint"`
}

type Endpoints struct {
	// The load balancer endpoints
	// +kubebuilder:validation:Required
	LbEndpoints []LbEndpoint `json:"lb_endpoints"`
}

type LoadAssignment struct {
	// The name of the cluster
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	ClusterName string `json:"cluster_name"`

	// List of endpoints
	// +kubebuilder:validation:Required
	Endpoints []Endpoints `json:"endpoints"`
}

type Cluster struct {
	// Name of the cluster
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// The timeout for connecting to the cluster
	// +kubebuilder:validation:Required
	ConnectTimeout string `json:"connect_timeout"`

	// The type of the cluster
	// +kubebuilder:validation:Required
	// +kube:validation:Enum=STRICT_DNS;LOGICAL_DNS;STATIC;EDS;ORIGINAL_DST;ENVIRONMENT_VARIABLE
	// +kube:validation:default=STRICT_DNS
	Type string `json:"type"`

	// The load balancing policy for the cluster
	// +kubebuilder:validation:Required
	// +kube:validation:Enum=ROUND_ROBIN;LEAST_REQUEST;RANDOM;RING_HASH;MAGLEV;ORIGINAL_DST_LB;CLUSTER_PROVIDED
	LbPolicy string `json:"lb_policy"`

	// Map of typed extension protocol options
	// +kubebuilder:validation:Optional
	TypedExtensionProtocolOptions runtime.RawExtension `json:"typed_extension_protocol_options,omitempty"`

	// The load assignment for the cluster
	// +kubebuilder:validation:Required
	LoadAssignment LoadAssignment `json:"load_assignment"`
}
