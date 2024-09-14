// internal/controllers/entrypoint/config.go

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

package entrypoint

import (
	"fmt"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

func InitEntryPointResourcesConfig(entryPoint *kodev1alpha2.EntryPoint) *common.EntryPointResourceConfig {

	var protocol kodev1alpha2.Protocol
	var identityReference kodev1alpha2.IdentityReference
	var gatewayName string

	if entryPoint.Spec.GatewaySpec.ExistingGatewayRef != nil {
		gatewayName = string(entryPoint.Spec.GatewaySpec.ExistingGatewayRef.Name)
	} else {
		gatewayName = fmt.Sprintf("%s-gateway", entryPoint.Name)
	}

	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.CertificateRefs != nil {
		protocol = kodev1alpha2.ProtocolHTTPS
	} else {
		protocol = kodev1alpha2.ProtocolHTTP
	}

	if entryPoint.Spec.AuthSpec != nil && entryPoint.Spec.AuthSpec.IdentityReference != nil {
		identityReference = *entryPoint.Spec.AuthSpec.IdentityReference
	}

	return &common.EntryPointResourceConfig{
		CommonConfig: common.CommonConfig{
			Labels:    createLabels(entryPoint),
			Name:      entryPoint.Name,
			Namespace: entryPoint.Namespace,
		},
		EntryPointSpec: entryPoint.Spec,

		GatewayName:      gatewayName,
		GatewayNamespace: entryPoint.Namespace,

		Protocol:          protocol,
		IdentityReference: &identityReference,
	}
}

func createLabels(entrypoint *kodev1alpha2.EntryPoint) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       entrypoint.Name,
		"app.kubernetes.io/managed-by": "kode-operator",
	}
}
