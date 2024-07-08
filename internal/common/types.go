/*
Copyright emil@jacero.se 2024.

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

package common

import (
	"fmt"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// Templates holds the fetched template configurations
type Templates struct {
	KodeTemplate              *kodev1alpha1.SharedKodeTemplateSpec
	EnvoyProxyConfig          *kodev1alpha1.SharedEnvoyProxyConfigSpec
	KodeTemplateName          string
	KodeTemplateNamespace     string
	EnvoyProxyConfigName      string
	EnvoyProxyConfigNamespace string
}

// KodeResourcesConfig holds configuration for Kode resources
type KodeResourcesConfig struct {
	Kode                kodev1alpha1.Kode
	KodeName            string
	KodeNamespace       string
	PVCName             string
	ServiceName         string
	Templates           Templates
	UserInitPlugins     []kodev1alpha1.InitPluginSpec
	TemplateInitPlugins []kodev1alpha1.InitPluginSpec
	Labels              map[string]string
	LocalServicePort    int32
	ExternalServicePort int32
}

type EntryPointResourceConfig struct {
	EntryPoint          kodev1alpha1.EntryPoint
	Labels              map[string]string
	EntryPointName      string
	EntryPointNamespace string
	EntryPointService   string
	EntryPointURL       string	
}

// BootstrapConfigOptions contains options for generating Envoy bootstrap config
type BootstrapConfigOptions struct {
	HTTPFilters  []kodev1alpha1.HTTPFilter
	Clusters     []kodev1alpha1.Cluster
	LocalPort    int32
	ExternalPort int32
}

type TemplateNotFoundError struct {
	NamespacedName types.NamespacedName
	Kind           string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Kind, e.NamespacedName)
}
