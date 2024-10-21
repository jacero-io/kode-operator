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
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
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
	Credentials  *kodev1alpha2.CredentialsSpec
	ServiceName  string

	UserInitPlugins []kodev1alpha2.InitPluginSpec
	Containers      []corev1.Container
	InitContainers  []corev1.Container

	Template *kodev1alpha2.Template
	Port     kodev1alpha2.Port
}

// EntryPointResourceConfig holds configuration for EntryPoint resources
type EntryPointResourceConfig struct {
	CommonConfig   CommonConfig
	EntryPointSpec kodev1alpha2.EntryPointSpec

	GatewayName      string
	GatewayNamespace string

	Protocol          kodev1alpha2.Protocol
	IdentityReference *kodev1alpha2.IdentityReference
}

func (c *EntryPointResourceConfig) IsHTTPS() bool {
	return c.Protocol == kodev1alpha2.ProtocolHTTPS
}

func (c *EntryPointResourceConfig) HasIdentityReference() bool {
	return c.IdentityReference != nil
}
