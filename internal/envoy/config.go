// envoy/config.go

/*
Copyright 2024 Emil Larsson.

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
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
)

const (
	RouterFilterType    = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	ExtAuthzFilterType  = "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz"
	BasicAuthFilterType = "type.googleapis.com/envoy.extensions.filters.http.basic_auth.v3.BasicAuth"
)

type BootstrapConfigGenerator struct {
	Log    logr.Logger
	CueCtx *cue.Context
}

//go:embed cue/bootstrap.cue
var embeddedCueBootstrapConfig string

//go:embed cue/schema.cue
var embeddedCueSchema string

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{
		Log:    log,
		CueCtx: cuecontext.New(),
	}
}

func (g *BootstrapConfigGenerator) Generate(options common.BootstrapConfigOptions) (string, error) {
	g.Log.Info("Starting bootstrap config generation")
	g.Log.V(1).Info("Config options", "options", options)

	var err error

	// Embedded CUE files
	cueFiles := map[string]string{
		"schema.cue":    embeddedCueSchema,
		"bootstrap.cue": embeddedCueBootstrapConfig,
	}

	// Add the basic auth filter if the auth type is basic
	if options.AuthConfig.AuthType == "basic" {
		credentials, err := generateBasicAuthConfig(options.Credentials.Username, options.Credentials.Password)
		if err != nil {
			return "", fmt.Errorf("failed to generate basic auth config: %w", err)
		}
		basicAuthFilter := kodev1alpha1.HTTPFilter{
			Name: "envoy.filters.http.basic_auth",
			TypedConfig: runtime.RawExtension{
				Raw: []byte(fmt.Sprintf(`{"@type": "%s", "users": {"inline_string": "%s"}}`, BasicAuthFilterType, credentials)),
			},
		}
		options.HTTPFilters = append([]kodev1alpha1.HTTPFilter{basicAuthFilter}, options.HTTPFilters...)
	}

	// Ensure that the Router filter is included in the HTTP filters
	options.HTTPFilters = g.ensureRouterFilter(options.HTTPFilters)

	// Write the embedded files to a temporary directory
	tempDir, err := g.writeEmbeddedFilesToTempDir(cueFiles)
	if err != nil {
		return "", fmt.Errorf("failed to write embedded files to temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Load and build the CUE instance
	value, err := g.loadAndBuildCueInstance(cueFiles, tempDir)
	if err != nil {
		return "", err
	}

	// Encode and fill paths
	value, err = g.encodeAndFillPaths(value, options)
	if err != nil {
		return "", err
	}

	// Encode to YAML
	result, err := g.encodeToYAML(value)
	if err != nil {
		return "", err
	}

	g.Log.Info("Successfully rendered bootstrap config")
	return result, nil
}

func (g *BootstrapConfigGenerator) ensureRouterFilter(filters []kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	routerFilter := kodev1alpha1.HTTPFilter{
		Name: "envoy.filters.http.router",
		TypedConfig: runtime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType)),
		},
	}

	for i, filter := range filters {
		if string(filter.TypedConfig.Raw) == fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType) {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	return append(filters, routerFilter)
}

func (g *BootstrapConfigGenerator) writeEmbeddedFilesToTempDir(files map[string]string) (string, error) {
	tempDir := os.TempDir()

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", name, err)
		}
	}

	return tempDir, nil
}

func (g *BootstrapConfigGenerator) loadAndBuildCueInstance(cueFiles map[string]string, tempDir string) (cue.Value, error) {
	var filepaths []string
	for name := range cueFiles {
		filepaths = append(filepaths, filepath.Join(tempDir, name))
	}

	g.Log.V(1).Info("Loading CUE instance")
	inst := load.Instances(filepaths, nil)[0]
	if inst.Err != nil {
		return cue.Value{}, fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	g.Log.V(1).Info("Building CUE instance")
	value := g.CueCtx.BuildInstance(inst)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to build CUE instance: %w", inst.Err)
	}

	g.Log.V(1).Info("Successfully loaded and built CUE instance")
	g.Log.V(1).Info("CUE instance", "value", value)
	return value, nil
}

func (g *BootstrapConfigGenerator) encodeAndFillPaths(value cue.Value, options common.BootstrapConfigOptions) (cue.Value, error) {

	paths := []struct {
		parsePath string
		valuePath string
		schema    string
		data      interface{}
	}{
		{"#HTTPFilters", "#GoHttpFilters", embeddedCueSchema, options.HTTPFilters},
		{"#Clusters", "#GoClusters", embeddedCueSchema, options.Clusters},
		{"#Port", "#GoLocalServicePort", embeddedCueSchema, uint32(options.LocalPort)},
		{"#Port", "#GoExternalServicePort", embeddedCueSchema, uint32(options.ExternalPort)},
	}

	for _, p := range paths {
		var err error
		value, err = encodeAndFillPath(g.CueCtx, value, p.parsePath, p.valuePath, p.schema, p.data)
		if err != nil {
			return cue.Value{}, fmt.Errorf("failed to encode and fill path %s: %w", p.parsePath, err)
		}
	}

	return value, nil
}

func (g *BootstrapConfigGenerator) encodeToYAML(value cue.Value) (string, error) {
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// EncodeAndFillPath encodes a data structure, fills it into a CUE value at a specified path, and validates the result
// The ctx is the CUE context
// The value is the CUE value to fill
// The parsePath is used to parse the schema
// The valuePath is used to fill the data structure into the CUE value
// The schema is used to validate the data structure
// The data is the data structure to encode and fill
// The function returns the updated CUE value and nil if successful
// If an error occurs, the function returns the original CUE value and the error
func encodeAndFillPath(ctx *cue.Context, value cue.Value, parsePath string, valuePath string, schema string, data interface{}) (cue.Value, error) {
	tempSchema := ctx.CompileString(schema).LookupPath(cue.ParsePath(parsePath))
	if tempSchema.Err() != nil {
		return value, fmt.Errorf("failed to parse path %s: %w", parsePath, tempSchema.Err())
	}

	valueAsCUE := ctx.Encode(data)
	if valueAsCUE.Err() != nil {
		return value, fmt.Errorf("failed to encode data: %w", valueAsCUE.Err())
	}

	unifiedValue, err := unifyAndValidate(tempSchema, valueAsCUE)
	if err != nil {
		return value, fmt.Errorf("failed to unify and validate: %w", err)
	}

	// Fill the unified value into the CUE value at the specified path
	value = value.FillPath(cue.ParsePath(valuePath), unifiedValue)
	if err := value.Err(); err != nil {
		return value, fmt.Errorf("failed to fill path %s: %w", valuePath, value.Err())
	}
	return value, nil
}

// unifyAndValidate unifies the schema and value, then validates the result
func unifyAndValidate(schema, value cue.Value) (cue.Value, error) {
	unifiedValue := schema.Unify(value)
	if err := unifiedValue.Validate(); err != nil {
		return cue.Value{}, fmt.Errorf("failed to validate unified value: %w", err)
	}
	return unifiedValue, nil
}


func generateBasicAuthConfig(username, password string) (string, error) {
	hash := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	if hash == "" {
		return "", fmt.Errorf("failed to generate basic auth config: hash is empty")
	}
	return hash, nil
}
