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

package common

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// masks sensitive string values
func MaskSensitiveValue(s string) string {
	if len(s) > 6 {
		return s[:2] + "******" + s[len(s)-2:]
	}
	return "****"
}

// masks sensitive environment variables
func MaskEnvVars(envs []corev1.EnvVar) []corev1.EnvVar {
	maskedEnvs := make([]corev1.EnvVar, len(envs))
	for i, env := range envs {
		if env.Name == "PASSWORD" || env.Name == "SECRET" {
			env.Value = MaskSensitiveValue(env.Value)
		}
		maskedEnvs[i] = env
	}
	return maskedEnvs
}

// logs an object's key details
func LogObject(log logr.Logger, obj client.Object, msg string) {
	log.Info(msg,
		"Kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"Namespace", obj.GetNamespace(),
		"Name", obj.GetName(),
	)
}

// checks if a string slice contains a specific string
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removes a string from a slice of strings
func RemoveString(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// joins multiple errors into a single error
func JoinErrors(errs ...error) error {
	var errStrings []string
	for _, err := range errs {
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}
	if len(errStrings) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errStrings, "; "))
}

// checks if an error is a "not found" error
func IsNotFound(err error) bool {
	return errors.IsNotFound(err)
}

// creates a types.NamespacedName from a namespace and name
func NamespacedName(namespace string, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}

// wraps a context with a timeout
func ContextWithTimeout(ctx context.Context, timeoutSeconds int64) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
}

// returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}

// returns a pointer to an int32
func Int32Ptr(i int32) *int32 {
	return &i
}

// returns a pointer to an int64
func Int64Ptr(i int64) *int64 {
	return &i
}

// returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// gets an environment variable or returns a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// merges multiple sets of labels
func MergeLabels(labelsSlice ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, labels := range labelsSlice {
		for k, v := range labels {
			result[k] = v
		}
	}
	return result
}

// returns the name of the PersistentVolumeClaim for the Kode instance
func GetPVCName(kode *kodev1alpha1.Kode) string {
	return kode.Name + "-pvc"
}

// EncodeAndFillPath encodes a data structure, fills it into a CUE value at a specified path, and validates the result
// The ctx is the CUE context
// The value is the CUE value to fill
// The parsePath is used to parse the schema
// The valuePath is used to fill the data structure into the CUE value
// The schema is used to validate the data structure
// The data is the data structure to encode and fill
// The function returns the updated CUE value and nil if successful
// If an error occurs, the function returns the original CUE value and the error
func EncodeAndFillPath(ctx *cue.Context, value cue.Value, parsePath string, valuePath string, schema string, data interface{}) (cue.Value, error) {
	tempSchema := ctx.CompileString(schema).LookupPath(cue.ParsePath(parsePath))
	if tempSchema.Err() != nil {
		return value, fmt.Errorf("failed to parse path %s: %w", parsePath, tempSchema.Err())
	}

	valueAsCUE := ctx.Encode(data)
	if valueAsCUE.Err() != nil {
		return value, fmt.Errorf("failed to encode data: %w", valueAsCUE.Err())
	}

	unifiedValue, err := unifyAndValidate(tempSchema, valueAsCUE)
	if err != nil {
		return value, fmt.Errorf("failed to unify and validate: %w", err)
	}

	// Fill the unified value into the CUE value at the specified path
	value = value.FillPath(cue.ParsePath(valuePath), unifiedValue)
	if err := value.Err(); err != nil {
		return value, fmt.Errorf("failed to fill path %s: %w", valuePath, value.Err())
	}
	return value, nil
}

// unifyAndValidate unifies the schema and value, then validates the result
func unifyAndValidate(schema, value cue.Value) (cue.Value, error) {
	unifiedValue := schema.Unify(value)
	if err := unifiedValue.Validate(); err != nil {
		return cue.Value{}, fmt.Errorf("failed to validate unified value: %w", err)
	}
	return unifiedValue, nil
}
