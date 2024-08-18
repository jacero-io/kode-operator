// internal/resource/resource_manager.go

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

package resource

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/jacero-io/kode-operator/internal/common"
)

// ResourceManager defines the interface for managing Kubernetes resources
type ResourceManager interface {
	CreateOrPatch(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error
	Delete(ctx context.Context, obj client.Object) error
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

type defaultResourceManager struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewDefaultResourceManager(client client.Client, log logr.Logger, scheme *runtime.Scheme) ResourceManager {
	return &defaultResourceManager{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (m *defaultResourceManager) CreateOrPatch(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error {
	// Add type information to the object
	if err := common.AddTypeInformationToObject(m.Scheme, obj); err != nil {
		return fmt.Errorf("failed to add type information to object: %w", err)
	}
	log := m.Log.WithValues(
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
	)

	log.V(1).Info("CreateOrPatch called",
		"objectType", fmt.Sprintf("%T", obj),
		"gvk", obj.GetObjectKind().GroupVersionKind().String())

	// Proceed with CreateOrPatch
	result, err := controllerutil.CreateOrPatch(ctx, m.Client, obj, f)
	if err != nil {
		log.Error(err, "Failed to create or patch resource")
		return err
	}

	log.V(1).Info("Resource operation completed", "Result", result)
	return nil
}

func (m *defaultResourceManager) Delete(ctx context.Context, obj client.Object) error {
	return m.Client.Delete(ctx, obj)
}

func (m *defaultResourceManager) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return m.Client.Get(ctx, key, obj)
}

func (m *defaultResourceManager) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return m.Client.List(ctx, list, opts...)
}
