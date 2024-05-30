package controller

import (
	"fmt"

	"encoding/json"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

const (
	EnvoyProxyContainerName = "envoy-proxy"
	EnvoyConfigVolumeName   = "envoy-config"
	EnvoyConfigMountPath    = "/etc/envoy"
)

func generateEnvoyProxyConfig(envoyProxyTemplate kodev1alpha1.EnvoyProxyTemplate) (string, error) {
	// Convert the struct to JSON
	jsonData, err := json.Marshal(envoyProxyTemplate)
	if err != nil {
		return "", fmt.Errorf("error marshalling to JSON: %w", err)
	}

	// Convert JSON to YAML
	var yamlData map[string]interface{}
	if err := json.Unmarshal(jsonData, &yamlData); err != nil {
		return "", fmt.Errorf("error unmarshalling JSON to map: %w", err)
	}
	yamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		return "", fmt.Errorf("error marshalling to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// createEnvoyConfigMap creates a ConfigMap for the EnvoyProxy configuration
// func createEnvoyConfigMap(ctx context.Context, c client.Client, namespace string, proxyRef *corev1.ObjectReference) (*corev1.ConfigMap, error) {
// 	envoyProxy := &kodev1alpha1.EnvoyProxy{}
// 	err := c.Get(ctx, client.ObjectKey{
// 		Namespace: proxyRef.Namespace,
// 		Name:      proxyRef.Name,
// 	}, envoyProxy)
// 	if err != nil {
// 		return nil, err
// 	}

// 	configMap := &corev1.ConfigMap{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      proxyRef.Name + "-config",
// 			Namespace: namespace,
// 		},
// 		Data: map[string]string{
// 			"envoy.yaml": envoyProxy.Spec.Config, // Assuming EnvoyProxySpec has a Config field with the configuration in string format
// 		},
// 	}

// 	if err := c.Create(ctx, configMap); err != nil {
// 		return nil, err
// 	}

// 	return configMap, nil
// }

// addEnvoyProxySidecar mutates a PodSpec to include the EnvoyProxy sidecar
// func addEnvoyProxySidecar(ctx context.Context, c client.Client, namespace string, podSpec *corev1.PodSpec, proxyRef *corev1.ObjectReference) error {
// 	configMap, err := createEnvoyConfigMap(ctx, c, namespace, proxyRef)
// 	if err != nil {
// 		return err
// 	}

// 	envoyProxy := &kodev1alpha1.EnvoyProxy{}
// 	err = c.Get(ctx, client.ObjectKey{
// 		Namespace: proxyRef.Namespace,
// 		Name:      proxyRef.Name,
// 	}, envoyProxy)
// 	if err != nil {
// 		return err
// 	}

// 	envoyContainer := corev1.Container{
// 		Name:  EnvoyProxyContainerName,
// 		Image: envoyProxy.Spec.Image, // Assuming EnvoyProxySpec has an Image field
// 		VolumeMounts: []corev1.VolumeMount{
// 			{
// 				Name:      EnvoyConfigVolumeName,
// 				MountPath: EnvoyConfigMountPath,
// 				SubPath:   "envoy.yaml",
// 			},
// 		},
// 		Ports: []corev1.ContainerPort{
// 			{
// 				Name:          "http",
// 				ContainerPort: 80,
// 			},
// 			{
// 				Name:          "https",
// 				ContainerPort: 443,
// 			},
// 		},
// 	}

// 	// Add the volume for the ConfigMap
// 	podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
// 		Name: EnvoyConfigVolumeName,
// 		VolumeSource: corev1.VolumeSource{
// 			ConfigMap: &corev1.ConfigMapVolumeSource{
// 				LocalObjectReference: corev1.LocalObjectReference{
// 					Name: configMap.Name,
// 				},
// 			},
// 		},
// 	})

// 	podSpec.Containers = append(podSpec.Containers, envoyContainer)
// 	return nil
// }
