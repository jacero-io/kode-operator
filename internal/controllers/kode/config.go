// internal/controllers/kode/config.go

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

package kode

import (
	"fmt"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
)

func InitKodeResourcesConfig(
	kode *kodev1alpha1.Kode,
	templates *common.Templates) *common.KodeResourceConfig {

	var localServicePort int32
	var externalServicePort int32
	var secretName string

	// If ExistingSecret is specified, use it
	if kode.Spec.Credentials.ExistingSecret != "" {
		secretName = kode.Spec.Credentials.ExistingSecret
	} else { // If ExistingSecret is not specified, use Kode.Name
		secretName = fmt.Sprintf("%s-auth", kode.Name)
	}

	localServicePort = templates.KodeTemplate.Port
	externalServicePort = templates.KodeTemplate.Port

	pvcName := GetPVCName(kode)
	serviceName := GetServiceName(kode)

	return &common.KodeResourceConfig{
		CommonConfig: common.CommonConfig{
			Labels:    createLabels(kode, templates),
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},

		KodeSpec:            kode.Spec,
		Credentials:         kodev1alpha1.CredentialsSpec{},
		TemplateInitPlugins: templates.KodeTemplate.ContainerSpec.InitPlugins,
		UserInitPlugins:     kode.Spec.InitPlugins,

		SecretName:      secretName,
		StatefulSetName: kode.Name,
		PVCName:         pvcName,
		ServiceName:     serviceName,

		Templates:      *templates,
		Containers:     []corev1.Container{},
		InitContainers: []corev1.Container{},

		LocalServicePort:    localServicePort,
		ExternalServicePort: externalServicePort,
	}
}

func createLabels(kode *kodev1alpha1.Kode, templates *common.Templates) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       kode.Name,
		"app.kubernetes.io/managed-by": "kode-operator",
		"kode.jacero.io/name":          kode.Name,
		"template.kode.jacero.io/name": templates.KodeTemplateName,
	}
}
