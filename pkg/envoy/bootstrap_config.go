package envoy

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
)

type BootstrapConfigGenerator struct {
	log logr.Logger
}

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{log: log}
}

func (g *BootstrapConfigGenerator) GenerateEnvoyConfig(config *common.KodeResourceConfig, useBasicAuth bool) (string, error) {
	bootstrapConfig := struct {
		Admin           AdminServer `json:"admin"`
		StaticClusters  Clusters    `json:"static_clusters"`
		StaticResources struct {
			Listeners Listeners `json:"listeners"`
			Clusters  Clusters  `json:"clusters"`
		} `json:"static_resources"`
	}{
		Admin: AdminServer{
			AccessLogPath: "/dev/stdout",
			Address: Address{
				SocketAddress: SocketAddress{
					Address:   "0.0.0.0",
					PortValue: 9901,
				},
			},
		},
		StaticResources: struct {
			Listeners Listeners `json:"listeners"`
			Clusters  Clusters  `json:"clusters"`
		}{
			Listeners: []Listener{
				{
					Name: "listener_0",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   "0.0.0.0",
							PortValue: Port(config.Port),
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []Filter{
								{
									Name: "envoy.filters.network.http_connection_manager",
									TypedConfig: TypedConfig{
										Type:       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
										CodecType:  "AUTO",
										StatPrefix: "ingress_http",
										AccessLog: []AccessLog{
											{
												Name: "envoy.access_loggers.stdout",
												TypedConfig: map[string]interface{}{
													"@type": "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
												},
											},
										},
										RouteConfig: RouteConfig{
											Name: "local_route",
											VirtualHosts: []struct {
												Name    string   `json:"name" yaml:"name"`
												Domains []string `json:"domains" yaml:"domains"`
												Routes  []Route  `json:"routes" yaml:"routes"`
											}{
												{
													Name:    "local_service",
													Domains: []string{"*"},
													Routes: []Route{
														{
															Match: struct {
																Prefix string `json:"prefix" yaml:"prefix"`
															}{
																Prefix: "/",
															},
															Route: &struct {
																Cluster       string `json:"cluster" yaml:"cluster"`
																PrefixRewrite string `json:"prefix_rewrite,omitempty" yaml:"prefixRewrite,omitempty"`
															}{
																Cluster: "local_service",
															},
														},
													},
												},
											},
										},
										HTTPFilters: g.getHTTPFilters(useBasicAuth),
									},
								},
							},
						},
					},
				},
			},
			Clusters: g.getClusters(config, useBasicAuth),
		},
	}

	// Convert the config to JSON
	jsonConfig, err := json.MarshalIndent(bootstrapConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Envoy bootstrap config: %w", err)
	}

	return string(jsonConfig), nil
}

func (g *BootstrapConfigGenerator) getHTTPFilters(useBasicAuth bool) []HTTPFilter {
	filters := []HTTPFilter{}

	if useBasicAuth {
		filters = append(filters, HTTPFilter{
			Name: "envoy.filters.http.ext_authz",
			TypedConfig: map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz",
				"grpc_service": map[string]interface{}{
					"envoy_grpc": map[string]interface{}{
						"cluster_name": "basic_auth_service",
					},
					"timeout": "0.25s",
				},
			},
		})
	}

	filters = append(filters, HTTPFilter{
		Name: "envoy.filters.http.router",
		TypedConfig: map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
		},
	})

	return filters
}

func (g *BootstrapConfigGenerator) getClusters(config *common.KodeResourceConfig, useBasicAuth bool) []Cluster {
	clusters := []Cluster{
		{
			Name:           "local_service",
			ConnectTimeout: "0.25s",
			Type:           "STATIC",
			LbPolicy:       "ROUND_ROBIN",
			LoadAssignment: LoadAssignment{
				ClusterName: "local_service",
				Endpoints: []LocalityLbEndpoints{
					{
						LbEndpoints: []LbEndpoint{
							{
								Endpoint: Endpoint{
									Address: Address{
										SocketAddress: SocketAddress{
											Address:   "127.0.0.1",
											PortValue: Port(config.Port),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if useBasicAuth {
		clusters = append(clusters, Cluster{
			Name:           "basic_auth_service",
			ConnectTimeout: "0.25s",
			Type:           "STATIC",
			LbPolicy:       "ROUND_ROBIN",
			LoadAssignment: LoadAssignment{
				ClusterName: "basic_auth_service",
				Endpoints: []LocalityLbEndpoints{
					{
						LbEndpoints: []LbEndpoint{
							{
								Endpoint: Endpoint{
									Address: Address{
										SocketAddress: SocketAddress{
											Address:   "127.0.0.1",
											PortValue: 9001,
										},
									},
								},
							},
						},
					},
				},
			},
		})
	}

	return clusters
}
