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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/statemachine"
	"github.com/jacero-io/kode-operator/pkg/constant"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureStatefulSet ensures that the StatefulSet exists for the Kode instance
func ensureStatefulSet(ctx context.Context, r statemachine.ReconcilerInterface, resource resource.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithName("StatefulSetEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring StatefulSet")

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CommonConfig.Name,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
	}

	_, err := resource.CreateOrPatch(ctx, statefulSet, func() error {
		constructedstatefulSet, err := constructStatefulSetSpec(r, kode, config)
		if err != nil {
			return fmt.Errorf("failed to construct StatefulSet spec: %v", err)
		}

		statefulSet.Spec = constructedstatefulSet.Spec

		return controllerutil.SetControllerReference(kode, statefulSet, r.GetScheme())
	})

	if err != nil {
		return fmt.Errorf("failed to create or patch StatefulSet: %v", err)
	}

	// maskedSpec := common.MaskSpec(statefulSet.Spec.Template.Spec.Containers[0]) // Mask sensitive values
	// log.V(1).Info("StatefulSet object created", "StatefulSet", statefulSet, "Spec", maskedSpec)

	return nil
}

// constructStatefulSetSpec constructs a StatefulSet for the Kode instance
func constructStatefulSetSpec(r statemachine.ReconcilerInterface, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) (*appsv1.StatefulSet, error) {
	log := r.GetLog().WithName("SatefulSetConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	replicas := int32(1)
	templateSpec := config.Template.ContainerTemplateSpec

	var workspace string
	var mountPath string

	workspace = path.Join(templateSpec.DefaultHome, templateSpec.DefaultWorkspace)
	mountPath = templateSpec.DefaultHome
	if config.KodeSpec.Workspace != nil {
		if config.KodeSpec.Home != nil {
			workspace = path.Join(*config.KodeSpec.Home, *config.KodeSpec.Workspace)
			mountPath = *config.KodeSpec.Home
		} else {
			workspace = path.Join(templateSpec.DefaultHome, *config.KodeSpec.Workspace)
		}
	}

	var containers []corev1.Container
	var initContainers []corev1.Container

	if templateSpec.Type == "code-server" {
		log.V(1).Info("Constructing CodeServer containers")
		containers = constructCodeServerContainers(kode, config, workspace)
		log.V(1).Info("Constructed CodeServer containers", "containers", containers)
	} else if templateSpec.Type == "webtop" {
		log.V(1).Info("Constructing Webtop containers")
		containers = constructWebtopContainers(kode, config)
		log.V(1).Info("Constructed Webtop containers", "containers", containers)
	} else {
		return nil, fmt.Errorf("unknown template type: %s", templateSpec.Type)
	}

	volumes, volumeMounts := constructVolumesAndMounts(mountPath, kode, config)
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
	for _, initPlugin := range config.Template.ContainerTemplateSpec.InitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	// Add UserInitPlugins as InitContainers
	for _, initPlugin := range config.UserInitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	statefulSet := &appsv1.StatefulSet{
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

	// Set RuntimeClass if defined
	if config.Template.ContainerTemplateSpec.Runtime != "" {
		runtimeClassName := string(config.Template.ContainerTemplateSpec.Runtime)
		statefulSet.Spec.Template.Spec.RuntimeClassName = &runtimeClassName
	}

	return statefulSet, nil
}

func constructCodeServerContainers(kode *kodev1alpha2.Kode, config *common.KodeResourceConfig, workspace string) []corev1.Container {
	env := []corev1.EnvVar{
		{Name: "PUID", Value: fmt.Sprintf("%d", config.Template.ContainerTemplateSpec.PUID)},
		{Name: "PGID", Value: fmt.Sprintf("%d", config.Template.ContainerTemplateSpec.PGID)},
		{Name: "TZ", Value: config.Template.ContainerTemplateSpec.TZ},
		{Name: "PORT", Value: fmt.Sprintf("%d", config.Port)},
		{Name: "USERNAME", Value: config.Credentials.Username},
		{Name: "DEFAULT_WORKSPACE", Value: workspace},
	}

	if config.Credentials.EnableBuiltinAuth && config.Credentials.Password != "" {
		env = append(env, corev1.EnvVar{
			Name: "PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kode.GetSecretName(),
					},
					Key: "password",
				},
			},
		})
	}

	return []corev1.Container{{
		Name:  "code-server",
		Image: config.Template.ContainerTemplateSpec.Image,
		Env:   env,
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: int32(config.Port),
		}},
	}}
}

func constructWebtopContainers(kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) []corev1.Container {
	env := []corev1.EnvVar{
		{Name: "PUID", Value: fmt.Sprintf("%d", config.Template.ContainerTemplateSpec.PUID)},
		{Name: "PGID", Value: fmt.Sprintf("%d", config.Template.ContainerTemplateSpec.PGID)},
		{Name: "TZ", Value: config.Template.ContainerTemplateSpec.TZ},
		{Name: "CUSTOM_PORT", Value: fmt.Sprintf("%d", config.Port)},
		{Name: "CUSTOM_USER", Value: config.Credentials.Username},
	}

	if config.Credentials.EnableBuiltinAuth && config.Credentials.Password != "" {
		env = append(env, corev1.EnvVar{
			Name: "PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kode.GetSecretName(),
					},
					Key: "password",
				},
			},
		})
	}

	return []corev1.Container{{
		Name:  "webtop",
		Image: config.Template.ContainerTemplateSpec.Image,
		Env:   env,
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: int32(config.Port),
		}},
	}}
}

func constructVolumesAndMounts(mountPath string, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Only add volume and volume mount if storage is explicitly defined
	if config.KodeSpec.Storage != nil {

		volumeSource := corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: kode.GetPVCName(),
			},
		}

		volume := corev1.Volume{
			Name:         constant.DefaultKodeVolumeStorageName,
			VolumeSource: volumeSource,
		}

		volumeMount := corev1.VolumeMount{
			Name:      constant.DefaultKodeVolumeStorageName,
			MountPath: mountPath,
		}

		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumes, volumeMounts
}

func constructInitPluginContainer(plugin kodev1alpha2.InitPluginSpec) corev1.Container {
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
