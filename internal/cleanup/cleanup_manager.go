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

package cleanup

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type defaultCleanupManager struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultCleanupManager(client client.Client, log logr.Logger) CleanupManager {
	return &defaultCleanupManager{
		client: client,
		log:    log,
	}
}

func (m *defaultCleanupManager) Cleanup(ctx context.Context, kode *kodev1alpha1.Kode) error {
	log := m.log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Starting cleanup of Kode resources")

	if err := m.deleteStatefulSet(ctx, kode); err != nil {
		return err
	}

	if err := m.deleteService(ctx, kode); err != nil {
		return err
	}

	// If KeepVolume is not set to true, delete PVC
	if !kode.Spec.DeepCopy().Storage.IsEmpty() && (kode.Spec.Storage.KeepVolume == nil || !*kode.Spec.Storage.KeepVolume) {
		if err := m.deletePVC(ctx, kode); err != nil {
			return err
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(kode, common.FinalizerName)
	if err := m.client.Update(ctx, kode); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return err
	}

	log.Info("Kode resources cleaned up successfully")
	return nil
}

func (m *defaultCleanupManager) Recycle(ctx context.Context, kode *kodev1alpha1.Kode) error {
	log := m.log.WithValues("kode", client.ObjectKeyFromObject(kode))
	log.Info("Starting recycle of Kode resources")

	if err := m.deleteStatefulSet(ctx, kode); err != nil {
		return err
	}

	if err := m.deleteService(ctx, kode); err != nil {
		return err
	}

	// If KeepVolume is set to true, do not delete PVC
	if !kode.Spec.DeepCopy().Storage.IsEmpty() && (kode.Spec.Storage.KeepVolume != nil && *kode.Spec.Storage.KeepVolume) {
		return nil
	}

	log.Info("Kode resources recycled successfully")
	return nil
}

func (m *defaultCleanupManager) deleteStatefulSet(ctx context.Context, kode *kodev1alpha1.Kode) error {
	return m.deleteResource(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
	}, "StatefulSet")
}

func (m *defaultCleanupManager) deleteService(ctx context.Context, kode *kodev1alpha1.Kode) error {
	return m.deleteResource(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
	}, "Service")
}

func (m *defaultCleanupManager) deletePVC(ctx context.Context, kode *kodev1alpha1.Kode) error {
	return m.deleteResource(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.GetPVCName(kode),
			Namespace: kode.Namespace,
		},
	}, "PersistentVolumeClaim")
}

func (m *defaultCleanupManager) deleteResource(ctx context.Context, obj client.Object, resourceType string) error {
	log := m.log.WithValues(resourceType, client.ObjectKeyFromObject(obj))

	if err := m.client.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s not found, skipping deletion", resourceType))
			return nil
		}
		log.Error(err, fmt.Sprintf("Failed to delete %s", resourceType))
		return err
	}

	log.Info(fmt.Sprintf("%s deleted successfully", resourceType))
	return nil
}
