package envoy

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
