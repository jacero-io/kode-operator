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

// TypedConfig represents the typed configuration for an Envoy filter
type TypedConfig struct {
	Type                string         `json:"@type"`
	TransportAPIVersion string       `json:"transport_api_version,omitempty"`
	HTTPService         *HTTPService  `json:"http_service,omitempty"`
	WithRequestBody     *WithRequestBody `json:"with_request_body,omitempty"`
	FailureModeAllow    *bool         `json:"failure_mode_allow,omitempty"`
	GRPCService         *GRPCService  `json:"grpc_service,omitempty"`
}

// HTTPService represents the HTTP service configuration
type HTTPService struct {
	ServerURI ServerURI `json:"server_uri"`
}

// ServerURI represents the server URI configuration
type ServerURI struct {
	URI     string `json:"uri"`
	Cluster string `json:"cluster"`
	Timeout string `json:"timeout"`
}

// WithRequestBody represents the request body configuration
type WithRequestBody struct {
	MaxRequestBytes     int  `json:"max_request_bytes"`
	AllowPartialMessage bool `json:"allow_partial_message"`
}

// GRPCService represents the gRPC service configuration
type GRPCService struct {
	EnvoyGRPC EnvoyGRPC `json:"envoy_grpc"`
	Timeout   string    `json:"timeout"`
}

// EnvoyGRPC represents the Envoy gRPC configuration
type EnvoyGRPC struct {
	ClusterName string `json:"cluster_name"`
}

// HTTPFilter represents an individual HTTP filter configuration
type HTTPFilter struct {
	Name        string      `json:"name"`
	TypedConfig TypedConfig `json:"typed_config"`
}
