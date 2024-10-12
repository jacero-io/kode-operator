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

package validation

import (
	"context"
	"fmt"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

// Validator defines the interface for validating Kode resources
type Validator interface {
	Validate(ctx context.Context, obj interface{}) error
}

type validator struct{}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) Validate(ctx context.Context, obj interface{}) error {
	switch obj := obj.(type) {
	case *kodev1alpha2.Kode:
		return v.validateKode(ctx, obj)
	case *kodev1alpha2.EntryPoint:
		return v.validateEntryPoint(ctx, obj)
	default:
		return fmt.Errorf("unsupported type: %T", obj)
	}
}
