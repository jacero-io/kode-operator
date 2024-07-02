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

package common

import (
	"fmt"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// BootstrapConfigOptions contains options for generating Envoy bootstrap config
type BootstrapConfigOptions struct {
	HTTPFilters []kodev1alpha1.HTTPFilter
	Clusters    []kodev1alpha1.Cluster
	ServicePort int32
	ExposePort  int32
}

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
	Kode                 kodev1alpha1.Kode
	Templates            Templates
	InitPluginConfig     kodev1alpha1.InitPluginSpec
	Labels               map[string]string
	Volumes              []corev1.Volume
	VolumeMounts         []corev1.VolumeMount
	VolumeMountName      string
	PersistentVolumeName string
}

// ReconcileResult represents the result of a reconciliation
type ReconcileResult struct {
	Requeue      bool
	RequeueAfter int64
	Error        error
}

type TemplateNotFoundError struct {
	NamespacedName types.NamespacedName
	Kind           string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Kind, e.NamespacedName)
}
