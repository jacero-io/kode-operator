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

// validateConfiguration performs comprehensive validation of all resources
func validateConfiguration(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithValues("kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, "phase", kode.Status.Phase)
	resource := r.GetResourceManager()

	log.V(1).Info("Starting resource validation")

	if err := validateSecret(ctx, resource, kode, config); err != nil {
		log.Error(err, "Secret validation failed")
		return fmt.Errorf("secret validation failed: %w", err)
	}

	if err := validateService(ctx, resource, kode, config); err != nil {
		log.Error(err, "Service validation failed")
		return fmt.Errorf("service validation failed: %w", err)
	}

	if err := validateStatefulSet(ctx, resource, kode, config); err != nil {
		log.Error(err, "StatefulSet validation failed")
		return fmt.Errorf("statefulset validation failed: %w", err)
	}

	if config.KodeSpec.Storage != nil {
		if err := validatePVC(ctx, resource, kode, config); err != nil {
			log.Error(err, "PVC validation failed")
			return fmt.Errorf("pvc validation failed: %w", err)
		}
	}

	log.V(1).Info("Resource validation completed successfully")
	return nil
}

func validateSecret(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}

	if err := resource.Get(ctx, namespacedName, secret); err != nil {
		return fmt.Errorf("failed to get Secret: %w", err)
	}

	// Validate required fields
	if _, exists := secret.Data["username"]; !exists {
		return fmt.Errorf("secret missing required field: username")
	}

	if kode.Spec.Credentials != nil && kode.Spec.Credentials.EnableBuiltinAuth {
		if _, exists := secret.Data["password"]; !exists {
			return fmt.Errorf("secret missing required field: password (required when EnableBuiltinAuth is true)")
		}
	}

	// Validate labels
	if !hasRequiredLabels(secret.Labels, config.CommonConfig.Labels) {
		return fmt.Errorf("secret missing required labels")
	}

	return nil
}

func validateService(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	service := &corev1.Service{}
	namespacedName := types.NamespacedName{Name: kode.GetServiceName(), Namespace: kode.Namespace}

	if err := resource.Get(ctx, namespacedName, service); err != nil {
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// Validate ports
	var hasPort bool
	for _, port := range service.Spec.Ports {
		if port.Port == int32(config.Port) {
			hasPort = true
			break
		}
	}
	if !hasPort {
		return fmt.Errorf("service missing required port: %d", config.Port)
	}

	// Validate selectors
	for key, value := range config.CommonConfig.Labels {
		if selectorValue, exists := service.Spec.Selector[key]; !exists || selectorValue != value {
			return fmt.Errorf("service missing or has incorrect selector: %s=%s", key, value)
		}
	}

	return nil
}

func validateStatefulSet(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	statefulSet := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{Name: kode.GetStatefulSetName(), Namespace: kode.Namespace}

	if err := resource.Get(ctx, namespacedName, statefulSet); err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	// Validate replicas
	if statefulSet.Spec.Replicas == nil || *statefulSet.Spec.Replicas != 1 {
		return fmt.Errorf("statefulset has incorrect replica count: expected 1, got %d",
			statefulSet.Spec.Replicas)
	}

	// Validate container configuration
	if len(statefulSet.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("statefulset has no containers configured")
	}

	mainContainer := statefulSet.Spec.Template.Spec.Containers[0]

	// Validate container image
	if mainContainer.Image != config.Template.ContainerTemplateSpec.Image {
		return fmt.Errorf("container has incorrect image: expected %s, got %s",
			config.Template.ContainerTemplateSpec.Image, mainContainer.Image)
	}

	// Validate volume mounts if storage is configured
	if config.KodeSpec.Storage != nil {
		hasStorageMount := false
		for _, mount := range mainContainer.VolumeMounts {
			if mount.Name == "kode-storage" {
				hasStorageMount = true
				break
			}
		}
		if !hasStorageMount {
			return fmt.Errorf("container missing required storage volume mount")
		}
	}

	return nil
}

func validatePVC(ctx context.Context, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	// Skip validation for existing claims
	if config.KodeSpec.Storage.ExistingVolumeClaim != nil {
		return nil
	}

	pvc := &corev1.PersistentVolumeClaim{}
	namespacedName := types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}

	if err := resource.Get(ctx, namespacedName, pvc); err != nil {
		return fmt.Errorf("failed to get PVC: %w", err)
	}

	// Validate storage class if specified
	if config.KodeSpec.Storage.StorageClassName != nil &&
		(pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != *config.KodeSpec.Storage.StorageClassName) {
		return fmt.Errorf("pvc has incorrect storage class: expected %s, got %s",
			*config.KodeSpec.Storage.StorageClassName, *pvc.Spec.StorageClassName)
	}

	// Validate access modes
	if !hasMatchingAccessModes(pvc.Spec.AccessModes, config.KodeSpec.Storage.AccessModes) {
		return fmt.Errorf("pvc has incorrect access modes")
	}

	// Validate storage size
	if config.KodeSpec.Storage.Resources != nil {
		requestedSize := config.KodeSpec.Storage.Resources.Requests.Storage()
		actualSize := pvc.Spec.Resources.Requests.Storage()
		if requestedSize.Cmp(*actualSize) != 0 {
			return fmt.Errorf("pvc has incorrect storage size: expected %s, got %s",
				requestedSize.String(), actualSize.String())
		}
	}

	return nil
}

// Helper functions
func hasRequiredLabels(actual, required map[string]string) bool {
	for key, value := range required {
		if actualValue, exists := actual[key]; !exists || actualValue != value {
			return false
		}
	}
	return true
}

func hasMatchingAccessModes(actual, required []corev1.PersistentVolumeAccessMode) bool {
	if len(actual) != len(required) {
		return false
	}
	actualMap := make(map[corev1.PersistentVolumeAccessMode]bool)
	for _, mode := range actual {
		actualMap[mode] = true
	}
	for _, mode := range required {
		if !actualMap[mode] {
			return false
		}
	}
	return true
}
