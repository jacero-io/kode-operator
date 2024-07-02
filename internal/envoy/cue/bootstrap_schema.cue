package bootstrap

#Port: uint32

#SocketAddress: {
    address:    string
    port_value: #Port
}

#Address: {
    socket_address: #SocketAddress
    pipe?: {...}
    envoy_internal_address?: {...}
}

// Route Definitions
#Route: {
    match: {
        prefix: string
    }
    route?: {
        cluster: string
        prefix_rewrite?: string
    }
    redirect?: {
        https_redirect: bool
    }
    typed_per_filter_config?: {...}
}

#Routes: [...#Route]

// Route Config Definitions
#RouteConfig: {
    name: string
    virtual_hosts: [{
        name: string
        domains: [string]
        routes: #Routes
    }]
}

#HTTPFilter: {
    name: string
    typed_config: {
        "@type": string
        ...
    }
}

#HTTPFilters: [...#HTTPFilter]

// Filter Definitions
#Filter: {
    name: string
    typed_config: {
        "@type": string
        codec_type: string | *"AUTO"
		stat_prefix: string | *name
        access_log: [{
            name: string
            typed_config: {
                "@type": string
                ...
            }
        }]
        upgrade_configs?: [{
            upgrade_type: string
            ...
        }]
        route_config: #RouteConfig
        http_filters: #HTTPFilters
    }
}

// Filter Definitions
#Filters: [...#Filter]
#FilterChains: [{
    filters: #Filters
}]

// Listener Definitions
#Listener: {
    name: string
    address: #Address
    filter_chains: #FilterChains
}
#Listeners: [...#Listener]

// Endpoint Definitions
#Endpoint: {
    address: #Address
    health_check_config?: {...}
    hostname?: string
    additional_addresses?: [...]
}

#LbEndpoint: {
    endpoint: #Endpoint
    health_status?: string
    metadata?: {...}
    load_balancing_weight?: {...}
}

#LocalityLbEndpoints: {
    locality?: {...}
    lb_endpoints: [...#LbEndpoint]
    load_balancer_endpoints?: {...}
    leds_cluster_locality_config?: {...}
    load_balancing_weight?: {...}
    priority?: uint32
}

#LoadAssignment: {
    cluster_name: string
    endpoints: [#LocalityLbEndpoints]
    policy?: {...}
}

// Cluster Definitions
#Cluster: {
    name: string
    connect_timeout: string | *"0.25s"
    type: string | *"STRICT_DNS"
    lb_policy: string | *"ROUND_ROBIN"
    typed_extension_protocol_options?: {...}
    load_assignment: #LoadAssignment
}

#Clusters: [...#Cluster]