// internal/controllers/kode/resource_checker.go

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// TODO: Add inactivity check
func (r *KodeReconciler) checkResourcesReady(ctx context.Context, config *common.KodeResourcesConfig) (bool, error) {
	log := r.Log.WithName("ResourceReadyChecker").WithValues("kode", common.ObjectKeyFromConfig(config))
	ctx, cancel := common.ContextWithTimeout(ctx, 20) // 20 seconds timeout
	defer cancel()

	// Check Secret
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: config.SecretName, Namespace: config.KodeNamespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Secret not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Secret: %w", err)
	}

	// TODO: Add check update status on:
	// - Image pull backoff
	// - Image pull error
	// - Image pull timeout
	// Check StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: config.KodeName, Namespace: config.KodeNamespace}, statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("StatefulSet not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	if statefulSet.Status.ReadyReplicas != statefulSet.Status.Replicas {
		log.Info("StatefulSet not ready", "ReadyReplicas", statefulSet.Status.ReadyReplicas, "DesiredReplicas", statefulSet.Status.Replicas)
		return false, nil
	}

	// Check Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: config.KodeNamespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Service not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Service: %w", err)
	}

	// Check PersistentVolumeClaim if storage is specified
	if !config.KodeSpec.Storage.IsEmpty() {
		pvc := &corev1.PersistentVolumeClaim{}
		err = r.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: config.KodeNamespace}, pvc)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("PersistentVolumeClaim not found")
				return false, nil
			}
			return false, fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			log.Info("PersistentVolumeClaim not bound", "Phase", pvc.Status.Phase)
			return false, nil
		}
	}

	// Check if Envoy sidecar is ready (if applicable)
	if config.Templates.KodeTemplate != nil && config.Templates.KodeTemplate.EnvoyConfigRef.Name != "" {
		ready, err := r.checkEnvoySidecarReady(ctx, config)
		if err != nil {
			return false, fmt.Errorf("failed to check Envoy sidecar readiness: %w", err)
		}
		if !ready {
			log.Info("Envoy sidecar not ready")
			return false, nil
		}
	}

	log.Info("All resources are ready")
	return true, nil
}

func (r *KodeReconciler) checkEnvoySidecarReady(ctx context.Context, config *common.KodeResourcesConfig) (bool, error) {
	log := r.Log.WithName("EnvoySidecarReadyChecker").WithValues("kode", common.ObjectKeyFromConfig(config))

	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: config.KodeName + "-0", Namespace: config.KodeNamespace}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Pod not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Pod: %w", err)
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "envoy-proxy" {
			if !containerStatus.Ready {
				log.Info("Envoy sidecar container not ready")
				return false, nil
			}
			return true, nil
		}
	}

	log.Info("Envoy sidecar container not found in pod")
	return false, nil
}
