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
	"os"
	"path/filepath"
	"strconv"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
)

// TODO: Improve how CUE files are loaded and used in the code.

//go:embed cue/bootstrap_schema.cue
var embeddedCueSchemaFile string

//go:embed cue/bootstrap.cue
var embeddedBootstrapCueFile string

const (
	// RestartPolicyAlways ContainerRestartPolicy = "Always"
	EnvoyProxyContainerName = "envoy-proxy"
	EnvoyProxyRunAsUser     = 1111
	ProxyInitContainerName  = "proxy-init"
	ProxyInitContainerImage = "openpolicyagent/proxy_init:v8"
	RouterFilterType        = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
)

func writeEmbeddedFilesToTempDir() (string, error) {
	tempDir := os.TempDir()

	files := map[string]string{
		"bootstrap_schema.cue": embeddedCueSchemaFile,
		"bootstrap.cue":        embeddedBootstrapCueFile,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write file %s", name)
		}
	}

	return tempDir, nil
}

// BootstrapConfigOptions contains the options for rendering the bootstrap config.
type BootstrapConfigOptions struct {
	HTTPFilters []kodev1alpha1.HTTPFilter
	Clusters    []kodev1alpha1.Cluster
	ServicePort int32
}

// Define the Router filter
var RouterFilter = kodev1alpha1.HTTPFilter{
	Name: "envoy.filters.http.router",
	TypedConfig: runtime.RawExtension{
		Raw: []byte(fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType)),
	},
}

func EnsureRouterFilter(filters []kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	for i, filter := range filters {
		if string(filter.TypedConfig.Raw) == fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType) {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	return append(filters, RouterFilter)
}

func unifyAndValidate(schema, value cue.Value) (cue.Value, error) {
	unifiedValue := schema.Unify(value)
	if err := unifiedValue.Validate(); err != nil {
		return cue.Value{}, errors.Wrap(err, "failed to validate unified value")
	}
	return unifiedValue, nil
}

func encodeToYAML(value cue.Value) (string, error) {
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", errors.Wrap(err, "failed to encode YAML")
	}
	return string(yamlBytes), nil
}

// encodeAndFillPath encodes the provided data and fills it into the CUE value at the specified path.
// It takes a CUE context, the CUE value, the CUE parse path, the CUE schema, the CUE schema string, and the data to encode and fill.
// If a CUE schema is provided, it validates the encoded value against the schema before filling it into the CUE value.
// Returns an error if encoding, validation, or filling fails.
func encodeAndFillPath(ctx *cue.Context, value cue.Value, cueParsePath string, cueSchema string, cueSchemaString string, data interface{}) error {
	var cueSchemaValue cue.Value
	if cueSchema != "" {
		cueSchemaValue = ctx.CompileString(cueSchemaString).LookupPath(cue.ParsePath(cueParsePath))
	}
	encodedValue := ctx.Encode(data)
	if encodedValue.Err() != nil {
		return errors.Wrap(encodedValue.Err(), fmt.Sprintf("failed to encode data for path %s", cueParsePath))
	}
	if cueSchema != "" {
		unifiedValue, err := unifyAndValidate(cueSchemaValue, encodedValue)
		if err != nil {
			return err
		}
		// Fill the unified value into the CUE value at the specified path
		value = value.FillPath(cue.ParsePath(cueParsePath), unifiedValue)
	} else {
		// Fill the encoded value into the CUE value at the specified path
		value = value.FillPath(cue.ParsePath(cueParsePath), encodedValue)
	}
	return nil
}

// getBootstrapConfig generates the bootstrap configuration based on the provided options.
// It reads the CUE schema file, builds a CUE instance, encodes and fills the required paths,
// and finally encodes the resulting value to YAML format.
// The generated bootstrap configuration is returned as a string.
// If any error occurs during the process, it is returned along with an empty string.
func getBootstrapConfig(log logr.Logger, options BootstrapConfigOptions) (string, error) {
	log.Info("Starting getBootstrapConfig", "options", options)

	cueSchemaString := string(embeddedCueSchemaFile)

	ctx := cuecontext.New()

	// Ensure that the Router filter is included in the HTTP filters
	options.HTTPFilters = EnsureRouterFilter(options.HTTPFilters)

	// Write the embedded files to a temporary directory
	tempDir, err := writeEmbeddedFilesToTempDir()
	if err != nil {
		log.Error(err, "Failed to write embedded files to temp directory")
		return "", errors.Wrap(err, "failed to write embedded files to temp directory")
	}
	defer os.RemoveAll(tempDir)

	// Load the CUE instance from the temporary directory
	cueFiles := []string{
		filepath.Join(tempDir, "bootstrap_schema.cue"),
		filepath.Join(tempDir, "bootstrap.cue"),
	}
	inst := load.Instances(cueFiles, nil)[0]
	if inst.Err != nil {
		log.Error(inst.Err, "Failed to load CUE instance")
		return "", errors.Wrap(inst.Err, "failed to load CUE instance")
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		log.Error(value.Err(), "Failed to build CUE instance")
		return "", errors.Wrap(value.Err(), "failed to build CUE instance")
	}

	type encodeAndFillPathArgs struct {
		cueParsePath    string
		cueSchema       string
		cueSchemaString string
		data            interface{}
	}

	args := []encodeAndFillPathArgs{
		{"#GoHttpFilters", "#HTTPFilters", cueSchemaString, options.HTTPFilters},
		{"#GoClusters", "#Clusters", cueSchemaString, options.Clusters},
		{"#GoLocalServicePort", "#Port", cueSchemaString, uint32(options.ServicePort)},
	}

	for _, arg := range args {
		log.Info("Encoding and filling path", "path", arg.cueParsePath, "data", arg.data)
		if err := encodeAndFillPath(ctx, value, arg.cueParsePath, arg.cueSchema, arg.cueSchemaString, arg.data); err != nil {
			log.Error(err, "Failed to encode and fill path", "path", arg.cueParsePath)
			return "", err
		}
	}

	result, err := encodeToYAML(value)
	if err != nil {
		log.Error(err, "Failed to encode to YAML")
		return "", err
	}

	log.Info("Successfully rendered bootstrap config")
	return result, nil
}

// constructEnvoyProxyContainer constructs the Envoy Proxy container and the Init container.
// It takes a logger, the sharedKodeTemplateSpec, and the sharedEnvoyProxyTemplateSpec as input.
// It returns the constructed Envoy Proxy container, the Init container, and an error if any.
func constructEnvoyProxyContainer(log logr.Logger,
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyTemplateSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec) (corev1.Container, corev1.Container, error) {
	var modifiedLog logr.Logger
	modifiedLog = log.WithName("constructEnvoyProxyContainer")

	modifiedLog.Info("Constructing Envoy Proxy container", "sharedKodeTemplateSpec", sharedKodeTemplateSpec, "sharedEnvoyProxyTemplateSpec", sharedEnvoyProxyTemplateSpec)

	servicePort := sharedKodeTemplateSpec.Port
	config, err := getBootstrapConfig(modifiedLog, BootstrapConfigOptions{
		HTTPFilters: sharedEnvoyProxyTemplateSpec.HTTPFilters,
		Clusters:    sharedEnvoyProxyTemplateSpec.Clusters,
		ServicePort: servicePort,
	})
	if err != nil {
		log.Error(err, "Failed to get bootstrap config")
		return corev1.Container{}, corev1.Container{}, err
	}

	envoyContainer := corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: sharedEnvoyProxyTemplateSpec.Image,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "envoy-http", ContainerPort: servicePort},
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

	log.Info("Successfully constructed Envoy Proxy and Init containers")
	return envoyContainer, initContainer, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
