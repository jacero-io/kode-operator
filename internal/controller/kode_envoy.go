// internal/controller/kode_envoy.go

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

package controller

import (
	"context"
	"fmt"

	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/envoy"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// ensureEnvoyContainer ensures that the Envoy container exists for the Kode instance
func (r *KodeReconciler) ensureEnvoy(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("EnvoyEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring Envoy Container")

	configGenerator := envoy.NewBootstrapConfigGenerator(r.Log.WithName("EnvoyConfigGenerator").WithValues("kode", client.ObjectKeyFromObject(&config.Kode)))
	evnoyContainers, envoyInitContainers, err := envoy.NewContainerConstructor(
		r.Log.WithName("EnvoyContainerConstructor").WithValues("kode", client.ObjectKeyFromObject(&config.Kode)),
		configGenerator).ConstructEnvoyContainers(config)
	if err != nil {
		return fmt.Errorf("failed to construct Envoy sidecar: %v", err)
	}
	config.Containers = append(config.Containers, evnoyContainers...)
	config.InitContainers = append(config.InitContainers, envoyInitContainers...)

	return nil
}
