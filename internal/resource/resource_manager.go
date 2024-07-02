package resource

import (
	"context"
	"fmt"

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
		err = m.updateSpec(obj, updated)
		if err != nil {
			log.Error(err, "Failed to update spec", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
			return err
		}

		if err := m.client.Update(ctx, updated); err != nil {
			log.Error(err, "Failed to update resource", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
			return err
		}
		log.Info("Resource updated", "Kind", obj.GetObjectKind().GroupVersionKind().Kind, "Name", obj.GetName())
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

func (m *defaultResourceManager) updateSpec(new, existing client.Object) error {
	switch v := existing.(type) {
	case *appsv1.StatefulSet:
		v.Spec = new.(*appsv1.StatefulSet).Spec
	case *corev1.Service:
		v.Spec = new.(*corev1.Service).Spec
	case *corev1.PersistentVolumeClaim:
		v.Spec = new.(*corev1.PersistentVolumeClaim).Spec
	default:
		return fmt.Errorf("unsupported resource type: %T", existing)
	}
	return nil
}
