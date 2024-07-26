// internal/controllers/kode/ensure.go

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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *KodeReconciler) ensureResources(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithValues("kode", types.NamespacedName{Name: config.KodeName, Namespace: config.KodeNamespace})

	// Update status to Creating before starting resource creation
	// if err := r.updateKodePhaseCreating(ctx, config); err != nil {
	// 	log.Error(err, "Failed to update status to Creating")
	// 	return err
	// }

	// Fetch the latest Kode object
	kode, err := r.getLatestKode(ctx, config.KodeName, config.KodeNamespace)
	if err != nil {
		return fmt.Errorf("failed to get Kode: %w", err)
	}

	// now := metav1.NewTime(time.Now())

	// Ensure Secret
	if err := r.ensureSecret(ctx, config, kode); err != nil {
		log.Error(err, "Failed to ensure Secret")
		// r.updateKodePhaseFailed(ctx, config, err, metav1.Condition{
		// 	Type:               "SecretCreationFailed",
		// 	Status:             metav1.ConditionTrue,
		// 	Reason:             "SecretCreationError",
		// 	Message:            fmt.Sprintf("Failed to create Secret: %s", err.Error()),
		// 	LastTransitionTime: now,
		// })
		return err
	}

	// Ensure Credentials
	if err := r.ensureCredentials(ctx, config, kode); err != nil {
		log.Error(err, "Failed to ensure Credentials")
		// r.updateKodePhaseFailed(ctx, config, err, metav1.Condition{
		// 	Type:               "CredentialsCreationFailed",
		// 	Status:             metav1.ConditionTrue,
		// 	Reason:             "CredentialsCreationError",
		// 	Message:            fmt.Sprintf("Failed to create Credentials: %s", err.Error()),
		// 	LastTransitionTime: now,
		// })
		return err
	}

	// If the KodeTemplate has an EnvoyConfigRef, ensure the EnvoyContainer
	if config.Templates.EnvoyProxyConfig != nil {
		if err := r.ensureSidecarContainers(ctx, config); err != nil {
			return err
		}
	}

	// Ensure StatefulSet
	if err := r.ensureStatefulSet(ctx, config, kode); err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		// r.updateKodePhaseFailed(ctx, config, err, metav1.Condition{
		// 	Type:               "StatefulSetCreationFailed",
		// 	Status:             metav1.ConditionTrue,
		// 	Reason:             "StatefulSetCreationError",
		// 	Message:            fmt.Sprintf("Failed to create StatefulSet: %s", err.Error()),
		// 	LastTransitionTime: now,
		// })
		return err
	}

	// Ensure Service
	if err := r.ensureService(ctx, config, kode); err != nil {
		log.Error(err, "Failed to ensure Service")
		// r.updateKodePhaseFailed(ctx, config, err, metav1.Condition{
		// 	Type:               "ServiceCreationFailed",
		// 	Status:             metav1.ConditionTrue,
		// 	Reason:             "ServiceCreationError",
		// 	Message:            fmt.Sprintf("Failed to create Service: %s", err.Error()),
		// 	LastTransitionTime: now,
		// })
		return err
	}

	// Ensure PVC if storage is specified
	if !kode.Spec.Storage.IsEmpty() {
		if err := r.ensurePVC(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure PVC")
			// r.updateKodePhaseFailed(ctx, config, err, metav1.Condition{
			// 	Type:               "PVCCreationFailed",
			// 	Status:             metav1.ConditionTrue,
			// 	Reason:             "PVCCreationError",
			// 	Message:            fmt.Sprintf("Failed to create PersistentVolumeClaim: %s", err.Error()),
			// 	LastTransitionTime: now,
			// })
			return err
		}
	}

	// Update status to Created after all resources are ensured
	// if err := r.updateKodePhaseCreated(ctx, config); err != nil {
	// 	log.Error(err, "Failed to update status to Created")
	// 	return err
	// }

	log.Info("All resources ensured successfully")
	return nil
}

func (r *KodeReconciler) ensureCredentials(ctx context.Context, config *common.KodeResourcesConfig, kode *kodev1alpha1.Kode) error {
	log := r.Log.WithName("CredentialsEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config))

	if config.KodeSpec.ExistingSecret != "" {
		// ExistingSecret is specified, fetch the secret
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: config.KodeSpec.ExistingSecret, Namespace: config.KodeNamespace}, secret)
		if err != nil {
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		username, password, err := common.GetUsernameAndPasswordFromSecret(secret)
		if err != nil {
			return fmt.Errorf("failed to get username and password from Secret: %w", err)
		}

		config.Credentials.Username = username
		config.Credentials.Password = password

		log.Info("Using existing secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))
	} else if config.KodeSpec.Password != "" {
		config.Credentials.Username = config.KodeSpec.Username
		config.Credentials.Password = config.KodeSpec.Password
	} else {
		config.Credentials.Username = config.KodeSpec.Username
		config.Credentials.Password = ""
	}

	log.Info("Credentials ensured successfully")
	return nil
}
