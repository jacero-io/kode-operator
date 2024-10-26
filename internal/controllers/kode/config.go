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
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/pkg/constant"
	corev1 "k8s.io/api/core/v1"
)

func InitKodeResourcesConfig(
	kode *kodev1alpha2.Kode,
	template *kodev1alpha2.Template) *common.KodeResourceConfig {
	var credentials *kodev1alpha2.CredentialsSpec

	if kode.Spec.Credentials == nil {
		credentials = &kodev1alpha2.CredentialsSpec{
			Username: constant.DefaultUsername,
		}
	} else {
		credentials = kode.Spec.Credentials
	}

	return &common.KodeResourceConfig{
		CommonConfig: common.CommonConfig{
			Labels:    createLabels(kode, template),
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		KodeSpec:    kode.Spec,
		Credentials: credentials,

		UserInitPlugins: kode.Spec.InitPlugins,
		Containers:      []corev1.Container{},
		InitContainers:  []corev1.Container{},

		Template: template,
		Port:     template.Port,
	}
}

func createLabels(kode *kodev1alpha2.Kode, template *kodev1alpha2.Template) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       kode.Name,
		"app.kubernetes.io/managed-by": "kode-operator",
		"kode.jacero.io/name":          kode.Name,
		"template.kode.jacero.io/name": string(template.Name),
	}
}
