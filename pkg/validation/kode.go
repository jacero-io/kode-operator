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
	"strings"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

func (v *validator) validateKode(ctx context.Context, kode *kodev1alpha2.Kode) error {
	if err := v.validateKodeSpec(kode); err != nil {
		return err
	}
	return nil
}

func (v *validator) validateKodeSpec(kode *kodev1alpha2.Kode) error {
	var errors []string

	// Validate InitPlugins (if any)
	for i, plugin := range kode.Spec.InitPlugins {
		if err := validateInitPlugin(plugin, i); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("kode spec validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate exisitingSecret
func validateExistingSecret(secretName string) error {
	if secretName == "" {
		return fmt.Errorf("existingSecret is required")
	}

	return nil

}

func validateInitPlugin(plugin kodev1alpha2.InitPluginSpec, index int) error {
	if plugin.Image == "" {
		return fmt.Errorf("initPlugin[%d]: image is required", index)
	}

	return nil
}
