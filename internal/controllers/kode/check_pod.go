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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KodeReconciler) checkPodResources(ctx context.Context, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	log := r.Log.WithValues("kode", types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace})
	log.V(1).Info("Checking if resources are ready")

	// Check Secret
	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: config.SecretName, Namespace: kode.Namespace}, secret); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Secret not found, resources not ready")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Secret: %w", err)
	}
	log.V(1).Info("Secret found")

	// Check Service
	service := &corev1.Service{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: kode.Namespace}, service); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Service not found, resources not ready")
			return false, nil
		}
		return false, fmt.Errorf("failed to get Service: %w", err)
	}
	log.V(1).Info("Service found")

	// Check StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: config.StatefulSetName, Namespace: kode.Namespace}, statefulSet); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("StatefulSet not found, resources not ready")
			return false, nil
		}
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	// Check if StatefulSet is ready
	if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
		log.V(1).Info("StatefulSet not ready", "ReadyReplicas", statefulSet.Status.ReadyReplicas, "DesiredReplicas", *statefulSet.Spec.Replicas)
		return false, nil
	}
	log.V(1).Info("StatefulSet ready")

	// Check PVC if storage is specified
	if !config.KodeSpec.Storage.IsEmpty() {
		pvc := &corev1.PersistentVolumeClaim{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: kode.Namespace}, pvc); err != nil {
			if errors.IsNotFound(err) {
				log.V(1).Info("PVC not found, resources not ready")
				return false, nil
			}
			return false, fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			log.V(1).Info("PVC not bound", "Phase", pvc.Status.Phase)
			return false, nil
		}
		log.V(1).Info("PVC ready")
	}

	// Check if pods are ready
	podList := &corev1.PodList{}
	if err := r.Client.List(ctx, podList, client.InNamespace(kode.Namespace), client.MatchingLabels(config.CommonConfig.Labels)); err != nil {
		return false, fmt.Errorf("failed to list Pods: %w", err)
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			log.V(1).Info("Pod not running", "PodName", pod.Name, "Phase", pod.Status.Phase)
			return false, nil
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
				log.V(1).Info("Pod not ready", "PodName", pod.Name, "ReadyStatus", condition.Status)
				return false, nil
			}
		}
	}
	log.V(1).Info("All pods are ready")

	log.V(1).Info("All resources are ready")
	return true, nil
}
