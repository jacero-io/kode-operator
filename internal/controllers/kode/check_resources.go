package kode

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
)

func checkResourcesReady(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))
	log.V(1).Info("Checking if resources are ready")

	resource := r.GetResourceManager()

	// Check Secret
	if err := checkSecretReady(ctx, resource, kode, config); err != nil {
		return false, err
	}

	// Check Service
	if err := checkServiceReady(ctx, resource, kode, config); err != nil {
		return false, err
	}

	// Check StatefulSet
	ready, err := checkStatefulSetReady(ctx, resource, kode, config)
	if err != nil {
		return false, err
	}
	if !ready {
		log.V(1).Info("StatefulSet not ready")
		return false, nil
	}

	// Check PVC if storage is specified
	if !config.KodeSpec.Storage.IsEmpty() {
		if err := checkPVCReady(ctx, resource, kode, config); err != nil {
			return false, err
		}
	}

	// Check if pods are ready
	podsReady, err := checkPodsReady(ctx, resource, kode, config)
	if err != nil {
		return false, err
	}
	if !podsReady {
		log.V(1).Info("Pods not ready")
		return false, nil
	}

	log.V(1).Info("All resources are ready")
	return true, nil
}

func checkResourcesDeleted(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (bool, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode))

	resource := r.GetResourceManager()

	// Check StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err := resource.Get(ctx, types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, statefulSet)
	if err == nil || !errors.IsNotFound(err) {
		log.V(1).Info("StatefulSet still exists or error occurred", "error", err)
		return false, client.IgnoreNotFound(err)
	}

	// Check Secret
	secret := &corev1.Secret{}
	err = resource.Get(ctx, types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}, secret)
	if err == nil || !errors.IsNotFound(err) {
		log.V(1).Info("Secret still exists or error occurred", "error", err)
		return false, client.IgnoreNotFound(err)
	}

	// Check PVC if not in test environment
	if !r.GetIsTestEnvironment() {
		if kode.Spec.Storage != nil && kode.Spec.Storage.ExistingVolumeClaim == nil {
			pvc := &corev1.PersistentVolumeClaim{}
			err = resource.Get(ctx, types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}, pvc)
			if err == nil || !errors.IsNotFound(err) {
				log.V(1).Info("PVC still exists or error occurred", "error", err)
				return false, client.IgnoreNotFound(err)
			}
		}
	}

	// Check Service
	service := &corev1.Service{}
	err = resource.Get(ctx, types.NamespacedName{Name: kode.GetServiceName(), Namespace: kode.Namespace}, service)
	if err == nil || !errors.IsNotFound(err) {
		log.V(1).Info("Service still exists or error occurred", "error", err)
		return false, client.IgnoreNotFound(err)
	}

	// All resources are deleted
	return true, nil
}

func checkSecretReady(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	secret := &corev1.Secret{}
	if err := resource.Get(ctx, types.NamespacedName{Name: config.SecretName, Namespace: kode.Namespace}, secret); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("secret not found")
		}
		return fmt.Errorf("failed to get Secret: %w", err)
	}
	return nil
}

func checkServiceReady(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	service := &corev1.Service{}
	if err := resource.Get(ctx, types.NamespacedName{Name: config.ServiceName, Namespace: kode.Namespace}, service); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("service not found")
		}
		return fmt.Errorf("failed to get Service: %w", err)
	}
	return nil
}

func checkStatefulSetReady(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	statefulSet := &appsv1.StatefulSet{}
	if err := resource.Get(ctx, types.NamespacedName{Name: config.StatefulSetName, Namespace: kode.Namespace}, statefulSet); err != nil {
		if errors.IsNotFound(err) {
			return false, fmt.Errorf("statefulSet not found")
		}
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
		return false, nil
	}
	return true, nil
}

func checkPVCReady(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := resource.Get(ctx, types.NamespacedName{Name: config.PVCName, Namespace: kode.Namespace}, pvc); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("PVC not found")
		}
		return fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}

	if pvc.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("PVC not bound")
	}
	return nil
}

func checkPodsReady(ctx context.Context, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	podList := &corev1.PodList{}
	if err := resource.List(ctx, podList, client.InNamespace(kode.Namespace), client.MatchingLabels(config.CommonConfig.Labels)); err != nil {
		return false, fmt.Errorf("failed to list Pods: %w", err)
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
				return false, nil
			}
		}
	}
	return true, nil
}
