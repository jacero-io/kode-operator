// internal/controllers/kode/ensure_envoy.go

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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	// "github.com/jacero-io/kode-operator/internal/envoy"
)

// ensureSidecarContainers ensures that the Envoy container exists for the Kode instance
func (r *KodeReconciler) ensureSidecarContainers(ctx context.Context, config *common.KodeResourcesConfig, kode *kodev1alpha1.Kode) error {
	log := r.Log.WithName("SidecarContainerEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config))

	log.V(1).Info("Ensuring sidecar containers")

	// if err := ensureEnvoySidecar(config, log); err != nil {
	// 	log.Error(err, "Failed to ensure Envoy sidecar container")

	// 	if envoy.IsEnvoyError(err) {
	// 		now := metav1.NewTime(time.Now())
	// 		var condition metav1.Condition

	// 		switch envoy.GetEnvoyErrorType(err) {
	// 		case envoy.EnvoyErrorTypeConfiguration:
	// 			condition = metav1.Condition{
	// 				Type:               string(common.ConditionTypeConfigured),
	// 				Status:             metav1.ConditionFalse,
	// 				Reason:             "EnvoyConfigurationFailed",
	// 				Message:            err.Error(),
	// 				LastTransitionTime: now,
	// 			}
	// 		case envoy.EnvoyErrorTypeCreation:
	// 			condition = metav1.Condition{
	// 				Type:               "EnvoyContainerCreationFailed",
	// 				Status:             metav1.ConditionTrue,
	// 				Reason:             "EnvoyContainerCreationError",
	// 				Message:            err.Error(),
	// 				LastTransitionTime: now,
	// 			}
	// 		default:
	// 			condition = metav1.Condition{
	// 				Type:               "EnvoyError",
	// 				Status:             metav1.ConditionTrue,
	// 				Reason:             "UnknownEnvoyError",
	// 				Message:            err.Error(),
	// 				LastTransitionTime: now,
	// 			}
	// 		}
	// 		return r.updateStatus(ctx, kode, kodev1alpha1.KodePhaseFailed, []metav1.Condition{condition}, err)
	// 	}

	// 	// Handle other sidecar container errors
	// 	return r.updateStatus(ctx, kode, kodev1alpha1.KodePhaseFailed, []metav1.Condition{{
	// 		Type:               "SidecarContainerCreationFailed",
	// 		Status:             metav1.ConditionTrue,
	// 		Reason:             "SidecarContainerCreationError",
	// 		Message:            fmt.Sprintf("Failed to create sidecar container: %s", err.Error()),
	// 		LastTransitionTime: metav1.NewTime(time.Now()),
	// 	}}, err)
	// }

	return nil
}

// func ensureEnvoySidecar(config *common.KodeResourcesConfig, log logr.Logger) error {
// 	configGenerator := envoy.NewBootstrapConfigGenerator(log.WithName("EnvoyConfigGenerator").WithValues("kode", common.ObjectKeyFromConfig(config)))
// 	envoyContainers, envoyInitContainers, err := envoy.NewContainerConstructor(
// 		log.WithName("EnvoyContainerConstructor").WithValues("kode", common.ObjectKeyFromConfig(config)),
// 		configGenerator).ConstructEnvoyContainers(config)
// 	if err != nil {
// 		if strings.Contains(err.Error(), "configuration") {
// 			return envoy.NewEnvoyError(envoy.EnvoyErrorTypeConfiguration, "Failed to configure Envoy", err)
// 		}
// 		return envoy.NewEnvoyError(envoy.EnvoyErrorTypeCreation, "Failed to create Envoy container", err)
// 	}
// 	config.Containers = append(config.Containers, envoyContainers...)
// 	config.InitContainers = append(config.InitContainers, envoyInitContainers...)

// 	return nil
// }
