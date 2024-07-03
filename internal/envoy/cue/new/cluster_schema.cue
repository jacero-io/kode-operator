package envoy

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
