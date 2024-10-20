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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"

	"github.com/jacero-io/kode-operator/pkg/envoy"
)

// ensureSidecarContainers ensures that the Envoy container exists for the Kode instance
func ensureSidecarContainers(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithName("SidecarContainerEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	log.V(1).Info("Ensuring sidecar containers")

	useBasicAuth := false
	if config.Template != nil && config.Template.EntryPointSpec != nil &&
		config.Template.EntryPointSpec.AuthSpec != nil {
		useBasicAuth = config.Template.EntryPointSpec.AuthSpec.AuthType == "basicAuth"
	}

	// Create Envoy config
	envoyConfigGenerator := envoy.NewBootstrapConfigGenerator(log)
	envoyConfig, err := envoyConfigGenerator.GenerateEnvoyConfig(config, useBasicAuth)
	if err != nil {
		return fmt.Errorf("failed to generate Envoy config: %w", err)
	}

	// Create sidecar containers
	envoyConstructor := envoy.NewContainerConstructor(log, envoyConfigGenerator)
	containers, initContainers, err := envoyConstructor.ConstructEnvoyContainers(config)
	if err != nil {
		return fmt.Errorf("failed to construct Envoy containers: %w", err)
	}

	// Add containers to the Kode resource
	kode.Spec.InitPlugins = append(kode.Spec.InitPlugins, kodev1alpha2.InitPluginSpec{
		Name:  "proxy-init",
		Image: initContainers[0].Image,
		Args:  initContainers[0].Args,
	})

	config.Containers = append(config.Containers, containers...)

	// Create ConfigMap for Envoy config
	envoyConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-envoy-config", kode.Name),
			Namespace: kode.Namespace,
		},
		Data: map[string]string{
			"envoy.yaml": envoyConfig,
		},
	}

	_, err = resource.CreateOrPatch(ctx, envoyConfigMap, func() error {
		return controllerutil.SetControllerReference(kode, envoyConfigMap, r.GetScheme())
	})
	if err != nil {
		return fmt.Errorf("failed to create or update Envoy ConfigMap: %w", err)
	}

	return nil
}