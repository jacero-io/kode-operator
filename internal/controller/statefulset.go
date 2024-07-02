/*
Copyright 2024.

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
	"path"
	"reflect"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/common"
	"github.com/emil-jacero/kode-operator/internal/envoy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureStatefulSet ensures that the StatefulSet exists for the Kode instance
func (r *KodeReconciler) ensureStatefulSet(ctx context.Context,
	kode *kodev1alpha1.Kode,
	labels map[string]string,
	templates *common.Templates) error {
	log := r.Log.WithName("StatefulSetEnsurer").WithValues("kode", client.ObjectKeyFromObject(kode))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring StatefulSet exists")

	statefulSet := r.constructStatefulSet(kode, labels, templates)
	if err := controllerutil.SetControllerReference(kode, statefulSet, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	err := r.ResourceManager.Ensure(ctx, statefulSet)
	if err != nil {
		log.Error(err, "Failed to ensure StatefulSet")
		return fmt.Errorf("failed to ensure StatefulSet: %w", err)
	}

	log.Info("Successfully ensured StatefulSet")
	return nil
}

// constructStatefulSet constructs a StatefulSet for the Kode instance
func (r *KodeReconciler) constructStatefulSet(kode *kodev1alpha1.Kode,
	labels map[string]string,
	templates *common.Templates) *appsv1.StatefulSet {
	log := r.Log.WithName("SatefulSetConstructor").WithValues("kode", client.ObjectKeyFromObject(kode))

	replicas := int32(1)
	templateSpec := templates.KodeTemplate

	var workspace string
	var mountPath string

	workspace = path.Join(templateSpec.DefaultHome, templateSpec.DefaultWorkspace)
	mountPath = templateSpec.DefaultHome
	if kode.Spec.Workspace != "" {
		if kode.Spec.Home != "" {
			workspace = path.Join(kode.Spec.Home, kode.Spec.Workspace)
			mountPath = kode.Spec.Home
		} else {
			workspace = path.Join(templateSpec.DefaultHome, kode.Spec.Workspace)
		}
	}

	var containers []corev1.Container
	var initContainers []corev1.Container

	if templateSpec.Type == "code-server" {
		containers = constructCodeServerContainers(kode, templateSpec, workspace)
	} else if templateSpec.Type == "webtop" {
		containers = constructWebtopContainers(kode, templateSpec)
	}

	volumes, volumeMounts := constructVolumesAndMounts(mountPath, kode)
	containers[0].VolumeMounts = volumeMounts

	if templates.EnvoyProxyConfig != nil {
		log.Info("EnvoyProxyConfig is defined", "Namespace", kode.Namespace, "Kode", kode.Name)
		envoySidecarContainer, envoyInitContainer, err := envoy.NewContainerConstructor(r.Log, envoy.NewBootstrapConfigGenerator(r.Log.WithName("EnvoyContainerConstructor"))).ConstructEnvoyProxyContainer(templateSpec, templates.EnvoyProxyConfig)
		if err != nil {
			log.Error(err, "Failed to construct EnvoyProxy sidecar", "Kode", kode.Name, "Error", err)
		} else {
			containers = append(containers, envoySidecarContainer)
			initContainers = append(initContainers, envoyInitContainer)
			log.Info("Added EnvoyProxy sidecar container and init container", "Kode", kode.Name, "Container", envoySidecarContainer.Name)
		}
	}

	// Add InitPlugins as InitContainers
	for _, initPlugin := range kode.Spec.InitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: kode.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:            labels,
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

	// common.logStatefulSetManifest(log, statefulSet)
	return statefulSet
}

func constructCodeServerContainers(kode *kodev1alpha1.Kode,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	workspace string) []corev1.Container {

	return []corev1.Container{{
		Name:  "kode-" + kode.Name,
		Image: templateSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PORT", Value: fmt.Sprintf("%d", templateSpec.Port)},
			{Name: "PUID", Value: fmt.Sprintf("%d", templateSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", templateSpec.PGID)},
			{Name: "TZ", Value: templateSpec.TZ},
			{Name: "USERNAME", Value: kode.Spec.User},
			{Name: "PASSWORD", Value: kode.Spec.Password},
			{Name: "DEFAULT_WORKSPACE", Value: workspace},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: templateSpec.Port,
		}},
	}}
}

func constructWebtopContainers(kode *kodev1alpha1.Kode,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec) []corev1.Container {

	return []corev1.Container{{
		Name:  "kode-" + kode.Name,
		Image: templateSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PUID", Value: fmt.Sprintf("%d", templateSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", templateSpec.PGID)},
			{Name: "TZ", Value: templateSpec.TZ},
			{Name: "CUSTOM_PORT", Value: fmt.Sprintf("%d", templateSpec.Port)},
			{Name: "CUSTOM_USER", Value: kode.Spec.User},
			{Name: "PASSWORD", Value: kode.Spec.Password},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: templateSpec.Port,
		}},
	}}
}

func constructVolumesAndMounts(mountPath string, kode *kodev1alpha1.Kode) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add volume and volume mount if storage is defined
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		var volumeSource corev1.VolumeSource

		if kode.Spec.Storage.ExistingVolumeClaim != "" {
			volumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: kode.Spec.Storage.ExistingVolumeClaim,
				},
			}
		} else if !kode.Spec.DeepCopy().Storage.IsEmpty() {
			volumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: common.GetPVCName(kode),
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
		Name:    "plugin-" + plugin.Name,
		Image:   plugin.Image,
		Command: plugin.Command,
		Args:    plugin.Args,
		Env:     plugin.EnvVars,
	}
}
