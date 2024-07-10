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

	"github.com/jacero-io/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KodeReconciler) ensureResources(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("ResourceEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	// Update status to Creating before starting resource creation
	if err := r.updateKodePhaseCreating(ctx, config); err != nil {
		log.Error(err, "Failed to update status to Creating")
		return err
	}

	// Ensure Secret
	if err := r.ensureSecret(ctx, config); err != nil {
		log.Error(err, "Failed to ensure Secret")
		return err
	}

	// Ensure Credentials
	if err := r.ensureCredentials(ctx, config); err != nil {
		log.Error(err, "Failed to ensure Credentials")
		return err
	}

	// If the KodeTemplate has an EnvoyConfigRef, ensure the EnvoyContainer
	if config.Templates.EnvoyProxyConfig != nil {
		if err := r.ensureSidecarContainers(config); err != nil {
			log.Error(err, "Failed to ensure EnvoyContainer")
			return err
		}
	}

	// Ensure StatefulSet
	if err := r.ensureStatefulSet(ctx, config); err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		return r.updateKodePhaseFailed(ctx, config, err)
	}

	// Ensure Service
	if err := r.ensureService(ctx, config); err != nil {
		log.Error(err, "Failed to ensure Service")
		return r.updateKodePhaseFailed(ctx, config, err)
	}

	// Ensure PVC if storage is specified
	if !config.Kode.Spec.Storage.IsEmpty() {
		if err := r.ensurePVC(ctx, config); err != nil {
			log.Error(err, "Failed to ensure PVC")
			return r.updateKodePhaseFailed(ctx, config, err)
		}
	}

	// Update status to Created after all resources are ensured
	if err := r.updateKodePhaseCreated(ctx, config); err != nil {
		log.Error(err, "Failed to update status to Created")
		return err
	}

	log.Info("All resources ensured successfully")
	return nil
}

func (r *KodeReconciler) ensureCredentials(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("CredentialsEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	if config.Kode.Spec.ExistingSecret != "" {
		// ExistingSecret is specified, fetch the secret
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: config.Kode.Spec.ExistingSecret, Namespace: config.KodeNamespace}, secret)
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
	} else if config.Kode.Spec.Password != "" {
		config.Credentials.Username = config.Kode.Spec.Username
		config.Credentials.Password = config.Kode.Spec.Password
	} else {
		config.Credentials.Username = config.Kode.Spec.Username
		config.Credentials.Password = ""
	}

	log.Info("Credentials ensured successfully")
	return nil
}
