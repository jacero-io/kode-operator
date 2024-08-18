bundle: {
	apiVersion: "v1alpha1"
	name:       "flux-aio"
	instances: {
		"flux": {
			module: {
				url:     "oci://ghcr.io/stefanprodan/modules/flux-aio" @timoni(runtime:string:FLUX_MODULE_URL)
				version: "latest"
			}
			namespace: "flux-system"
			values: {
				hostNetwork:     true
				securityProfile: "restricted"
			}
		}
	}
}
