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
	"strings"
	"time"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func ObjectKeyFromConfig(config *KodeResourcesConfig) types.NamespacedName {
	return types.NamespacedName{
		Name:      config.KodeName,
		Namespace: config.KodeNamespace,
	}
}

// returns the name of the PersistentVolumeClaim for the Kode instance
func GetPVCName(kode *kodev1alpha1.Kode) string {
	return kode.Name + "-pvc"
}

// returns the name of the Kode service
func GetServiceName(kode *kodev1alpha1.Kode) string {
	return kode.Name + "-svc"
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
