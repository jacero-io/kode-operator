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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCurrentTime() metav1.Time {
	return metav1.NewTime(time.Now())
}

func GetLatestKode(ctx context.Context, client client.Client, name, namespace string) (*kodev1alpha2.Kode, error) {
	kode := &kodev1alpha2.Kode{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, kode)
	return kode, err
}

func GetLatestEntryPoint(ctx context.Context, client client.Client, name string) (*kodev1alpha2.ClusterEntryPoint, error) {
	entryPoint := &kodev1alpha2.ClusterEntryPoint{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: metav1.NamespaceAll}, entryPoint)
	return entryPoint, err
}

func ObjectKeyFromConfig(config CommonConfig) types.NamespacedName {
	return types.NamespacedName{
		Name:      config.Name,
		Namespace: config.Namespace,
	}
}

// masks sensitive string values
func maskSensitiveValue(s string) string {
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
			maskedData[k] = maskSensitiveValue(string(v))
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
			env.Value = maskSensitiveValue(env.Value)
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
	password := string(secret.Data["password"])
	if password == "" {
		password = ""
		err := fmt.Errorf("password not found in secret")
		return username, password, err
	}
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
func AddTypeInformationToObject(obj runtime.Object) error {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range gvks {
		if len(gvk.Kind) == 0 {
			continue
		}
		if len(gvk.Version) == 0 || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}

	return nil
}
