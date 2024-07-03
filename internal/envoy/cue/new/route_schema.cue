package envoy

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
