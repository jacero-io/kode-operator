// Bundle for using with Talos
bundle: {
	apiVersion: "v1alpha1"
	name:       "flux-aio"
	instances: {
		"flux": {
			module: {
				url:     "oci://ghcr.io/stefanprodan/modules/flux-aio"
				version: "latest"
			}
			namespace: "flux-system"
			values: {
                controllers: {
                    helm: enabled:         true
                    kustomize: enabled:    true
                    notification: enabled: true
                }
				hostNetwork:     true
				securityProfile: "privileged"
				env: {}
			}
		}
		"eg": {
			module: {
				url: "oci://ghcr.io/stefanprodan/modules/flux-helm-release"
				version: "latest"
			}
			namespace: "kube-system"
			values: {
				repository: url: "oci://docker.io/envoyproxy/gateway-helm"
				chart: {
					name:     "gateway-helm"
					version:  "v0.0.0-latest"
				}
				helmValues: {}
			}
		}
	}
}