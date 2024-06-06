package bootstrap

// HTTPFilterRouter default filter
#HTTPFilterRouter: #HTTPFilter & {
    name: "envoy.filters.http.router"
	typed_config: "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
}

#AdminServer: {
	access_log_path: string | *"/dev/null"
	address: #Address & { socket_address: {
        address: "127.0.0.1"
        port_value: 19000
        }
    }
}

#ReadyServer: {
    Address: string | *"0.0.0.0"
    Port:    int | *19001
    ReadinessPath: string | *"/ready"
}

#AccessLog: [{
    name: "envoy.access_loggers.stdout"
    typed_config: "@type": "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog"
}]

#ListenerHTTP: #Listener & {
    name: "listener_http"
    address: #Address & { socket_address: {
        address: "0.0.0.0"
        port_value: 80
        }
    }
    filter_chains: [{
        filters: [{
            name: "envoy.filters.network.http_connection_manager.http"
            typed_config: {
                "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
				stat_prefix: "ingress_http"
                access_log: #AccessLog
                http_filters: [#HTTPFilterRouter]
                route_config: {
                    name: "local_route"
                    virtual_hosts: [{
                        name: "local_service"
                        domains: ["*"]
                        routes: [{
                            match: {prefix: "/"}
                            redirect: https_redirect: true
                        }]
                    }]
                }
            }
        }]
    }]
}

#ListenerHTTPS: #Listener & {
    name: "listener_https"
    address: #Address & { socket_address: {
        address: "0.0.0.0"
        port_value: 443
        }
    }
    filter_chains: [{
        filters: [{
            name: "envoy.filters.network.http_connection_manager.https"
            typed_config: {
                "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
				stat_prefix: "ingress_https"
                access_log: #AccessLog
                upgrade_configs: [{
                    upgrade_type: "websocket"
                }]
                http_filters: [#ExtAuthzOPAServiceFilter,#HTTPFilterRouter]
                route_config: {
                    name: "local_route"
                    virtual_hosts: [{
                        name: "local_service"
                        domains: ["*"]
                        routes: [{
                            match: {prefix: "/"}
                            route: {cluster: "webtop1"}
                        }]
                    }]
                }
            }
        }]
    }]
}

#ExtAuthzOPAServiceFilter: #HTTPFilter & {
    name: "envoy.filters.http.ext_authz"
    typed_config: {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz"
        with_request_body: {
            max_request_bytes:     8192
            allow_partial_message: true
        }
        failure_mode_allow: false
        grpc_service: {
            envoy_grpc: cluster_name: #ExtAuthzOPAService.load_assignment.cluster_name
            timeout: "0.250s"
        }
        transport_api_version: "V3"
    }
}

#ExtAuthzOPAService: #Cluster & {
    name:      "ext_authz-opa-service"
    typed_extension_protocol_options: "envoy.extensions.upstreams.http.v3.HttpProtocolOptions": {}
    load_assignment: {
        cluster_name: "ext_authz-opa-service"
        endpoints: [{
            lb_endpoints: [{
                address: { socket_address: {
                    address: "ext_authz-http-service"
                    port_value: 8181
                }}
            }]
        }]
    }
}

#BootstrapConfig: {
    clusters: #Clusters & [#ExtAuthzOPAService]
    listeners: #Listeners & [#ListenerHTTP, #ListenerHTTPS]
}

admin: #AdminServer
static_resources: #BootstrapConfig
