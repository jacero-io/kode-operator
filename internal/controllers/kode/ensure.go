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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *KodeReconciler) ensureResources(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("ResoruceEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config))

	// Fetch the latest Kode object
	kode, err := common.GetLatestKode(ctx, r.Client, config.KodeName, config.KodeNamespace)
	if err != nil {
		return fmt.Errorf("failed to get Kode: %w", err)
	}

	// Check if the resource has changed since last reconciliation
	if kode.Generation != kode.Status.ObservedGeneration {
		log.Info("Resource has changed, ensuring all resources")

		// Update status to Creating before starting resource creation
		if kode.Status.Phase != kodev1alpha1.KodePhasePending {
			if kode.Status.Phase == kodev1alpha1.KodePhaseActive {
				if err := r.updateKodePhasePending(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else {
				if err := r.updateKodePhaseCreating(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Creating")
					return err
				}
			}
		}

		// Ensure Secret
		if err := r.ensureSecret(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure Secret")
			r.updateKodePhaseFailed(ctx, kode, err, metav1.Condition{
				Type:               "SecretCreationFailed",
				Status:             metav1.ConditionTrue,
				Reason:             "SecretCreationError",
				Message:            fmt.Sprintf("Failed to create Secret: %s", err.Error()),
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		// Ensure Credentials
		if err := r.ensureCredentials(ctx, config); err != nil {
			log.Error(err, "Failed to ensure Credentials")
			r.updateKodePhaseFailed(ctx, kode, err, metav1.Condition{
				Type:               "CredentialsCreationFailed",
				Status:             metav1.ConditionTrue,
				Reason:             "CredentialsCreationError",
				Message:            fmt.Sprintf("Failed to create Credentials: %s", err.Error()),
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		// If the KodeTemplate has an EnvoyConfigRef, ensure the EnvoyContainer
		if config.Templates.EnvoyProxyConfig != nil {
			if err := r.ensureSidecarContainers(ctx, config, kode); err != nil {
				log.Error(err, "Failed to ensure sidecar containers")
				return err
			}
		}

		// Ensure StatefulSet
		if err := r.ensureStatefulSet(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure StatefulSet")
			r.updateKodePhaseFailed(ctx, kode, err, metav1.Condition{
				Type:               "StatefulSetCreationFailed",
				Status:             metav1.ConditionTrue,
				Reason:             "StatefulSetCreationError",
				Message:            fmt.Sprintf("Failed to create StatefulSet: %s", err.Error()),
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		// Ensure Service
		if err := r.ensureService(ctx, config, kode); err != nil {
			log.Error(err, "Failed to ensure Service")
			r.updateKodePhaseFailed(ctx, kode, err, metav1.Condition{
				Type:               "ServiceCreationFailed",
				Status:             metav1.ConditionTrue,
				Reason:             "ServiceCreationError",
				Message:            fmt.Sprintf("Failed to create Service: %s", err.Error()),
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		// Ensure PVC if storage is specified
		if !kode.Spec.Storage.IsEmpty() {
			if err := r.ensurePVC(ctx, config, kode); err != nil {
				log.Error(err, "Failed to ensure PVC")
				r.updateKodePhaseFailed(ctx, kode, err, metav1.Condition{
					Type:               "PVCCreationFailed",
					Status:             metav1.ConditionTrue,
					Reason:             "PVCCreationError",
					Message:            fmt.Sprintf("Failed to create PersistentVolumeClaim: %s", err.Error()),
					LastTransitionTime: metav1.Now(),
				})
				return err
			}
		}

		// Update status to Created after all resources are ensured
		if kode.Status.Phase != kodev1alpha1.KodePhasePending {
			if kode.Status.Phase == kodev1alpha1.KodePhaseActive {
				if err := r.updateKodePhasePending(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Pending")
					return err
				}
			} else {
				if err := r.updateKodePhaseCreated(ctx, kode); err != nil {
					log.Error(err, "Failed to update status to Created")
					return err
				}
			}
		}
	}
	// else {
	// 	log.Info("Resource hasn't changed, checking readiness")
	// }

	// // Check if all resources are ready
	// ready, err := r.checkResourcesReady(ctx, config)
	// if err != nil {
	// 	log.Error(err, "Failed to check resource readiness")
	// 	return err
	// }

	// if !ready {
	// 	log.V(1).Info("Resources not ready, updating status to Pending")
	// 	if err := r.updateKodePhasePending(ctx, kode); err != nil {
	// 		log.Error(err, "Failed to update status to Pending")
	// 		return err
	// 	}
	// } else {
	// 	log.V(1).Info("All resources ready, updating status to Active")
	// 	if err := r.updateKodePhaseActive(ctx, kode); err != nil {
	// 		log.Error(err, "Failed to update status to Active")
	// 		return err
	// 	}
	// }

	return nil
}

func (r *KodeReconciler) ensureCredentials(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("CredentialsEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config))

	if config.KodeSpec.ExistingSecret != "" {
		// ExistingSecret is specified, fetch the secret
		secret := &corev1.Secret{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: config.KodeSpec.ExistingSecret, Namespace: config.KodeNamespace}, secret)
		if err != nil {
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		username, password, err := common.GetUsernameAndPasswordFromSecret(secret)
		if err != nil {
			return fmt.Errorf("failed to get username and password from Secret: %w", err)
		}

		config.Credentials.Username = username
		config.Credentials.Password = password

		log.V(1).Info("Using existing secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))
	} else if config.KodeSpec.Password != "" {
		config.Credentials.Username = config.KodeSpec.Username
		config.Credentials.Password = config.KodeSpec.Password
	} else {
		config.Credentials.Username = config.KodeSpec.Username
		config.Credentials.Password = ""
	}

	log.V(1).Info("Credentials ensured successfully")
	return nil
}
