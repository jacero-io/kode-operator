// internal/controllers/kode/ensure_secret.go

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
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures that the Secret exists for the Kode instance
func (r *KodeReconciler) ensureSecret(ctx context.Context, config *common.KodeResourcesConfig, kode *kodev1alpha1.Kode) error {
	log := r.Log.WithName("SecretEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring Secret")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SecretName,
			Namespace: config.KodeNamespace,
		},
	}

	if config.KodeSpec.ExistingSecret != "" {
		// ExistingSecret is specified, fetch the secret
		err := r.ResourceManager.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		if err != nil {
			return fmt.Errorf("failed to get Secret: %v", err)
		}

		log.V(1).Info("Using existing secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))

		config.Secret = *secret

	} else {
		// ExistingSecret is not specified, create or patch the secret
		err := r.ResourceManager.CreateOrPatch(ctx, secret, func() error {
			constructedSecret, err := r.constructSecretSpec(config)
			if err != nil {
				return fmt.Errorf("failed to construct Secret spec: %v", err)
			}

			// Update metadata for the secret
			secret.Data = constructedSecret.Data
			secret.ObjectMeta.Labels = constructedSecret.ObjectMeta.Labels
			secret.ObjectMeta.Annotations = constructedSecret.ObjectMeta.Annotations

			return controllerutil.SetControllerReference(kode, secret, r.Scheme)
		})

		if err != nil {
			return fmt.Errorf("failed to create or patch Secret: %v", err)
		}

		config.Secret = *secret
	}

	return nil
}

// constructSecret constructs a Secret for the Kode instance
func (r *KodeReconciler) constructSecretSpec(config *common.KodeResourcesConfig) (*corev1.Secret, error) {
	log := r.Log.WithName("SecretConstructor").WithValues("kode", common.ObjectKeyFromConfig(config))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SecretName,
			Namespace: config.KodeNamespace,
			Labels:    config.Labels,
		},
		Data: map[string][]byte{
			"username": []byte(config.Credentials.Username),
			"password": []byte(config.Credentials.Password),
		},
	}

	log.V(1).Info("Using constructed secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))

	return secret, nil
}
