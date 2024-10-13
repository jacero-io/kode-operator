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

package v1alpha2

import (
	"context"
	"fmt"
	"strings"

	"github.com/jacero-io/kode-operator/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid  bool
	Errors []error
}

// Validatable is an interface that should be implemented by resources that can be validated
type Validatable interface {
	Validate(ctx context.Context) ValidationResult
}

// SetValidationCondition updates the validation condition based on the validation result
func SetValidationCondition(status *CommonStatus, result ValidationResult) {
	if result.Valid {
		status.SetCondition(constant.ConditionTypeValidated, metav1.ConditionTrue, "ValidationSucceeded", "Resource validation succeeded")
	} else {
		errorStrings := make([]string, len(result.Errors))
		for i, err := range result.Errors {
			errorStrings[i] = err.Error()
		}
		errorMsg := strings.Join(errorStrings, "; ")
		status.SetCondition(constant.ConditionTypeValidated, metav1.ConditionFalse, "ValidationFailed", fmt.Sprintf("Resource validation failed: %s", errorMsg))
	}
}

// Validate for Kode performs validation on a Kode resource
func (k *Kode) Validate(ctx context.Context) ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate InitPlugins
	for i, plugin := range k.Spec.InitPlugins {
		if err := validateInitPlugin(plugin, i); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err)
		}
	}

	return result
}

// Validate for EntryPoint performs validation on an EntryPoint resource
func (e *EntryPoint) Validate(ctx context.Context) ValidationResult {
	result := ValidationResult{Valid: true}

	return result
}

// Helper functions

func validateInitPlugin(plugin InitPluginSpec, index int) error {
	if plugin.Image == "" {
		return fmt.Errorf("initPlugin[%d]: image is required", index)
	}
	return nil
}
