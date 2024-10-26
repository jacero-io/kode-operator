package envoy

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/pkg/constant"
	"gopkg.in/yaml.v2"
)

type BootstrapConfigGenerator struct {
	log logr.Logger
}

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{log: log}
}

func (g *BootstrapConfigGenerator) GenerateEnvoyConfig(config *common.KodeResourceConfig, useBasicAuth bool) (string, error) {
	// Create HTTP connection manager config
	httpConnManager := HTTPConnectionManager{
		Type:       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
		StatPrefix: "ingress_http",
		CodecType:  "AUTO",
		RouteConfig: RouteConfig{
			Name: "local_route",
			VirtualHosts: []VirtualHost{
				{
					Name:    "local_service",
					Domains: []string{"*"},
					Routes: []Route{
						{
							Match: Match{
								Prefix: "/",
							},
							Route: &SingleRoute{
								Cluster: "local_service",
							},
						},
					},
				},
			},
		},
		HTTPFilters: g.getHTTPFilters(useBasicAuth),
	}

	// Marshal HTTP connection manager to get structured map
	httpConnBytes, err := yaml.Marshal(httpConnManager)
	if err != nil {
		return "", fmt.Errorf("failed to marshal HTTP connection manager: %w", err)
	}

	var httpConnMap map[string]interface{}
	if err := yaml.Unmarshal(httpConnBytes, &httpConnMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal HTTP connection manager: %w", err)
	}

	// Create complete bootstrap config
	bootstrapConfig := BootstrapConfig{
		Admin: AdminServer{
			AccessLogPath: "/dev/stdout",
			Address: Address{
				SocketAddress: SocketAddress{
					Address:   "0.0.0.0",
					PortValue: 9901,
				},
			},
		},
		StaticResources: StaticResources{
			Listeners: []Listener{
				{
					Name: "listener_0",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   "0.0.0.0",
							PortValue: uint32(config.Port),
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []Filter{
								{
									Name: "envoy.filters.network.http_connection_manager",
									TypedConfig: DynamicValue{
										Value: httpConnMap,
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

	// Marshal complete config to YAML
	yamlConfig, err := yaml.Marshal(bootstrapConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Envoy bootstrap config: %w", err)
	}

	// Log configurations
	g.log.V(1).Info("Generated complete Envoy bootstrap config",
		"port", config.Port,
		"useBasicAuth", useBasicAuth,
		"yaml", "\n"+string(yamlConfig))

	clustersYAML, err := yaml.Marshal(g.getClusters(config, useBasicAuth))
	if err != nil {
		g.log.Error(err, "Failed to marshal clusters config for debugging")
	} else {
		g.log.V(1).Info("Generated clusters config",
			"useBasicAuth", useBasicAuth,
			"yaml", "\n"+string(clustersYAML))
	}

	return string(yamlConfig), nil
}

func (g *BootstrapConfigGenerator) getHTTPFilters(useBasicAuth bool) []HTTPFilterConfig {
	filters := []HTTPFilterConfig{}

	if useBasicAuth {
		filters = append(filters, HTTPFilterConfig{
			Name: "envoy.filters.http.ext_authz",
			TypedConfig: DynamicValue{
				Value: map[string]interface{}{
					"@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz",
					"grpc_service": map[string]interface{}{
						"envoy_grpc": map[string]interface{}{
							"cluster_name": "basic_auth_service",
						},
						"timeout": "0.25s",
					},
				},
			},
		})
	}

	// Router filter
	filters = append(filters, HTTPFilterConfig{
		Name: "envoy.filters.http.router",
		TypedConfig: DynamicValue{
			Value: map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
			},
		},
	})

	return filters
}

func (g *BootstrapConfigGenerator) getClusters(config *common.KodeResourceConfig, useBasicAuth bool) Clusters {
	clusters := Clusters{
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
											PortValue: uint32(constant.DefaultKodePodPort),
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
