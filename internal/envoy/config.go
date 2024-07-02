// envoy/config.go

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

//go:embed cue/bootstrap_schema.cue
var embeddedCueSchemaFile string

//go:embed cue/bootstrap.cue
var embeddedBootstrapCueFile string

func NewBootstrapConfigGenerator(log logr.Logger) *BootstrapConfigGenerator {
	return &BootstrapConfigGenerator{
		log: log,
		ctx: cuecontext.New(),
	}
}

func (g *BootstrapConfigGenerator) Generate(options common.BootstrapConfigOptions) (string, error) {
	g.log.Info("Starting bootstrap config generation", "options", options)

	var err error

	// Ensure that the Router filter is included in the HTTP filters
	options.HTTPFilters = g.ensureRouterFilter(options.HTTPFilters)

	// Write the embedded files to a temporary directory
	tempDir, err := g.writeEmbeddedFilesToTempDir()
	if err != nil {
		return "", fmt.Errorf("failed to write embedded files to temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Load and build the CUE instance
	value, err := g.loadAndBuildCueInstance(tempDir)
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

func (g *BootstrapConfigGenerator) writeEmbeddedFilesToTempDir() (string, error) {
	tempDir := os.TempDir()
	files := map[string]string{
		"bootstrap_schema.cue": embeddedCueSchemaFile,
		"bootstrap.cue":        embeddedBootstrapCueFile,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", name, err)
		}
	}

	return tempDir, nil
}

func (g *BootstrapConfigGenerator) loadAndBuildCueInstance(tempDir string) (cue.Value, error) {
	cueFiles := []string{
		filepath.Join(tempDir, "bootstrap_schema.cue"),
		filepath.Join(tempDir, "bootstrap.cue"),
	}

	g.log.V(1).Info("Loading CUE instance")
	inst := load.Instances(cueFiles, nil)[0]
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
		{"#HTTPFilters", "#GoHttpFilters", embeddedCueSchemaFile, options.HTTPFilters},
		{"#Clusters", "#GoClusters", embeddedCueSchemaFile, options.Clusters},
		{"#Port", "#GoLocalServicePort", embeddedCueSchemaFile, uint32(options.LocalPort)},
		{"#Port", "#GoExternalServicePort", embeddedCueSchemaFile, uint32(options.ExternalPort)},
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
