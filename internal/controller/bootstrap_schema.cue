package bootstrap

#Address: socket_address: {
            address:    string
            port_value: int
        }

//////////////////////////////////////////////////////////////////////////////////////////////////////////
// Definitions for the bootstrap configuration

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

//////////////////////////////////////////////////////////////////////////////////////////////////////////
// CLUSTERS

// Endpoint Definitions
#Endpoint: {
    address: #Address
}

#LBEndpoints: [...#Endpoint] | *[]

#Endpoints: [{
    lb_endpoints: #LBEndpoints
}]

// Load Assignment Definitions
#LoadAssignment: {
    cluster_name: string
    endpoints: #Endpoints
}

// Cluster Definitions
#Cluster: {
    name: string
    connect_timeout: string | *"0.25s"
    type: string | *"STRICT_DNS"
    lb_policy: string | *"ROUND_ROBIN"
    typed_extension_protocol_options?: "envoy.extensions.upstreams.http.v3.HttpProtocolOptions": {} | *{}
    load_assignment: #LoadAssignment
}

#Clusters: [...#Cluster]
