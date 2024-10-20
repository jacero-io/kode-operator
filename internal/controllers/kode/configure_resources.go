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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
)

func detectChanges(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (map[string]interface{}, error) {
	resource := r.GetResourceManager()

	changes := make(map[string]interface{})

	// For updates, compare with existing resources
	if err := detectSecretChanges(ctx, resource, kode, config, changes); err != nil {
		return nil, err
	}
	if err := detectServiceChanges(ctx, r, resource, kode, config, changes); err != nil {
		return nil, err
	}
	if err := detectStatefulSetChanges(ctx, r, resource, kode, config, changes); err != nil {
		return nil, err
	}
	if config.KodeSpec.Storage != nil {
		if err := detectPVCChanges(ctx, r, resource, kode, config, changes); err != nil {
			return nil, err
		}
	}

	return changes, nil
}

func applyConfiguration(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	resource := r.GetResourceManager()

	if _, ok := changes["secret"]; ok {
		if err := ensureSecret(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure Secret: %w", err)
		}
	}

	if _, ok := changes["service"]; ok {
		if err := ensureService(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure Service: %w", err)
		}
	}

	if _, ok := changes["statefulset"]; ok {
		if err := ensureStatefulSet(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet: %w", err)
		}
	}

	if _, ok := changes["pvc"]; ok {
		// TODO: Add check for if CSI resize is supported. Defaults to true for now
		resizeSupported := true
		if err := ensurePersistentVolumeClaim(ctx, r, resource, kode, config, resizeSupported); err != nil {
			return fmt.Errorf("failed to ensure PersistentVolumeClaim: %w", err)
		}
	}

	return nil
}

func detectSecretChanges(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	existing := &corev1.Secret{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["secret"] = true
			return nil
		}
		return err
	}

	// Compare existing secret with desired configuration
	if !reflect.DeepEqual(existing.Data, map[string][]byte{
		"username": []byte(config.Credentials.Username),
		"password": []byte(config.Credentials.Password),
	}) {
		changes["secret"] = true
	}

	return nil
}

func detectServiceChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	existing := &corev1.Service{}
	err := resource.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: kode.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["service"] = true
			return nil
		}
		return err
	}

	// Compare existing service with desired configuration
	desiredService, err := constructServiceSpec(r, config)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existing.Spec.Ports, desiredService.Spec.Ports) ||
		!reflect.DeepEqual(existing.Spec.Selector, desiredService.Spec.Selector) {
		changes["service"] = true
	}

	return nil
}

func detectStatefulSetChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	existing := &appsv1.StatefulSet{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetStatefulSetName(), Namespace: kode.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["statefulset"] = true
			return nil
		}
		return err
	}

	// Compare existing statefulset with desired configuration
	desiredStatefulSet, err := constructStatefulSetSpec(r, kode, config)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existing.Spec.Template.Spec.Containers, desiredStatefulSet.Spec.Template.Spec.Containers) ||
		!reflect.DeepEqual(existing.Spec.VolumeClaimTemplates, desiredStatefulSet.Spec.VolumeClaimTemplates) {
		changes["statefulset"] = true
	}

	return nil
}

func detectPVCChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	existing := &corev1.PersistentVolumeClaim{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["pvc"] = true
			return nil
		}
		return err
	}

	// Compare existing PVC with desired configuration
	desiredPVC, err := constructPVCSpec(r, config)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existing.Spec.Resources, desiredPVC.Spec.Resources) ||
		!reflect.DeepEqual(existing.Spec.AccessModes, desiredPVC.Spec.AccessModes) {
		changes["pvc"] = true
	}

	return nil
}
