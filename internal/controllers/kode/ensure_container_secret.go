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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures that the Secret exists for the Kode instance
func ensureSecret(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithName("SecretEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring Secret")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.GetSecretName(),
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
	}

	if config.Credentials != nil && config.Credentials.ExistingSecret != nil {
		// ExistingSecret is specified, fetch the secret
		err := resource.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		if err != nil {
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		username, password, err := common.GetUsernameAndPasswordFromSecret(secret)
		if err != nil {
			return fmt.Errorf("failed to get username and password from Secret: %w", err)
		}

		config.Credentials.Username = username
		config.Credentials.Password = password
		log.V(1).Info("Updated config.Credentials with", "Username", username, "Password", common.MaskString(password))

		log.V(1).Info("Using existing secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))

	} else {
		if config.Credentials == nil {
			return fmt.Errorf("config.Credentials is nil")
		}
		// ExistingSecret is not specified, create or patch the secret
		_, err := resource.CreateOrPatch(ctx, secret, func() error {
			constructedSecret, err := constructSecretSpec(r, config)
			if err != nil {
				return fmt.Errorf("failed to construct Secret spec: %v", err)
			}

			// Update metadata for the secret
			secret.Data = constructedSecret.Data

			return controllerutil.SetControllerReference(kode, secret, r.GetScheme())
		})

		if err != nil {
			return fmt.Errorf("failed to create or patch Secret: %v", err)
		}
	}

	return nil
}

// constructSecret constructs a Secret for the Kode instance
func constructSecretSpec(r statemachine.ReconcilerInterface, config *common.KodeResourceConfig) (*corev1.Secret, error) {
	log := r.GetLog().WithName("SecretConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"username": []byte(config.Credentials.Username),
			"password": []byte(config.Credentials.Password),
		},
	}

	log.V(1).Info("Using constructed secret", "Name", secret.Name, "Data", common.MaskSecretData(secret))

	return secret, nil
}
