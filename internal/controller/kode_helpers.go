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
	"fmt"
	"strconv"

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
	EnvoyProxyRunAsUser     = 1111
	ProxyInitContainerName  = "proxy-init"
	ProxyInitContainerImage = "openpolicyagent/proxy_init:v8"
)

//go:embed bootstrap_schema.cue
var schemaFile string

// GetRenderedBootstrapConfigOptions contains the options for rendering the bootstrap config.
type GetRenderedBootstrapConfigOptions struct {
	CueFiles    []string
	HTTPFilters []kodev1alpha1.HTTPFilter
	Clusters    []kodev1alpha1.Cluster
	ServicePort int32
}

// Define the Router filter
var RouterFilter = kodev1alpha1.HTTPFilter{
	Name: "envoy.filters.http.router",
	TypedConfig: runtime.RawExtension{
		Raw: []byte(`{"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}`),
	},
}

func EnsureRouterFilter(filters []kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	for i, filter := range filters {
		if string(filter.TypedConfig.Raw) == `{"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}` {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	return append(filters, RouterFilter)
}

func unifyAndValidate(ctx *cue.Context, schema, value cue.Value) (cue.Value, error) {
	unifiedValue := schema.Unify(value)
	if err := unifiedValue.Validate(); err != nil {
		return cue.Value{}, fmt.Errorf("failed to unify value with schema: %w", err)
	}
	return unifiedValue, nil
}

func encodeToYAML(value cue.Value) (string, error) {
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// GetRenderedBootstrapConfig renders the bootstrap config using the provided options.
func GetRenderedBootstrapConfig(log *logr.Logger, options GetRenderedBootstrapConfigOptions) (string, error) {
	options.HTTPFilters = EnsureRouterFilter(options.HTTPFilters)

	ctx := cuecontext.New()
	httpFiltersSchema := ctx.CompileString(schemaFile).LookupPath(cue.ParsePath("#HTTPFilters"))
	clustersSchema := ctx.CompileString(schemaFile).LookupPath(cue.ParsePath("#Clusters"))

	// Load the CUE instance from the provided files
	inst := load.Instances(options.CueFiles, nil)[0]
	if inst.Err != nil {
		return "", fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return "", fmt.Errorf("failed to build CUE instance: %w", value.Err())
	}

	httpFiltersValue := ctx.Encode(options.HTTPFilters)
	if httpFiltersValue.Err() != nil {
		return "", fmt.Errorf("failed to encode HTTPFilters: %w", httpFiltersValue.Err())
	}
	unifiedHTTPFilterValues, err := unifyAndValidate(ctx, httpFiltersSchema, httpFiltersValue)
	if err != nil {
		return "", err
	}
	value = value.FillPath(cue.ParsePath("#GoHttpFilters"), unifiedHTTPFilterValues)

	clustersValue := ctx.Encode(options.Clusters)
	if clustersValue.Err() != nil {
		return "", fmt.Errorf("failed to encode Clusters: %w", clustersValue.Err())
	}
	unifiedClusterValues, err := unifyAndValidate(ctx, clustersSchema, clustersValue)
	if err != nil {
		return "", err
	}
	value = value.FillPath(cue.ParsePath("#GoClusters"), unifiedClusterValues)

	servicePortValue := ctx.Encode(options.ServicePort)
	if servicePortValue.Err() != nil {
		return "", fmt.Errorf("failed to encode Clusters: %w", servicePortValue.Err())
	}
	value = value.FillPath(cue.ParsePath("#GoLocalServicePort"), servicePortValue)

	return encodeToYAML(value)
}

// constructEnvoyProxyContainer constructs the Envoy Proxy container
func constructEnvoyProxyContainer(log *logr.Logger,
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyTemplateSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec) (corev1.Container, corev1.Container, error) {

	ServicePort := sharedKodeTemplateSpec.Port
	config, err := GetRenderedBootstrapConfig(log, GetRenderedBootstrapConfigOptions{
		CueFiles:    []string{"internal/controller/bootstrap.cue", "internal/controller/bootstrap_schema.cue"},
		HTTPFilters: sharedEnvoyProxyTemplateSpec.HTTPFilters,
		Clusters:    sharedEnvoyProxyTemplateSpec.Clusters,
		ServicePort: ServicePort,
	})
	if err != nil {
		return corev1.Container{}, corev1.Container{}, err
	}

	envoyContainer := corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: sharedEnvoyProxyTemplateSpec.Image,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: ServicePort},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: int64Ptr(EnvoyProxyRunAsUser),
		},
	}

	initContainer := corev1.Container{
		Name:  ProxyInitContainerName,
		Image: ProxyInitContainerImage,
		// Documentation: https://github.com/open-policy-agent/opa-envoy-plugin/blob/main/proxy_init/proxy_init.sh
		Args: []string{"-p", "8000", "-u", strconv.FormatInt(*int64Ptr(EnvoyProxyRunAsUser), 16)},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsNonRoot: boolPtr(false),
			RunAsUser:    int64Ptr(0),
		},
	}

	return envoyContainer, initContainer, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
