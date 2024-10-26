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

package envoy

import (
	"encoding/json"
)

// Add this type to replace runtime.RawExtension
type DynamicValue struct {
	Value interface{} `json:",inline" yaml:",inline"`
}

// Add marshal/unmarshal methods for proper encoding
func (d *DynamicValue) UnmarshalJSON(data []byte) error {
	// Parse into a generic interface{}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	d.Value = v
	return nil
}

func (d *DynamicValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Parse into a generic interface{}
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	d.Value = v
	return nil
}

func (d DynamicValue) MarshalJSON() ([]byte, error) {
	if d.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(d.Value)
}

func (d DynamicValue) MarshalYAML() (interface{}, error) {
	return d.Value, nil
}

type AccessLog struct {
	Name        string       `json:"name" yaml:"name"`
	TypedConfig DynamicValue `json:"typed_config" yaml:"typed_config"`
}

type SocketAddress struct {
	Address   string `json:"address" yaml:"address"`
	Protocol  string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	PortValue uint32 `json:"port_value,omitempty" yaml:"port_value,omitempty"`
}

type Pipe struct {
	Path string `json:"path" yaml:"path"`
	Mode uint32 `json:"mode,omitempty" yaml:"mode,omitempty"`
}

type EnvoyInternalAddress struct {
	ServerListenerName string `json:"server_listener_name" yaml:"server_listener_name"`
	EndpointId         string `json:"endpoint_id,omitempty" yaml:"endpoint_id,omitempty"`
}

type Address struct {
	SocketAddress        SocketAddress        `json:"socket_address" yaml:"socket_address"`
	Pipe                 Pipe                 `json:"pipe,omitempty" yaml:"pipe,omitempty"`
	EnvoyInternalAddress EnvoyInternalAddress `json:"envoy_internal_address,omitempty" yaml:"envoy_internal_address,omitempty"`
}

type AdditionalAddress struct {
	Address Address `json:"address" yaml:"address"`
}

type HealthCheckConfig struct {
	PortValue                uint32  `json:"port_value" yaml:"port_value"`
	Hostname                 string  `json:"hostname" yaml:"hostname,omitempty"`
	Address                  Address `json:"address" yaml:"address,omitempty"`
	DisableActiveHealthCheck bool    `json:"disable_active_health_check,omitempty" yaml:"disable_active_health_check,omitempty"`
}

type UpgradeConfig struct {
	UpgradeType string `json:"upgrade_type" yaml:"upgrade_type"`
}

type Endpoint struct {
	Address             Address             `json:"address" yaml:"address"`
	HealthCheckConfig   HealthCheckConfig   `json:"health_check_config,omitempty" yaml:"health_check_config,omitempty"`
	Hostname            string              `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	AdditionalAddresses []AdditionalAddress `json:"additional_addresses,omitempty" yaml:"additional_addresses,omitempty"`
}

type LbEndpoint struct {
	Endpoint            Endpoint     `json:"endpoint" yaml:"endpoint"`
	HealthStatus        string       `json:"health_status,omitempty" yaml:"health_status,omitempty"`
	Metadata            DynamicValue `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	LoadBalancingWeight uint32       `json:"load_balancing_weight,omitempty" yaml:"load_balancing_weight,omitempty"`
}

type Locality struct {
	Region  string `json:"region,omitempty" yaml:"region,omitempty"`
	Zone    string `json:"zone,omitempty" yaml:"zone,omitempty"`
	SubZone string `json:"sub_zone,omitempty" yaml:"sub_zone,omitempty"`
}

type LocalityLbEndpoints struct {
	Locality            Locality     `json:"locality,omitempty" yaml:"locality,omitempty"`
	Metadata            DynamicValue `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	LbEndpoints         []LbEndpoint `json:"lb_endpoints" yaml:"lb_endpoints"`
	LoadBalancingWeight uint32       `json:"load_balancing_weight,omitempty" yaml:"load_balancing_weight,omitempty"`
	Priority            uint32       `json:"priority,omitempty" yaml:"priority,omitempty"`
}

type LoadAssignment struct {
	ClusterName string                `json:"cluster_name" yaml:"cluster_name"`
	Endpoints   []LocalityLbEndpoints `json:"endpoints" yaml:"endpoints"`
	Policy      DynamicValue          `json:"policy,omitempty" yaml:"policy,omitempty"`
}

type HTTPFilter struct {
	Name        string       `json:"name" yaml:"name"`
	TypedConfig DynamicValue `json:"typed_config" yaml:"typed_config"`
}

type Cluster struct {
	Name                          string         `json:"name" yaml:"name"`
	ConnectTimeout                string         `json:"connect_timeout" yaml:"connect_timeout"`
	Type                          string         `json:"type" yaml:"type"`
	LbPolicy                      string         `json:"lb_policy" yaml:"lb_policy"`
	TypedExtensionProtocolOptions DynamicValue   `json:"typed_extension_protocol_options,omitempty" yaml:"typed_extension_protocol_options,omitempty"`
	LoadAssignment                LoadAssignment `json:"load_assignment" yaml:"load_assignment"`
}

type Listener struct {
	Name         string        `json:"name" yaml:"name"`
	Address      Address       `json:"address" yaml:"address"`
	FilterChains []FilterChain `json:"filter_chains" yaml:"filter_chains"`
}

type Filter struct {
	Name        string       `json:"name" yaml:"name"`
	TypedConfig DynamicValue `json:"typed_config" yaml:"typed_config"`
}

type FilterChain struct {
	Filters []Filter `json:"filters" yaml:"filters"`
}

type Match struct {
	Prefix string `json:"prefix" yaml:"prefix"`
}

type SingleRoute struct {
	Cluster       string `json:"cluster" yaml:"cluster"`
	PrefixRewrite string `json:"prefix_rewrite,omitempty" yaml:"prefix_rewrite,omitempty"`
}

type Redirect struct {
	HTTPSRedirect bool `json:"https_redirect" yaml:"https_redirect"`
}

type Route struct {
	Match                Match        `json:"match" yaml:"match"`
	Route                *SingleRoute `json:"route,omitempty" yaml:"route,omitempty"`
	Redirect             *Redirect    `json:"redirect,omitempty" yaml:"redirect,omitempty"`
	TypedPerFilterConfig DynamicValue `json:"typed_per_filter_config,omitempty" yaml:"typed_per_filter_config,omitempty"`
}

type VirtualHost struct {
	Name    string   `json:"name" yaml:"name"`
	Domains []string `json:"domains" yaml:"domains"`
	Routes  []Route  `json:"routes" yaml:"routes"`
}

type RouteConfig struct {
	Name         string        `json:"name" yaml:"name"`
	VirtualHosts []VirtualHost `json:"virtual_hosts" yaml:"virtual_hosts"`
}

type AdminServer struct {
	AccessLogPath string  `json:"access_log_path" yaml:"access_log_path"`
	Address       Address `json:"address" yaml:"address"`
}

type HTTPFilterConfig struct {
	Name        string       `yaml:"name"`
	TypedConfig DynamicValue `yaml:"typed_config,omitempty"`
}

// BootstrapConfig represents the complete Envoy bootstrap configuration
type BootstrapConfig struct {
	Admin           AdminServer     `yaml:"admin"`
	StaticResources StaticResources `yaml:"static_resources"`
}

type StaticResources struct {
	Listeners []Listener `yaml:"listeners"`
	Clusters  []Cluster  `yaml:"clusters"`
}

type HTTPConnectionManager struct {
	Type        string             `yaml:"@type"`
	StatPrefix  string             `yaml:"stat_prefix"`
	CodecType   string             `yaml:"codec_type"`
	RouteConfig RouteConfig        `yaml:"route_config"`
	HTTPFilters []HTTPFilterConfig `yaml:"http_filters"`
}

// Slice types
type Clusters []Cluster
type Listeners []Listener
type Filters []Filter
type Routes []Route
type HTTPFilters []HTTPFilter
