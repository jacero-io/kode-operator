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

package common

import (
	"context"
	"fmt"
	"time"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCurrentTime() metav1.Time {
	return metav1.NewTime(time.Now())
}

func GetLatestKode(ctx context.Context, client client.Client, name string, namespace string) (*kodev1alpha2.Kode, error) {
	kode := &kodev1alpha2.Kode{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, kode)
	return kode, err
}

func GetLatestEntryPoint(ctx context.Context, client client.Client, name string, namespace string) (*kodev1alpha2.EntryPoint, error) {
	entryPoint := &kodev1alpha2.EntryPoint{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, entryPoint)
	return entryPoint, err
}

func ObjectKeyFromConfig(config CommonConfig) types.NamespacedName {
	return types.NamespacedName{
		Name:      config.Name,
		Namespace: config.Namespace,
	}
}

// masks sensitive string values
func MaskString(s string) string {
	masked := ""
	for range s {
		masked += "*"
	}
	return masked
}

// masks sensitive values in a secret
func MaskSecretData(secret *corev1.Secret) map[string]string {
	maskedData := make(map[string]string)

	secret = secret.DeepCopy()
	for k, v := range secret.Data {
		if k == "password" || k == "secret" {
			maskedData[k] = MaskString(string(v))
		} else {
			maskedData[k] = string(v)
		}
	}
	return maskedData
}

// masks sensitive environment variables
func MaskEnvVars(envs []corev1.EnvVar) []corev1.EnvVar {
	maskedEnvs := make([]corev1.EnvVar, len(envs))
	for i, env := range envs {
		if env.Name == "PASSWORD" || env.Name == "SECRET" {
			env.Value = MaskString(env.Value)
		}
		maskedEnvs[i] = env
	}
	return maskedEnvs
}

// masks sensitive values in a container spec
func MaskSpec(spec corev1.Container) corev1.Container {
	spec.Env = MaskEnvVars(spec.Env)
	return spec
}

func GetUsernameAndPasswordFromSecret(secret *corev1.Secret) (string, string, error) {
	username := string(secret.Data["username"])
	if username == "" {
		return "", "", fmt.Errorf("username not found in secret")
	}
	// Password can be empty
	password := string(secret.Data["password"])
	return username, password, nil
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

// addTypeInformationToObject adds TypeMeta information to a runtime.Object based upon the loaded scheme.Scheme
// taken from: https://github.com/kubernetes/kubernetes/issues/3030#issuecomment-700099699
func AddTypeInformationToObject(scheme *runtime.Scheme, obj runtime.Object) error {
	// Check if obj is nil
	if obj == nil {
		return fmt.Errorf("input object is nil")
	}

	// Check if the object already has type information
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		// If type information is missing, try to add it
		gvks, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			return fmt.Errorf("failed to get object kinds: %w", err)
		}
		if len(gvks) > 0 {
			gvk := gvks[0]
			obj.GetObjectKind().SetGroupVersionKind(gvk)
		} else {
			return fmt.Errorf("no valid GroupVersionKind found for object of type %T", obj)
		}
	}

	return nil
}
