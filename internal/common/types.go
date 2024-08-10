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

package common

import (
	"fmt"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Templates holds the fetched template configurations
type Templates struct {
	KodeTemplate              *kodev1alpha1.SharedKodeTemplateSpec
	KodeTemplateName          string
	KodeTemplateNamespace     string
}

// KodeResourcesConfig holds configuration for Kode resources
type KodeResourcesConfig struct {
	KodeSpec            kodev1alpha1.KodeSpec
	Labels              map[string]string

	KodeName            string
	KodeNamespace       string

	Credentials         kodev1alpha1.CredentialsSpec

	SecretName          string
	StatefulSetName     string
	PVCName             string
	ServiceName         string

	Templates           Templates
	Containers          []corev1.Container
	InitContainers      []corev1.Container

	TemplateInitPlugins []kodev1alpha1.InitPluginSpec
	UserInitPlugins     []kodev1alpha1.InitPluginSpec

	LocalServicePort    int32
	ExternalServicePort int32
}

// EntryPointResourceConfig holds configuration for EntryPoint resources
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
	Credentials  kodev1alpha1.CredentialsSpec
}

type TemplateNotFoundError struct {
	NamespacedName types.NamespacedName
	Kind           string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Kind, e.NamespacedName)
}
