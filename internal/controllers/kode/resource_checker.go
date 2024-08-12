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

package kode

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
func (r *KodeReconciler) checkResourcesReady(ctx context.Context, config *common.KodeResourceConfig) (bool, error) {
	log := r.Log.WithName("ResourceReadyChecker").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))
	ctx, cancel := common.ContextWithTimeout(ctx, 20) // 20 seconds timeout
	defer cancel()

	// Check Secret
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: config.SecretName, Namespace: config.CommonConfig.Namespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Secret not found")
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
	err = r.Client.Get(ctx, types.NamespacedName{Name: config.CommonConfig.Name, Namespace: config.CommonConfig.Namespace}, statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("StatefulSet not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	if statefulSet.Status.ReadyReplicas != statefulSet.Status.Replicas {
		log.V(1).Info("StatefulSet not ready", "ReadyReplicas", statefulSet.Status.ReadyReplicas, "DesiredReplicas", statefulSet.Status.Replicas)
		return false, nil
	}

	// Check Service
	service := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: config.CommonConfig.Namespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Service not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Service: %w", err)
	}

	// Check PersistentVolumeClaim if storage is specified
	if !config.KodeSpec.Storage.IsEmpty() {
		pvc := &corev1.PersistentVolumeClaim{}
		err = r.Client.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: config.CommonConfig.Namespace}, pvc)
		if err != nil {
			if errors.IsNotFound(err) {
				log.V(1).Info("PersistentVolumeClaim not found")
				return false, nil
			}
			return false, fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			log.V(1).Info("PersistentVolumeClaim not bound", "Phase", pvc.Status.Phase, "Spec", pvc.Spec, "Status", pvc.Status)
			return false, nil
		}
	}

	// Check if Envoy sidecar is ready (if applicable)
	// if config.Templates.KodeTemplate != nil && config.Templates.KodeTemplate.EnvoyConfigRef.Name != "" {
	// 	ready, err := r.checkEnvoySidecarReady(ctx, config)
	// 	if err != nil {
	// 		return false, fmt.Errorf("failed to check Envoy sidecar readiness: %w", err)
	// 	}
	// 	if !ready {
	// 		log.V(1).Info("Envoy sidecar not ready")
	// 		return false, nil
	// 	}
	// }

	log.V(1).Info("All resources are ready")
	return true, nil
}

// func (r *KodeReconciler) checkEnvoySidecarReady(ctx context.Context, config *common.KodeResourceConfig) (bool, error) {
// 	log := r.Log.WithName("EnvoySidecarReadyChecker").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

// 	pod := &corev1.Pod{}
// 	err := r.Client.Get(ctx, types.NamespacedName{Name: config.KodeName + "-0", Namespace: config.KodeNamespace}, pod)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			log.V(1).Info("Pod not found")
// 			return false, nil
// 		}
// 		return false, fmt.Errorf("failed to get Pod: %w", err)
// 	}

// 	for _, containerStatus := range pod.Status.ContainerStatuses {
// 		if containerStatus.Name == "envoy-proxy" {
// 			if !containerStatus.Ready {
// 				log.V(1).Info("Envoy sidecar container not ready")
// 				return false, nil
// 			}
// 			return true, nil
// 		}
// 	}

// 	log.V(1).Info("Envoy sidecar container not found in pod")
// 	return false, nil
// }
