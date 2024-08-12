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
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type KodeCleanupResource struct {
	Kode *kodev1alpha2.Kode
}

func NewKodeCleanupResource(kode *kodev1alpha2.Kode) cleanup.CleanupableResource {
	return &KodeCleanupResource{Kode: kode}
}

func (k *KodeCleanupResource) GetResources() []cleanup.Resource {
	resources := []cleanup.Resource{
		{
			Name:      k.Kode.Name,
			Namespace: k.Kode.Namespace,
			Kind:      "StatefulSet",
			Object:    &appsv1.StatefulSet{},
		},
		{
			Name:      GetServiceName(k.Kode),
			Namespace: k.Kode.Namespace,
			Kind:      "Service",
			Object:    &corev1.Service{},
		},
	}

	// Add PVC to resources only if it's not using an existing claim
	if k.Kode.Spec.Storage.ExistingVolumeClaim == "" {
		resources = append(resources, cleanup.Resource{
			Name:      GetPVCName(k.Kode),
			Namespace: k.Kode.Namespace,
			Kind:      "PersistentVolumeClaim",
			Object:    &corev1.PersistentVolumeClaim{},
		})
	}

	return resources
}

func (k *KodeCleanupResource) ShouldDelete(resource cleanup.Resource) bool {
	// Delete all resources except PVC if KeepVolume is true
	if k.Kode.Spec.Storage.KeepVolume != nil && *k.Kode.Spec.Storage.KeepVolume {
		return resource.Kind != "PersistentVolumeClaim"
	}
	return true
}
