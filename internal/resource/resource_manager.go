package resource

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultResourceManager struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultResourceManager(client client.Client, log logr.Logger) ResourceManager {
	return &defaultResourceManager{
		client: client,
		log:    log,
	}
}

func (m *defaultResourceManager) Ensure(ctx context.Context, obj client.Object) error {
	log := m.log.WithValues("resource", client.ObjectKeyFromObject(obj))

	err := m.client.Create(ctx, obj)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create resource", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
			return err
		}

		// Resource already exists, try to update
		existing := obj.DeepCopyObject().(client.Object)
		key := client.ObjectKeyFromObject(obj)
		if err := m.client.Get(ctx, key, existing); err != nil {
			log.Error(err, "Failed to get existing resource", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
			return err
		}

		// Update the existing object with new spec
		updated := existing.DeepCopyObject().(client.Object)
		var updateNeeded bool
		var updateErr error

		switch v := updated.(type) {
		case *corev1.PersistentVolumeClaim:
			updateNeeded = m.updatePVCSpec(obj.(*corev1.PersistentVolumeClaim), v)
		default:
			updateNeeded, updateErr = m.updateSpec(obj, v)
		}

		if updateErr != nil {
			log.Error(updateErr, "Failed to update spec", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
			return updateErr
		}

		if updateNeeded {
			if err := m.client.Update(ctx, updated); err != nil {
				log.Error(err, "Failed to update resource", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
				return err
			}
			log.Info("Resource updated", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
		} else {
			log.Info("No update needed", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
		}
	} else {
		log.Info("Resource created", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
	}

	return nil
}

func (m *defaultResourceManager) Delete(ctx context.Context, obj client.Object) error {
	return m.client.Delete(ctx, obj)
}

func (m *defaultResourceManager) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return m.client.Get(ctx, key, obj)
}

func (m *defaultResourceManager) Update(ctx context.Context, obj client.Object) error {
	return m.client.Update(ctx, obj)
}

func (m *defaultResourceManager) updateSpec(new, existing client.Object) (bool, error) {
	updated := false
	switch v := existing.(type) {
	case *appsv1.StatefulSet:
		if !reflect.DeepEqual(v.Spec.Template, new.(*appsv1.StatefulSet).Spec.Template) {
			v.Spec.Template = new.(*appsv1.StatefulSet).Spec.Template
			updated = true
		}
		// Update other mutable fields as needed
	case *corev1.Service:
		if !reflect.DeepEqual(v.Spec.Ports, new.(*corev1.Service).Spec.Ports) {
			v.Spec.Ports = new.(*corev1.Service).Spec.Ports
			updated = true
		}
		// Update other mutable fields as needed
	default:
		return false, fmt.Errorf("unsupported resource type: %T", existing)
	}
	return updated, nil
}

func (m *defaultResourceManager) updatePVCSpec(new, existing *corev1.PersistentVolumeClaim) bool {
	updated := false

	// Update labels
	if !reflect.DeepEqual(existing.Labels, new.Labels) {
		existing.Labels = new.Labels
		updated = true
	}

	// Update annotations
	if !reflect.DeepEqual(existing.Annotations, new.Annotations) {
		existing.Annotations = new.Annotations
		updated = true
	}

	// Update resources.requests.storage (only if increasing)
	if new.Spec.Resources.Requests != nil && existing.Spec.Resources.Requests != nil {
		newStorage := new.Spec.Resources.Requests[corev1.ResourceStorage]
		existingStorage := existing.Spec.Resources.Requests[corev1.ResourceStorage]
		if newStorage.Cmp(existingStorage) > 0 {
			existing.Spec.Resources.Requests[corev1.ResourceStorage] = newStorage
			updated = true
		}
	}
	return updated
}
