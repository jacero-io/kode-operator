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

package controller

import (
	"fmt"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
)

// type ContainerRestartPolicy string

const (
	// RestartPolicyAlways ContainerRestartPolicy = "Always"
	envoyCfgFileName         = "bootstrap.yaml"
	EnvoyProxyContainerImage = "envoyproxy/envoy:v1.30.0"
	EnvoyProxyContainerName  = "envoy-proxy"
)

// GetRenderedBootstrapConfigOptions contains the options for rendering the bootstrap config.
func GetRenderedBootstrapConfig() (string, error) {
	// Create a new CUE context
	ctx := cuecontext.New()

	// Load the CUE instance from the bootstrap.cue and bootstrap_schema.cue files
	inst := load.Instances([]string{"bootstrap.cue", "bootstrap_schema.cue"}, nil)[0]
	if inst.Err != nil {
		return "", fmt.Errorf("failed to load instance: %v", inst.Err)
	}

	// Build the CUE value from the instance
	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return "", fmt.Errorf("failed to build instance: %v", value.Err())
	}

	// Convert the CUE value to YAML
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %v", err)
	}

	// Return the resulting YAML as a string
	return string(yamlBytes), nil
}

func constructEnvoyProxyContainer(kodeTemplate *kodev1alpha1.KodeTemplate, envoyProxyTemplate *kodev1alpha1.EnvoyProxyTemplate) (corev1.Container, error) {
	config, err := GetRenderedBootstrapConfig()
	if err != nil {
		return corev1.Container{}, err
	}
	return corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: EnvoyProxyContainerImage,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: 8000},
		},
		// RestartPolicy: "Always",
	}, nil
}
