// envoy/config.go

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
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/common"
	"github.com/go-logr/logr"
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
	log logr.Logger
	ctx *cue.Context
}

//go:embed cue/bootstrap.cue
var embeddedCueBootstrapConfig string

//go:embed cue/schema.cue
var embeddedCueSchema string

// //go:embed cue/cluster_schema.cue
// var embeddedCueClusterSchema string

// //go:embed cue/filter_schema.cue
// var embeddedCueFilterSchema string

// //go:embed cue/listener_schema.cue
// var embeddedCueListenerSchema string

// //go:embed cue/route_schema.cue
// var embeddedCueRouteSchema string

// //go:embed cue/common_schema.cue
// var embeddedCueCommonSchema string

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{
		log: log,
		ctx: cuecontext.New(),
	}
}

func (g *BootstrapConfigGenerator) Generate(options common.BootstrapConfigOptions) (string, error) {
	g.log.Info("Starting bootstrap config generation")
	g.log.V(1).Info("Config options", "options", options)

	cueFiles := map[string]string{
		"schema.cue": embeddedCueSchema,
		// "cluster_schema.cue.cue": embeddedCueClusterSchema,
		// "filter_schema.cue":      embeddedCueFilterSchema,
		// "listener_schema.cue":    embeddedCueListenerSchema,
		// "route_schema.cue":       embeddedCueRouteSchema,
		// "common_schema.cue":      embeddedCueCommonSchema,
		"bootstrap.cue": embeddedCueBootstrapConfig,
	}

	var err error

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
	// return "", fmt.Errorf("FAILED TO DO STUFF: %w", err)

	g.log.Info("Successfully rendered bootstrap config")
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

	g.log.V(1).Info("Loading CUE instance")
	inst := load.Instances(filepaths, nil)[0]
	if inst.Err != nil {
		return cue.Value{}, fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	g.log.V(1).Info("Building CUE instance")
	value := g.ctx.BuildInstance(inst)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to build CUE instance: %w", inst.Err)
	}

	g.log.V(1).Info("Successfully loaded and built CUE instance")
	g.log.V(2).Info("CUE instance", "value", value)
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
		value, err = common.EncodeAndFillPath(g.ctx, value, p.parsePath, p.valuePath, p.schema, p.data)
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
