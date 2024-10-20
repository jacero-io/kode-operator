package envoy

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
)

type ContainerConstructor struct {
	log             logr.Logger
	configGenerator ConfigGenerator
}

type ConfigGenerator interface {
	GenerateEnvoyConfig(config *common.KodeResourceConfig, useBasicAuth bool) (string, error)
}

func NewContainerConstructor(log logr.Logger, configGenerator ConfigGenerator) *ContainerConstructor {
	return &ContainerConstructor{
		log:             log,
		configGenerator: configGenerator,
	}
}

func (c *ContainerConstructor) ConstructEnvoyContainers(config *common.KodeResourceConfig) ([]corev1.Container, []corev1.Container, error) {
	var containers []corev1.Container
	var initContainers []corev1.Container

	useBasicAuth := false
	if config.Template != nil && config.Template.EntryPointSpec != nil &&
		config.Template.EntryPointSpec.AuthSpec != nil {
		useBasicAuth = config.Template.EntryPointSpec.AuthSpec.AuthType == "basicAuth"
	}

	// Generate Envoy config
	envoyConfig, err := c.configGenerator.GenerateEnvoyConfig(config, useBasicAuth)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Envoy config: %w", err)
	}

	// Create proxy setup container
	proxySetupContainer := c.createProxySetupContainer(config)
	initContainers = append(initContainers, proxySetupContainer)

	// Create Envoy container
	envoyContainer := c.createEnvoyContainer(config, envoyConfig)
	containers = append(containers, envoyContainer)

	// Add basic auth sidecar if required
	if useBasicAuth {
		basicAuthContainer := c.createBasicAuthContainer(config)
		containers = append(containers, basicAuthContainer)
	}

	return containers, initContainers, nil
}

func (c *ContainerConstructor) createEnvoyContainer(config *common.KodeResourceConfig, envoyConfig string) corev1.Container {
	return corev1.Container{
		Name:  "envoy",
		Image: "envoyproxy/envoy:v1.22.0",
		Ports: []corev1.ContainerPort{
			{ContainerPort: 8080, Name: "http"},
			{ContainerPort: 8443, Name: "https"},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "envoy-config",
				MountPath: "/etc/envoy",
			},
		},
		Command: []string{"envoy", "-c", "/etc/envoy/envoy.yaml"},
	}
}

func (c *ContainerConstructor) createBasicAuthContainer(config *common.KodeResourceConfig) corev1.Container {
	return corev1.Container{
		Name:  "basic-auth-sidecar",
		Image: "your-basic-auth-sidecar-image:latest",
		Ports: []corev1.ContainerPort{
			{ContainerPort: 9001, Name: "auth"},
		},
		Env: []corev1.EnvVar{
			{
				Name: "AUTH_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: config.CommonConfig.Name + "-auth",
						},
						Key: "username",
					},
				},
			},
			{
				Name: "AUTH_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: config.CommonConfig.Name + "-auth",
						},
						Key: "password",
					},
				},
			},
		},
	}
}

func (c *ContainerConstructor) createProxySetupContainer(config *common.KodeResourceConfig) corev1.Container {
	return corev1.Container{
		Name:  "proxy-setup",
		Image: "openpolicyagent/proxy_init:v8",
		Args: []string{
			"--proxy-uid=1337",
			"--proxy-name=envoy",
			"--redirect-inbound=true",
			"--redirect-outbound=true",
			"--inbound-ports-to-ignore=8080,8443",
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsUser:  common.Int64Ptr(0),
			RunAsGroup: common.Int64Ptr(0),
		},
	}
}
