package envoy

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
