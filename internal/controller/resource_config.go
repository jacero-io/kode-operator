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
	// If EnvoyProxyConfig is specified, use ExternalServicePort
	if templates.EnvoyProxyConfigName != "" {
		localServicePort = common.DefaultLocalServicePort
		externalServicePort = templates.KodeTemplate.Port
	}

	return &common.KodeResourcesConfig{
		Kode:                *kode,
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
		"app.kubernetes.io/name":           "kode-" + kode.Name,
		"app.kubernetes.io/managed-by":     "kode-operator",
		"kode.jacero.io/name":              kode.Name,
		"template.kode.jacero.io/name":     templates.KodeTemplateName,
		"envoy-config.kode.jacero.io/name": templates.EnvoyProxyConfigName,
	}
}
