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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resourcev1"
	"github.com/jacero-io/kode-operator/internal/statemachine"
)

func applyConfiguration(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	resource := r.GetResourceManager()
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	// Order matters for dependencies
	if _, ok := changes["secret"]; ok {
		log.V(1).Info("Applying secret changes")
		if err := ensureSecret(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure Secret: %w", err)
		}
	}

	if _, ok := changes["service"]; ok {
		log.V(1).Info("Applying service changes")
		if err := ensureService(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure Service: %w", err)
		}
	}

	if _, ok := changes["statefulset"]; ok {
		log.V(1).Info("Applying statefulset changes")
		if err := ensureStatefulSet(ctx, r, resource, kode, config); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet: %w", err)
		}
	}

	if !kode.Spec.Storage.IsEmpty() {
		if _, ok := changes["pvc"]; ok {
			log.V(1).Info("Applying PVC changes")
			// TODO: Add check for if CSI resize is supported. Defaults to true for now
			resizeSupported := true
			if err := ensurePersistentVolumeClaim(ctx, r, resource, kode, config, resizeSupported); err != nil {
				return fmt.Errorf("failed to ensure PersistentVolumeClaim: %w", err)
			}
		}
	}

	return nil
}

func detectChanges(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (map[string]interface{}, error) {
	log := r.GetLog().WithValues(
		"kode", client.ObjectKeyFromObject(kode),
		"function", "detectChanges",
		"phase", kode.Status.Phase,
	)
	log.V(1).Info("Starting change detection")

	resource := r.GetResourceManager()
	changes := make(map[string]interface{})

	// Detect changes for each resource type
	for _, check := range []struct {
		name string
		fn   func() error
	}{
		{"secret", func() error { return detectSecretChanges(ctx, r, resource, kode, config, changes) }},
		{"service", func() error { return detectServiceChanges(ctx, r, resource, kode, config, changes) }},
		{"statefulset", func() error { return detectStatefulSetChanges(ctx, r, resource, kode, config, changes) }},
	} {
		log.V(1).Info("Checking resource for changes", "resource", check.name)
		if err := check.fn(); err != nil {
			log.Error(err, "Failed to detect changes", "resource", check.name)
			return nil, err
		}
	}

	// Check PVC changes if storage is configured
	if !kode.Spec.Storage.IsEmpty() {
		log.V(1).Info("Checking PVC for changes")
		if err := detectPVCChanges(ctx, r, resource, kode, config, changes); err != nil {
			log.Error(err, "Failed to detect PVC changes")
			return nil, err
		}
	}

	log.V(1).Info("Change detection completed", "changes", changes)
	return changes, nil
}

func detectSecretChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	log := r.GetLog().WithValues(
		"kode", client.ObjectKeyFromObject(kode),
		"function", "detectSecretChanges",
		"phase", kode.Status.Phase,
	)

	existing := &corev1.Secret{}
	namespacedName := types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}

	err := resource.Get(ctx, namespacedName, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["secret"] = true
			return nil
		}
		return err
	}

	desiredData := map[string][]byte{
		"username": []byte(config.Credentials.Username),
		"password": []byte(config.Credentials.Password),
	}

	// Compare and log differences
	if !reflect.DeepEqual(existing.Data, desiredData) {
		changes["secret"] = true
		// Safely log differences without exposing sensitive data
		if _, exists := existing.Data["username"]; !exists {
			log.V(1).Info("Username field missing in existing secret")
		}
		if _, exists := existing.Data["password"]; !exists {
			log.V(1).Info("Password field missing in existing secret")
		}
	}

	return nil
}

func detectServiceChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	log := r.GetLog().WithValues(
		"kode", client.ObjectKeyFromObject(kode),
		"function", "detectServiceChanges",
		"phase", kode.Status.Phase,
	)

	existing := &corev1.Service{}
	namespacedName := types.NamespacedName{Name: kode.GetServiceName(), Namespace: kode.Namespace}

	err := resource.Get(ctx, namespacedName, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Service not found, marking for creation")
			changes["service"] = true
			return nil
		}
		return err
	}

	desiredService, err := constructServiceSpec(r, config)
	if err != nil {
		return err
	}

	// Compare and log specific differences
	if !reflect.DeepEqual(existing.Spec.Ports, desiredService.Spec.Ports) {
		log.V(1).Info("Service ports changed",
			"existing", existing.Spec.Ports,
			"desired", desiredService.Spec.Ports)
		changes["service"] = true
	}

	if !reflect.DeepEqual(existing.Spec.Selector, desiredService.Spec.Selector) {
		log.V(1).Info("Service selector changed",
			"existing", existing.Spec.Selector,
			"desired", desiredService.Spec.Selector)
		changes["service"] = true
	}

	return nil
}

func detectStatefulSetChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	existing := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{Name: kode.GetStatefulSetName(), Namespace: kode.Namespace}

	err := resource.Get(ctx, namespacedName, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			changes["statefulset"] = true
			return nil
		}
		return err
	}

	desiredStatefulSet, err := constructStatefulSetSpec(ctx, r, resource, kode, config)
	if err != nil {
		return err
	}

	// Check for container image differences
	for i, existingContainer := range existing.Spec.Template.Spec.Containers {
		if i >= len(desiredStatefulSet.Spec.Template.Spec.Containers) {
			continue
		}
		desiredContainer := desiredStatefulSet.Spec.Template.Spec.Containers[i]

		if existingContainer.Image != desiredContainer.Image {
			log.Info("Container image mismatch detected",
				"container", existingContainer.Name,
				"currentImage", existingContainer.Image,
				"desiredImage", desiredContainer.Image)
			changes["statefulset"] = true
			break
		}
	}

	// Check pod template hash
	existingHash := existing.Spec.Template.Annotations["kode.jacero.io/pod-template-hash"]
	desiredHash := desiredStatefulSet.Spec.Template.Annotations["kode.jacero.io/pod-template-hash"]
	if existingHash != desiredHash {
		log.Info("Pod template hash mismatch",
			"currentHash", existingHash,
			"desiredHash", desiredHash)
		changes["statefulset"] = true
	}

	// Check actual pod images
	podList := &corev1.PodList{}
	if err := resource.List(ctx, podList,
		client.InNamespace(kode.Namespace),
		client.MatchingLabels(config.CommonConfig.Labels)); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		for i, container := range pod.Spec.Containers {
			if i >= len(desiredStatefulSet.Spec.Template.Spec.Containers) {
				continue
			}
			desiredContainer := desiredStatefulSet.Spec.Template.Spec.Containers[i]

			if container.Image != desiredContainer.Image {
				log.Info("Pod container image mismatch detected",
					"pod", pod.Name,
					"container", container.Name,
					"currentImage", container.Image,
					"desiredImage", desiredContainer.Image)
				changes["statefulset"] = true
				// Force pod deletion to ensure update
				if err := resource.Delete(ctx, &pod); err != nil {
					log.Error(err, "Failed to delete pod with mismatched image")
				}
				break
			}
		}
	}

	return nil
}

func detectPVCChanges(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, changes map[string]interface{}) error {
	log := r.GetLog().WithValues(
		"kode", client.ObjectKeyFromObject(kode),
		"function", "detectPVCChanges",
		"phase", kode.Status.Phase,
	)

	// Skip if using existing claim
	if config.KodeSpec.Storage.ExistingVolumeClaim != nil {
		log.V(1).Info("Using existing PVC, skipping change detection",
			"existingClaim", *config.KodeSpec.Storage.ExistingVolumeClaim)
		return nil
	}

	existing := &corev1.PersistentVolumeClaim{}
	namespacedName := types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}

	err := resource.Get(ctx, namespacedName, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("PVC not found, marking for creation")
			changes["pvc"] = true
			return nil
		}
		return err
	}

	desiredPVC, err := constructPVCSpec(r, config)
	if err != nil {
		log.Error(err, "Failed to construct desired PVC spec")
		return err
	}

	// Compare storage size
	if !existing.Spec.Resources.Requests.Storage().Equal(*desiredPVC.Spec.Resources.Requests.Storage()) {
		log.V(1).Info("PVC storage size changed",
			"existing", existing.Spec.Resources.Requests.Storage(),
			"desired", desiredPVC.Spec.Resources.Requests.Storage())
		changes["pvc"] = true
	}

	// Compare access modes
	if !reflect.DeepEqual(existing.Spec.AccessModes, desiredPVC.Spec.AccessModes) {
		log.V(1).Info("PVC access modes changed",
			"existing", existing.Spec.AccessModes,
			"desired", desiredPVC.Spec.AccessModes)
		changes["pvc"] = true
	}

	// Compare storage class if specified
	if desiredPVC.Spec.StorageClassName != nil {
		if existing.Spec.StorageClassName == nil || *existing.Spec.StorageClassName != *desiredPVC.Spec.StorageClassName {
			log.V(1).Info("PVC storage class changed",
				"existing", existing.Spec.StorageClassName,
				"desired", desiredPVC.Spec.StorageClassName)
			changes["pvc"] = true
		}
	}

	return nil
}
