// internal/controller/kode_secret.go

/*
Copyright emil@jacero.se 2024.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures that the Secret exists for the Kode instance
func (r *KodeReconciler) ensureSecret(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("SecretEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring Secret")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.KodeName,
			Namespace: config.KodeNamespace,
		},
	}

	// Create or patch the secret
	err := r.ResourceManager.CreateOrPatch(ctx, secret, func() error {
		constructedSecret, err := r.constructSecretSpec(config)
		if err != nil {
			return fmt.Errorf("failed to construct Secret spec: %v", err)
		}

		secret.Data = constructedSecret.Data
		secret.ObjectMeta.Labels = constructedSecret.ObjectMeta.Labels

		return controllerutil.SetControllerReference(&config.Kode, secret, r.Scheme)
	})

	// Add secret to the KodeResourcesConfig
	config.ExistingSecret = *secret
	log.V(1).Info("Secret ensured", "secret", secret)

	if err != nil {
		return fmt.Errorf("failed to create or patch Secret: %v", err)
	}

	return nil
}

// constructSecret constructs a Secret for the Kode instance
func (r *KodeReconciler) constructSecretSpec(config *common.KodeResourcesConfig) (*corev1.Secret, error) {
	log := r.Log.WithName("SecretConstructor").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.KodeName,
			Namespace: config.KodeNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(config.Kode.Spec.User),
			"password": []byte(config.Kode.Spec.Password),
		},
	}

	maskedData := common.MaskSecretData(secret)
	log.V(1).Info("Secret constructed", "Name", secret.Name, "Data", maskedData)

	return secret, nil
}
