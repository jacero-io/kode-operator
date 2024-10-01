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
				env: {
					"KUBERNETES_SERVICE_HOST": "localhost"
					"KUBERNETES_SERVICE_PORT": "7445"
				}
			}
		}
		"cilium-hr": {
			module: {
				url: "oci://ghcr.io/stefanprodan/modules/flux-helm-release"
				version: "latest"
			}
			namespace: "kube-system"
			values: {
				repository: url: "https://helm.cilium.io/"
				chart: {
					name:     "cilium"
					version:  "1.16.1"
				}
				helmValues: {
					ipam: mode:               "kubernetes"
					kubeProxyReplacement:     true
					securityContext: capabilities: {
						ciliumAgent: [
							"CHOWN",
							"KILL",
							"NET_ADMIN",
							"NET_RAW",
							"IPC_LOCK",
							"SYS_ADMIN",
							"SYS_RESOURCE",
							"DAC_OVERRIDE",
							"FOWNER",
							"SETGID",
							"SETUID",
						]
						cleanCiliumState: [
							"NET_ADMIN",
							"SYS_ADMIN",
							"SYS_RESOURCE",
						]
					}
					cgroup: {
						autoMount: enabled: false
						hostRoot:           "/sys/fs/cgroup"
					}
					k8sServiceHost: "localhost"
					k8sServicePort: 7445
				}
			}
		}
	}
}