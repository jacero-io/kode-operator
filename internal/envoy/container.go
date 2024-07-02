/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package envoy

import (
	"strconv"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

type ContainerConstructor struct {
	log             logr.Logger
	configGenerator *BootstrapConfigGenerator
}

func NewContainerConstructor(log logr.Logger, configGenerator *BootstrapConfigGenerator) *ContainerConstructor {
	return &ContainerConstructor{
		log:             log,
		configGenerator: configGenerator,
	}
}

func (c *ContainerConstructor) ConstructEnvoyProxyContainer(
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyTemplateSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec,
) (corev1.Container, corev1.Container, error) {
	c.log.Info("Constructing Envoy Proxy container")

	config, err := c.configGenerator.Generate(common.BootstrapConfigOptions{
		HTTPFilters: sharedEnvoyProxyTemplateSpec.HTTPFilters,
		Clusters:    sharedEnvoyProxyTemplateSpec.Clusters,
		ServicePort: common.InternalServicePort,
		ExposePort:  common.ExternalServicePort,
	})
	if err != nil {
		c.log.Error(err, "Failed to generate bootstrap config")
		return corev1.Container{}, corev1.Container{}, err
	}

	envoyContainer := corev1.Container{
		Name:  common.EnvoyProxyContainerName,
		Image: sharedEnvoyProxyTemplateSpec.Image,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "envoy-http", ContainerPort: common.ExternalServicePort},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: common.Int64Ptr(common.EnvoyProxyRunAsUser),
		},
	}

	proxySetupContainer := corev1.Container{
		Name:  common.ProxyInitContainerName,
		Image: common.ProxyInitContainerImage,
		Args:  []string{"-p", strconv.Itoa(int(common.ExternalServicePort)), "-u", strconv.FormatInt(common.EnvoyProxyRunAsUser, 16)},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsNonRoot: common.BoolPtr(false),
			RunAsUser:    common.Int64Ptr(0),
		},
	}

	c.log.Info("Successfully constructed Envoy Proxy and Init containers")
	return envoyContainer, proxySetupContainer, nil
}
