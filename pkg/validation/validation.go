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
	"fmt"
	"strings"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ValidationResult struct {
	Valid  bool
	Errors []string
}

// SetValidationCondition updates the validation condition based on the validation result
func SetValidationCondition(status *kodev1alpha2.CommonStatus, result ValidationResult) {
	if result.Valid {
		status.SetCondition(constant.ConditionTypeValidated, metav1.ConditionTrue, "ValidationSucceeded", "Resource validation succeeded")
	} else {
		errorMsg := strings.Join(result.Errors, "; ")
		status.SetCondition(constant.ConditionTypeValidated, metav1.ConditionFalse, "ValidationFailed", fmt.Sprintf("Resource validation failed: %s", errorMsg))
	}
}
