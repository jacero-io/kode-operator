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
	_ "embed"
	"encoding/json"
	"fmt"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
)

// type ContainerRestartPolicy string

const (
	// RestartPolicyAlways ContainerRestartPolicy = "Always"
	EnvoyProxyContainerName = "envoy-proxy"
)

//go:embed bootstrap_schema.cue
var schemaFile string

// GetRenderedBootstrapConfigOptions contains the options for rendering the bootstrap config.
type GetRenderedBootstrapConfigOptions struct {
	CueFiles    []string
	HTTPFilters []kodev1alpha1.HTTPFilter
}

// Define the Router filter
var RouterFilter = kodev1alpha1.HTTPFilter{
	Name: "envoy.filters.http.router",
	TypedConfig: runtime.RawExtension{
		Raw: []byte(`{"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}`),
	},
}

// Utility function to convert a filter configuration to runtime.RawExtension
func toRawExtension(filterConfig interface{}) (*runtime.RawExtension, error) {
	data, err := json.Marshal(filterConfig)
	if err != nil {
		return nil, err
	}
	return &runtime.RawExtension{Raw: data}, nil
}

// EnsureRouterFilter ensures that the Router filter is added last in the list of filters.
func EnsureRouterFilter(filters []kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	// Check if the Router filter is already present
	for _, filter := range filters {
		if string(filter.TypedConfig.Raw) == `{"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}` {
			// Remove the Router filter if it's not the last one
			filters = append(filters[:len(filters)-1], filters[len(filters):]...)
			break
		}
	}
	// Append the Router filter as the last filter
	return append(filters, RouterFilter)
}

// GetRenderedBootstrapConfig renders the bootstrap config using the provided options.
func GetRenderedBootstrapConfig(log *logr.Logger, options GetRenderedBootstrapConfigOptions) (string, error) {
	// Ensure the Router filter is included
	options.HTTPFilters = EnsureRouterFilter(options.HTTPFilters)

	// Create a new CUE context
	ctx := cuecontext.New()
	schema := ctx.CompileString(schemaFile).LookupPath(cue.ParsePath("#HTTPFilters"))
	log.V(1).Info("Schema", "schema", schema)

	// Load the CUE instance from the provided files
	inst := load.Instances(options.CueFiles, nil)[0]
	if inst.Err != nil {
		return "", fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	// Build the CUE value from the instance
	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return "", fmt.Errorf("failed to build CUE instance: %w", value.Err())
	}

	// Add HTTPFilters to the CUE context
	filtersValue := ctx.Encode(options.HTTPFilters)
	if filtersValue.Err() != nil {
		return "", fmt.Errorf("failed to encode HTTPFilters: %w", filtersValue.Err())
	}
	log.V(1).Info("HTTPFilters", "filtersValue", filtersValue)
	unified := schema.Unify(filtersValue)
	if err := unified.Validate(); err != nil {
		return "", fmt.Errorf("failed to unify HTTPFilters with Schema: %w", filtersValue.Err())
	}
	log.V(1).Info("Unified", "unified", unified)

	value = value.FillPath(cue.ParsePath("#GoHttpFilters"), filtersValue)

	// Convert the CUE value to YAML
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}

	// Return the resulting YAML as a string
	return string(yamlBytes), nil
}

// constructEnvoyProxyContainer constructs the Envoy Proxy container
func constructEnvoyProxyContainer(log *logr.Logger,
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyTemplateSpec *kodev1alpha1.EnvoyProxyConfigSpec) (corev1.Container, error) {

	// ContainerPort := templateSpec.Spec.Port
	config, err := GetRenderedBootstrapConfig(log, GetRenderedBootstrapConfigOptions{
		CueFiles:    []string{"internal/controller/bootstrap.cue"},
		HTTPFilters: sharedEnvoyProxyTemplateSpec.HTTPFilters,
	})
	if err != nil {
		return corev1.Container{}, fmt.Errorf("error rendering bootstrap config: %w", err)
	}

	return corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: sharedEnvoyProxyTemplateSpec.Image,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: 8000},
		},
	}, nil
}
