package envoy

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
	"gopkg.in/yaml.v2"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type BootstrapConfigGenerator struct {
	log logr.Logger
}

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{log: log}
}

func (g *BootstrapConfigGenerator) GenerateEnvoyConfig(config *common.KodeResourceConfig, useBasicAuth bool) (string, error) {
	httpConnectionManager := map[string]interface{}{
		"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
		"stat_prefix": "ingress_http",
		"codec_type":  "AUTO",
		"route_config": map[string]interface{}{
			"name": "local_route",
			"virtual_hosts": []map[string]interface{}{
				{
					"name":    "local_service",
					"domains": []string{"*"},
					"routes": []map[string]interface{}{
						{
							"match": map[string]interface{}{
								"prefix": "/",
							},
							"route": map[string]interface{}{
								"cluster": "local_service",
							},
						},
					},
				},
			},
		},
		"http_filters": g.getHTTPFilters(useBasicAuth),
	}

	httpConnectionManagerJSON, err := json.Marshal(httpConnectionManager)
	if err != nil {
		return "", fmt.Errorf("failed to marshal HTTP connection manager: %w", err)
	}

	bootstrapConfig := struct {
		Admin           AdminServer `yaml:"admin"`
		StaticResources struct {
			Listeners Listeners `yaml:"listeners"`
			Clusters  Clusters  `yaml:"clusters"`
		} `yaml:"static_resources"`
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
			Listeners Listeners `yaml:"listeners"`
			Clusters  Clusters  `yaml:"clusters"`
		}{
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
									TypedConfig: runtime.RawExtension{
										Raw: httpConnectionManagerJSON,
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

	// Convert the config to YAML
	yamlConfig, err := yaml.Marshal(bootstrapConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Envoy bootstrap config to YAML: %w", err)
	}

	return string(yamlConfig), nil
}

func (g *BootstrapConfigGenerator) getHTTPFilters(useBasicAuth bool) []map[string]interface{} {
	filters := []map[string]interface{}{}

	if useBasicAuth {
		filters = append(filters, map[string]interface{}{
			"name": "envoy.filters.http.ext_authz",
			"typed_config": map[string]interface{}{
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

	filters = append(filters, map[string]interface{}{
		"name": "envoy.filters.http.router",
		"typed_config": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
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
											PortValue: uint32(config.Port),
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
