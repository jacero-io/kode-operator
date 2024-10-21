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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resourcev1"
	"github.com/jacero-io/kode-operator/internal/statemachine"
)

// validateConfiguration checks if all required resources exist and are in the desired state
func validateConfiguration(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	resource := r.GetResourceManager()

	// Check if all required resources exist and are in the desired state
	if err := validateSecret(ctx, resource, kode); err != nil {
		return err
	}

	if err := validateService(ctx, resource, kode); err != nil {
		return err
	}

	if err := validateStatefulSet(ctx, resource, kode); err != nil {
		return err
	}

	if config.KodeSpec.Storage != nil {
		if err := validatePVC(ctx, resource, kode); err != nil {
			return err
		}
	}

	return nil
}

func validateSecret(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode) error {
	secret := &corev1.Secret{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}, secret)
	if err != nil {
		return fmt.Errorf("failed to get Secret: %w", err)
	}
	// Add more specific validation if needed
	return nil
}

func validateService(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode) error {
	service := &corev1.Service{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetServiceName(), Namespace: kode.Namespace}, service)
	if err != nil {
		return fmt.Errorf("failed to get Service: %w", err)
	}
	// Add more specific validation if needed
	return nil
}

func validateStatefulSet(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode) error {
	statefulSet := &appsv1.StatefulSet{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetStatefulSetName(), Namespace: kode.Namespace}, statefulSet)
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
		return fmt.Errorf("StatefulSet not ready: %d/%d replicas available", statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas)
	}
	return nil
}

func validatePVC(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode) error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}, pvc)
	if err != nil {
		return fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}
	if pvc.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("PersistentVolumeClaim not bound: current phase is %s", pvc.Status.Phase)
	}
	return nil
}
