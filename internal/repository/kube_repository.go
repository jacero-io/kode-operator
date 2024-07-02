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

package repository

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultRepository struct {
	client client.Client
}

func NewDefaultRepository(c client.Client) Repository {
	return &defaultRepository{client: c}
}

func (r *defaultRepository) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	return r.client.Get(ctx, key, obj)
}

func (r *defaultRepository) Create(ctx context.Context, obj client.Object) error {
	return r.client.Create(ctx, obj)
}

func (r *defaultRepository) Update(ctx context.Context, obj client.Object) error {
	return r.client.Update(ctx, obj)
}

func (r *defaultRepository) Delete(ctx context.Context, obj client.Object) error {
	return r.client.Delete(ctx, obj)
}

func (r *defaultRepository) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return r.client.List(ctx, list, opts...)
}
