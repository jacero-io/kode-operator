// internal/controller/resource_config.go

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

package controller

import (
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
)

func InitKodeResourcesConfig(
	kode *kodev1alpha1.Kode,
	templates *common.Templates) *common.KodeResourcesConfig {

	var localServicePort int32
	var externalServicePort int32

	// If EnvoyProxyConfig is not specified, use KodeTemplate.Port
	localServicePort = templates.KodeTemplate.Port
	externalServicePort = templates.KodeTemplate.Port
	// If EnvoyProxyConfig is specified, use the default local service port
	if templates.EnvoyProxyConfigName != "" {
		localServicePort = common.DefaultLocalServicePort
		externalServicePort = templates.KodeTemplate.Port
	}

	pvcName := kode.Name + "-pvc"
	serviceName := kode.Name + "-svc"

	return &common.KodeResourcesConfig{
		Kode:                *kode,
		KodeName:            kode.Name,
		KodeNamespace:       kode.Namespace,
		PVCName:             pvcName,
		ServiceName:         serviceName,
		Templates:           *templates,
		Labels:              CreateLabels(kode, templates),
		UserInitPlugins:     kode.Spec.InitPlugins,
		TemplateInitPlugins: templates.KodeTemplate.InitPlugins,
		LocalServicePort:    localServicePort,
		ExternalServicePort: externalServicePort,
	}
}

func CreateLabels(kode *kodev1alpha1.Kode, templates *common.Templates) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":           kode.Name,
		"app.kubernetes.io/managed-by":     "kode-operator",
		"kode.jacero.io/name":              kode.Name,
		"template.kode.jacero.io/name":     templates.KodeTemplateName,
		"envoy-config.kode.jacero.io/name": templates.EnvoyProxyConfigName,
	}
}
