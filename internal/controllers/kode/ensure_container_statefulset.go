// internal/controllers/kode/ensure_statefulset.go

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
	"path"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureStatefulSet ensures that the StatefulSet exists for the Kode instance
func (r *KodeReconciler) ensureStatefulSet(ctx context.Context, config *common.KodeResourceConfig, kode *kodev1alpha1.Kode) error {
	log := r.Log.WithName("StatefulSetEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring StatefulSet")

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CommonConfig.Name,
			Namespace: config.CommonConfig.Namespace,
		},
	}

	err := r.ResourceManager.CreateOrPatch(ctx, statefulSet, func() error {
		constructedstatefulSet, err := r.constructStatefulSetSpec(config)
		if err != nil {
			return fmt.Errorf("failed to construct StatefulSet spec: %v", err)
		}

		statefulSet.Spec = constructedstatefulSet.Spec
		statefulSet.ObjectMeta.Labels = constructedstatefulSet.ObjectMeta.Labels

		return controllerutil.SetControllerReference(kode, statefulSet, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to create or patch StatefulSet: %v", err)
	}

	// maskedSpec := common.MaskSpec(statefulSet.Spec.Template.Spec.Containers[0]) // Mask sensitive values
	// log.V(1).Info("StatefulSet object created", "StatefulSet", statefulSet, "Spec", maskedSpec)

	return nil
}

// constructStatefulSetSpec constructs a StatefulSet for the Kode instance
func (r *KodeReconciler) constructStatefulSetSpec(config *common.KodeResourceConfig) (*appsv1.StatefulSet, error) {
	log := r.Log.WithName("SatefulSetConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	replicas := int32(1)
	templateSpec := config.Templates.KodeTemplate

	var workspace string
	var mountPath string

	workspace = path.Join(templateSpec.ContainerSpec.DefaultHome, templateSpec.ContainerSpec.DefaultWorkspace)
	mountPath = templateSpec.ContainerSpec.DefaultHome
	if config.KodeSpec.Workspace != "" {
		if config.KodeSpec.Home != "" {
			workspace = path.Join(config.KodeSpec.Home, config.KodeSpec.Workspace)
			mountPath = config.KodeSpec.Home
		} else {
			workspace = path.Join(templateSpec.ContainerSpec.DefaultHome, config.KodeSpec.Workspace)
		}
	}

	var containers []corev1.Container
	var initContainers []corev1.Container

	if templateSpec.ContainerSpec.Type == "code-server" {
		containers = constructCodeServerContainers(config, workspace)
	} else if templateSpec.ContainerSpec.Type == "webtop" {
		containers = constructWebtopContainers(config)
	} else {
		return nil, fmt.Errorf("unknown template type: %s", templateSpec.ContainerSpec.Type)
	}

	volumes, volumeMounts := constructVolumesAndMounts(mountPath, config)
	log.V(1).Info("Constructed volumes and mounts", "volumes", volumes, "volumeMounts", volumeMounts)
	containers[0].VolumeMounts = volumeMounts

	// If KodeResourceConfig has initContainers, append to initContainers
	if config.InitContainers != nil {
		initContainers = append(initContainers, config.InitContainers...)
		for _, container := range config.Containers {
			log.V(1).Info("InitContainer added", "Name", container.Name)
			log.V(2).Info("InitContainer added", "Name", container.Name, "Container", container)
		}
	}

	// If KodeResourceConfig has containers, append to containers
	if config.Containers != nil {
		containers = append(containers, config.Containers...)
		for _, container := range config.Containers {
			log.V(1).Info("Container added", "Name", container.Name)
			log.V(2).Info("Container added", "Name", container.Name, "Container", container)
		}
	}

	// Add TemplateInitPlugins as InitContainers
	for _, initPlugin := range config.TemplateInitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	// Add UserInitPlugins as InitContainers
	for _, initPlugin := range config.UserInitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.StatefulSetName,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: config.CommonConfig.Labels,
			},
			ServiceName: config.ServiceName,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:            config.CommonConfig.Labels,
					CreationTimestamp: metav1.Time{},
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers:     containers,
					Volumes:        volumes,
				},
			},
		},
	}

	return statefulSet, nil
}

func constructCodeServerContainers(config *common.KodeResourceConfig,
	workspace string) []corev1.Container {

	return []corev1.Container{{
		Name:  "code-server",
		Image: config.Templates.KodeTemplate.ContainerSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PUID", Value: fmt.Sprintf("%d", config.Templates.KodeTemplate.ContainerSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", config.Templates.KodeTemplate.ContainerSpec.PGID)},
			{Name: "TZ", Value: config.Templates.KodeTemplate.ContainerSpec.TZ},
			{Name: "PORT", Value: fmt.Sprintf("%d", config.LocalServicePort)},
			{Name: "USERNAME", Value: config.KodeSpec.Credentials.Username},
			// {Name: "PASSWORD", Value: config.Kode.Spec.Password},
			{Name: "DEFAULT_WORKSPACE", Value: workspace},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: config.LocalServicePort,
		}},
	}}
}

func constructWebtopContainers(config *common.KodeResourceConfig) []corev1.Container {

	return []corev1.Container{{
		Name:  "webtop",
		Image: config.Templates.KodeTemplate.ContainerSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PUID", Value: fmt.Sprintf("%d", config.Templates.KodeTemplate.ContainerSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", config.Templates.KodeTemplate.ContainerSpec.PGID)},
			{Name: "TZ", Value: config.Templates.KodeTemplate.ContainerSpec.TZ},
			{Name: "CUSTOM_PORT", Value: fmt.Sprintf("%d", config.LocalServicePort)},
			{Name: "CUSTOM_USER", Value: config.KodeSpec.Credentials.Username},
			// {Name: "PASSWORD", Value: config.Kode.Spec.Password},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: config.LocalServicePort,
		}},
	}}
}

func constructVolumesAndMounts(mountPath string, config *common.KodeResourceConfig) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Only add volume and volume mount if storage is explicitly defined
	if !config.KodeSpec.Storage.IsEmpty() {
		var volumeSource corev1.VolumeSource

		if config.KodeSpec.Storage.ExistingVolumeClaim != "" {
			volumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.KodeSpec.Storage.ExistingVolumeClaim,
				},
			}
		} else {
			volumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.PVCName,
				},
			}
		}

		volume := corev1.Volume{
			Name:         common.KodeVolumeStorageName,
			VolumeSource: volumeSource,
		}

		volumeMount := corev1.VolumeMount{
			Name:      common.KodeVolumeStorageName,
			MountPath: mountPath,
		}

		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumes, volumeMounts
}

func constructInitPluginContainer(plugin kodev1alpha1.InitPluginSpec) corev1.Container {
	return corev1.Container{
		Name:         "plugin-" + plugin.Name,
		Image:        plugin.Image,
		Command:      plugin.Command,
		Args:         plugin.Args,
		Env:          plugin.Env,
		EnvFrom:      plugin.EnvFrom,
		VolumeMounts: plugin.VolumeMounts,
	}
}
