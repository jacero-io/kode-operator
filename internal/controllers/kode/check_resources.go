package kode

import (
	"context"
	"fmt"
	"reflect"
	"runtime"

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

// ResourceReadyCheck defines a function type for resource readiness checks
type ResourceReadyCheck func(context.Context, statemachine.ReconcilerInterface, resourcev1.ResourceManager, *kodev1alpha2.Kode, *common.KodeResourceConfig) (bool, error)

func checkResourcesReady(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	log.V(1).Info("Starting resource readiness checks")

	resource := r.GetResourceManager()
	checks := ComposeResourceChecks()

	log.V(1).Info("Resource checks composed", "totalChecks", len(checks))

	// Execute each check in sequence
	for i, check := range checks {
		checkName := runtime.FuncForPC(reflect.ValueOf(check).Pointer()).Name()
		log.V(1).Info("Executing resource check",
			"checkNumber", i+1,
			"checkName", checkName)

		ready, err := check(ctx, r, resource, kode, config)
		if err != nil {
			log.Error(err, "Resource check failed",
				"checkName", checkName)
			return false, err
		}
		if !ready {
			log.V(1).Info("Resource not ready",
				"checkName", checkName)
			return false, nil
		}
		log.V(1).Info("Resource check passed",
			"checkName", checkName)
	}

	log.V(1).Info("All resource checks passed successfully")
	return true, nil
}

func checkResourcesDeleted(ctx context.Context, r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode) (bool, error) {
	log := r.GetLog().WithValues("kode", client.ObjectKeyFromObject(kode), "phase", kode.Status.Phase)
	resource := r.GetResourceManager()

	// Define resources to check for deletion
	resourcesToCheck := []struct {
		name      string
		nameFunc  func() string
		resource  client.Object
		skipCheck func() bool
	}{
		{
			name:     "StatefulSet",
			nameFunc: kode.GetStatefulSetName,
			resource: &appsv1.StatefulSet{},
		},
		{
			name:     "Secret",
			nameFunc: kode.GetSecretName,
			resource: &corev1.Secret{},
		},
		{
			name:     "Service",
			nameFunc: kode.GetServiceName,
			resource: &corev1.Service{},
		},
		{
			name:     "PVC",
			nameFunc: kode.GetPVCName,
			resource: &corev1.PersistentVolumeClaim{},
			skipCheck: func() bool {
				return r.GetIsTestEnvironment() ||
					kode.Spec.Storage == nil ||
					kode.Spec.Storage.ExistingVolumeClaim != nil
			},
		},
	}

	// Check each resource
	for _, res := range resourcesToCheck {
		if res.skipCheck != nil && res.skipCheck() {
			continue
		}

		err := resource.Get(ctx, types.NamespacedName{
			Name:      res.nameFunc(),
			Namespace: kode.Namespace,
		}, res.resource)

		if err == nil || !errors.IsNotFound(err) {
			log.V(1).Info(fmt.Sprintf("%s still exists or error occurred", res.name),
				"error", err,
				"name", res.nameFunc(),
				"namespace", kode.Namespace)
			return false, client.IgnoreNotFound(err)
		}
	}

	log.V(1).Info("All resources are deleted")
	return true, nil
}

// checkSecretReady only checks if Secret exists and has required fields
func checkSecretReady(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, _ *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues(
		"kode", fmt.Sprintf("%s/%s", kode.Namespace, kode.Name), "phase", kode.Status.Phase,
		"function", "checkSecretReady")

	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{Name: kode.GetSecretName(), Namespace: kode.Namespace}

	log.V(1).Info("Checking secret readiness",
		"secretName", namespacedName.Name,
		"namespace", namespacedName.Namespace)

	if err := resource.Get(ctx, namespacedName, secret); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Secret not found")
			return false, nil
		}
		log.Error(err, "Failed to get Secret")
		return false, fmt.Errorf("failed to get Secret: %w", err)
	}

	// Check required fields
	hasUsername := false
	hasPassword := false
	if _, exists := secret.Data["username"]; exists {
		hasUsername = true
	}
	if _, exists := secret.Data["password"]; exists {
		hasPassword = true
	}

	log.V(1).Info("Secret field check",
		"hasUsername", hasUsername,
		"hasPassword", hasPassword,
		"requiresPassword", kode.Spec.Credentials != nil && kode.Spec.Credentials.EnableBuiltinAuth)

	if !hasUsername || (kode.Spec.Credentials != nil && kode.Spec.Credentials.EnableBuiltinAuth && !hasPassword) {
		return false, nil
	}

	log.V(1).Info("Secret is ready")
	return true, nil
}

// checkServiceReady only checks if Service exists and has correct port
func checkServiceReady(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues(
		"kode", fmt.Sprintf("%s/%s", kode.Namespace, kode.Name), "phase", kode.Status.Phase,
		"function", "checkServiceReady")

	service := &corev1.Service{}
	namespacedName := types.NamespacedName{Name: kode.GetServiceName(), Namespace: kode.Namespace}

	log.V(1).Info("Checking service readiness",
		"serviceName", namespacedName.Name,
		"namespace", namespacedName.Namespace,
		"expectedPort", config.Port)

	if err := resource.Get(ctx, namespacedName, service); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Service not found")
			return false, nil
		}
		log.Error(err, "Failed to get Service")
		return false, fmt.Errorf("failed to get Service: %w", err)
	}

	// Check only for port existence
	for _, port := range service.Spec.Ports {
		if port.Port == int32(config.Port) {
			log.V(1).Info("Service is ready")
			return true, nil
		}
	}

	log.V(1).Info("Service missing required port",
		"expectedPort", config.Port)
	return false, nil
}

// checkStatefulSetReady checks if StatefulSet is ready and replicas are available
func checkStatefulSetReady(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, _ *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues(
		"kode", fmt.Sprintf("%s/%s", kode.Namespace, kode.Name), "phase", kode.Status.Phase,
		"function", "checkStatefulSetReady")

	statefulSet := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{Name: kode.GetStatefulSetName(), Namespace: kode.Namespace}

	if err := resource.Get(ctx, namespacedName, statefulSet); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("StatefulSet not found")
			return false, nil
		}
		log.Error(err, "Failed to get StatefulSet")
		return false, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	log.V(1).Info("StatefulSet status",
		"readyReplicas", statefulSet.Status.ReadyReplicas,
		"desiredReplicas", *statefulSet.Spec.Replicas)

	ready := statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas
	if ready {
		log.V(1).Info("StatefulSet is ready")
	}
	return ready, nil
}

// checkPVCReady checks if PVC exists and is bound
func checkPVCReady(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, _ *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues(
		"kode", fmt.Sprintf("%s/%s", kode.Namespace, kode.Name), "phase", kode.Status.Phase,
		"function", "checkPVCReady")

	// Skip if storage not configured
	if kode.Spec.Storage == nil {
		log.V(1).Info("Storage not specified, skipping PVC check")
		return true, nil
	}

	// Skip if using existing volume claim
	if kode.Spec.Storage.ExistingVolumeClaim != nil {
		log.V(1).Info("Using existing volume claim, skipping PVC check",
			"existingClaim", *kode.Spec.Storage.ExistingVolumeClaim)
		return true, nil
	}

	pvc := &corev1.PersistentVolumeClaim{}
	namespacedName := types.NamespacedName{Name: kode.GetPVCName(), Namespace: kode.Namespace}

	log.V(1).Info("Checking PVC readiness",
		"pvcName", namespacedName.Name,
		"namespace", namespacedName.Namespace)

	if err := resource.Get(ctx, namespacedName, pvc); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("PVC not found")
			return false, nil
		}
		log.Error(err, "Failed to get PVC")
		return false, fmt.Errorf("failed to get PVC: %w", err)
	}

	log.V(1).Info("PVC status",
		"phase", pvc.Status.Phase,
		"capacity", pvc.Status.Capacity)

	ready := pvc.Status.Phase == corev1.ClaimBound
	if ready {
		log.V(1).Info("PVC is ready")
	} else {
		log.V(1).Info("PVC not ready", "currentPhase", pvc.Status.Phase)
	}
	return ready, nil
}

// checkPodsReady checks if pods are running and ready
func checkPodsReady(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (bool, error) {
	log := r.GetLog().WithValues(
		"kode", fmt.Sprintf("%s/%s", kode.Namespace, kode.Name), "phase", kode.Status.Phase,
		"function", "checkPodsReady")

	podList := &corev1.PodList{}
	if err := resource.List(ctx, podList,
		client.InNamespace(kode.Namespace),
		client.MatchingLabels(config.CommonConfig.Labels)); err != nil {
		log.Error(err, "Failed to list pods")
		return false, fmt.Errorf("failed to list pods: %w", err)
	}

	log.V(1).Info("Retrieved pod list",
		"podCount", len(podList.Items))

	if len(podList.Items) == 0 {
		log.V(1).Info("No pods found")
		return false, nil
	}

	for _, pod := range podList.Items {
		log.V(1).Info("Checking pod status",
			"podName", pod.Name,
			"phase", pod.Status.Phase,
			"ready", isPodReady(&pod))

		if pod.Status.Phase != corev1.PodRunning || !isPodReady(&pod) {
			log.V(1).Info("Pod not ready",
				"podName", pod.Name,
				"phase", pod.Status.Phase,
				"ready", isPodReady(&pod))
			return false, nil
		}
	}

	log.V(1).Info("All pods are ready")
	return true, nil
}

// isPodReady checks if all containers in the pod are ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// ComposeResourceChecks returns a slice of all resource ready checks
func ComposeResourceChecks() []ResourceReadyCheck {
	return []ResourceReadyCheck{
		checkSecretReady,
		checkServiceReady,
		checkStatefulSetReady,
		checkPVCReady,
		checkPodsReady,
	}
}
