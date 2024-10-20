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

type Port uint32

type AccessLog struct {
	Name        string                 `json:"name" yaml:"name"`
	TypedConfig map[string]interface{} `json:"typed_config" yaml:"typedConfig"`
}

type SocketAddress struct {
	Address   string `json:"address" yaml:"address"`
	PortValue Port   `json:"port_value" yaml:"portValue"`
}

type Address struct {
	SocketAddress        SocketAddress `json:"socket_address" yaml:"socketAddress"`
	Pipe                 interface{}   `json:"pipe,omitempty" yaml:"pipe,omitempty"`
	EnvoyInternalAddress interface{}   `json:"envoy_internal_address,omitempty" yaml:"envoyInternalAddress,omitempty"`
}

type UpgradeConfig struct {
	UpgradeType string `json:"upgrade_type" yaml:"upgradeType"`
}

type Cluster struct {
	Name                          string         `json:"name" yaml:"name"`
	ConnectTimeout                string         `json:"connect_timeout" yaml:"connectTimeout"`
	Type                          string         `json:"type" yaml:"type"`
	LbPolicy                      string         `json:"lb_policy" yaml:"lbPolicy"`
	TypedExtensionProtocolOptions interface{}    `json:"typed_extension_protocol_options,omitempty" yaml:"typedExtensionProtocolOptions,omitempty"`
	LoadAssignment                LoadAssignment `json:"load_assignment" yaml:"loadAssignment"`
}

type Listener struct {
	Name         string        `json:"name" yaml:"name"`
	Address      Address       `json:"address" yaml:"address"`
	FilterChains []FilterChain `json:"filter_chains" yaml:"filterChains"`
}

type Endpoint struct {
	Address             Address       `json:"address" yaml:"address"`
	HealthCheckConfig   interface{}   `json:"health_check_config,omitempty" yaml:"healthCheckConfig,omitempty"`
	Hostname            string        `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	AdditionalAddresses []interface{} `json:"additional_addresses,omitempty" yaml:"additionalAddresses,omitempty"`
}

type LbEndpoint struct {
	Endpoint            Endpoint    `json:"endpoint" yaml:"endpoint"`
	HealthStatus        string      `json:"health_status,omitempty" yaml:"healthStatus,omitempty"`
	Metadata            interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	LoadBalancingWeight interface{} `json:"load_balancing_weight,omitempty" yaml:"loadBalancingWeight,omitempty"`
}

type LocalityLbEndpoints struct {
	Locality                  interface{}  `json:"locality,omitempty" yaml:"locality,omitempty"`
	LbEndpoints               []LbEndpoint `json:"lb_endpoints" yaml:"lbEndpoints"`
	LoadBalancerEndpoints     interface{}  `json:"load_balancer_endpoints,omitempty" yaml:"loadBalancerEndpoints,omitempty"`
	LedsClusterLocalityConfig interface{}  `json:"leds_cluster_locality_config,omitempty" yaml:"ledsClusterLocalityConfig,omitempty"`
	LoadBalancingWeight       interface{}  `json:"load_balancing_weight,omitempty" yaml:"loadBalancingWeight,omitempty"`
	Priority                  uint32       `json:"priority,omitempty" yaml:"priority,omitempty"`
}

type LoadAssignment struct {
	ClusterName string                `json:"cluster_name" yaml:"clusterName"`
	Endpoints   []LocalityLbEndpoints `json:"endpoints" yaml:"endpoints"`
	Policy      interface{}           `json:"policy,omitempty" yaml:"policy,omitempty"`
}

type HTTPFilter struct {
	Name        string                 `json:"name" yaml:"name"`
	TypedConfig map[string]interface{} `json:"typed_config" yaml:"typedConfig"`
}

type TypedConfig struct {
	Type           string          `json:"@type" yaml:"@type"`
	CodecType      string          `json:"codec_type" yaml:"codecType"`
	StatPrefix     string          `json:"stat_prefix" yaml:"statPrefix"`
	AccessLog      []AccessLog     `json:"access_log" yaml:"accessLog"`
	UpgradeConfigs []UpgradeConfig `json:"upgrade_configs,omitempty" yaml:"upgradeConfigs,omitempty"`
	RouteConfig    RouteConfig     `json:"route_config" yaml:"routeConfig"`
	HTTPFilters    []HTTPFilter    `json:"http_filters" yaml:"httpFilters"`
}

type Filter struct {
	Name        string      `json:"name" yaml:"name"`
	TypedConfig TypedConfig `json:"typed_config" yaml:"typedConfig"`
}

type FilterChain struct {
	Filters []Filter `json:"filters" yaml:"filters"`
}

type Route struct {
	Match struct {
		Prefix string `json:"prefix" yaml:"prefix"`
	} `json:"match" yaml:"match"`
	Route *struct {
		Cluster       string `json:"cluster" yaml:"cluster"`
		PrefixRewrite string `json:"prefix_rewrite,omitempty" yaml:"prefixRewrite,omitempty"`
	} `json:"route,omitempty" yaml:"route,omitempty"`
	Redirect *struct {
		HTTPSRedirect bool `json:"https_redirect" yaml:"httpsRedirect"`
	} `json:"redirect,omitempty" yaml:"redirect,omitempty"`
	TypedPerFilterConfig interface{} `json:"typed_per_filter_config,omitempty" yaml:"typedPerFilterConfig,omitempty"`
}

type RouteConfig struct {
	Name         string `json:"name" yaml:"name"`
	VirtualHosts []struct {
		Name    string   `json:"name" yaml:"name"`
		Domains []string `json:"domains" yaml:"domains"`
		Routes  []Route  `json:"routes" yaml:"routes"`
	} `json:"virtual_hosts" yaml:"virtualHosts"`
}

type AdminServer struct {
	AccessLogPath string  `json:"access_log_path" yaml:"accessLogPath"`
	Address       Address `json:"address" yaml:"address"`
	PortValue     Port    `json:"port_value" yaml:"portValue"`
}

// Slice types
type Clusters []Cluster
type Listeners []Listener
type Filters []Filter
type Routes []Route
type HTTPFilters []HTTPFilter
