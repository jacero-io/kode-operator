package kode

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	basconfig "github.com/jacero-io/basic-auth-sidecar/pkg/config"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resourcev1"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/envoy"
)

func ensureSidecarContainers(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) ([]corev1.Container, []corev1.Container, error) {
	log := r.GetLog().WithName("SidecarContainerEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	log.V(1).Info("Ensuring sidecar containers")

	// Check if Basic Auth ConfigMap is needed
	// useBasicAuth := false
	// if config.Template != nil && config.Template.EntryPointSpec != nil &&
	// 	config.Template.EntryPointSpec.AuthSpec != nil {
	// 	useBasicAuth = config.Template.EntryPointSpec.AuthSpec.AuthType == "basicAuth"
	// }

	// Create Basic Auth ConfigMap if needed
	// if useBasicAuth {
	// 	basConfigMap := &corev1.ConfigMap{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name:      fmt.Sprintf("%s-basic-auth-config", kode.Name),
	// 			Namespace: kode.Namespace,
	// 			Labels:    config.CommonConfig.Labels,
	// 		},
	// 	},
	// 	_, err := resource.CreateOrPatch(ctx, basConfigMap, func() error {
	// 		if basConfigMap, err := createBasicAuthConfigMap(ctx, r, resource, kode, config); err != nil {
	// 			return nil, nil, fmt.Errorf("failed to create Basic Auth ConfigMap: %w", err)
	// 		}
	// 		return controllerutil.SetControllerReference(kode, basConfigMap, r.GetScheme())
	// 	}),
	// },

	// Create Envoy config
	envoyConfigGenerator := envoy.NewBootstrapConfigGenerator(log)

	// Create sidecar containers
	envoyConstructor := envoy.NewContainerConstructor(log, envoyConfigGenerator)
	containers, initContainers, err := envoyConstructor.ConstructEnvoyContainers(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct Envoy containers: %w", err)
	}

	return containers, initContainers, nil
}

func constructBasicAuthConfigMap(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (*corev1.ConfigMap, error) {
	log := r.GetLog().WithName("BasicAuthConfigMapConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	log.V(1).Info("Constructing Basic Auth ConfigMap")

	basicAuthConfig := basconfig.NewDefaultConfig()

	yamlData, err := yaml.Marshal(basicAuthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal basic auth config: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-basic-auth-config", kode.Name),
			Namespace: kode.Namespace,
		},
		Data: map[string]string{
			"config.yaml": string(yamlData),
		},
	}

	return configMap, nil
}
