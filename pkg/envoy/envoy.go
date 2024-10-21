package envoy

import (
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
)

const (
	EnvoyProxyContainerName  = "envoy-proxy"
	EnvoyProxyContainerImage = "envoyproxy/envoy:v1.31-latest"
	EnvoyProxyRunAsUser      = 1337
	ProxyInitContainerName   = "proxy-init"
	ProxyInitContainerImage  = "openpolicyagent/proxy_init:v8"
	BasicAuthContainerPort   = 9001
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

func (c *ContainerConstructor) createProxySetupContainer(config *common.KodeResourceConfig) corev1.Container {
	return corev1.Container{
		Name:  ProxyInitContainerName,
		Image: ProxyInitContainerImage,
		Args:  []string{"-p", strconv.Itoa(int(config.Port)), "-u", strconv.FormatInt(EnvoyProxyRunAsUser, 16)},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsNonRoot: common.BoolPtr(false),
			RunAsUser:    common.Int64Ptr(0),
		},
	}
}

func (c *ContainerConstructor) createEnvoyContainer(config *common.KodeResourceConfig, envoyConfig string) corev1.Container {
	return corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: EnvoyProxyContainerImage,
		Args:  []string{"--config-yaml", envoyConfig},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: int32(config.Port)},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: common.Int64Ptr(EnvoyProxyRunAsUser),
		},
	}
}

func (c *ContainerConstructor) createBasicAuthContainer(config *common.KodeResourceConfig) corev1.Container {
	return corev1.Container{
		Name:  "basic-auth-sidecar",
		Image: "your-basic-auth-sidecar-image:v0.0.0-latest",
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
