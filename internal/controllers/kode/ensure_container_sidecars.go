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

	"github.com/go-logr/logr"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
)

// ensureSidecarContainers ensures that the Envoy container exists for the Kode instance
func ensureSidecarContainers(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithName("SidecarContainerEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	log.V(1).Info("Ensuring sidecar containers")

	return nil
}

func ensureEnvoySidecar(config *common.KodeResourceConfig, log logr.Logger) error {

	return nil
}
