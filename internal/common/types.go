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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type CommonConfig struct {
	Labels    map[string]string
	Name      string
	Namespace string
}

// KodeResourceConfig holds configuration for Kode resources
type KodeResourceConfig struct {
	CommonConfig CommonConfig
	KodeSpec     kodev1alpha2.KodeSpec
	Credentials  kodev1alpha2.CredentialsSpec
	Port         int32

	SecretName      string
	StatefulSetName string
	PVCName         string
	ServiceName     string

	UserInitPlugins []kodev1alpha2.InitPluginSpec
	Containers      []corev1.Container
	InitContainers  []corev1.Container

	Template *kodev1alpha2.Template
}

// EntryPointResourceConfig holds configuration for EntryPoint resources
type EntryPointResourceConfig struct {
	CommonConfig CommonConfig

	EntryPointSpec kodev1alpha2.EntryPointSpec
}

type TemplateNotFoundError struct {
	NamespacedName types.NamespacedName
	Kind           string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Kind, e.NamespacedName)
}
