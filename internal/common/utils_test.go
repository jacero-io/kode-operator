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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

var (
	testScheme *runtime.Scheme
)

func init() {
	testScheme = runtime.NewScheme()
	// Register your custom types
	if err := kodev1alpha2.AddToScheme(testScheme); err != nil {
		panic(fmt.Sprintf("Failed to add kodev1alpha2 to scheme: %v", err))
	}
	// Register core types if needed
	if err := corev1.AddToScheme(testScheme); err != nil {
		panic(fmt.Sprintf("Failed to add corev1 to scheme: %v", err))
	}
}

func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build()
}

func TestGetLatestKode(t *testing.T) {
	kode := &kodev1alpha2.Kode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kode",
			Namespace: "default",
		},
	}

	fakeClient := newFakeClient(kode)
	result, err := GetLatestKode(context.TODO(), fakeClient, "test-kode", "default")
	assert.NoError(t, err)
	assert.Equal(t, kode, result)
}

func TestGetLatestKode_NotFound(t *testing.T) {
	fakeClient := newFakeClient() // No objects
	result, err := GetLatestKode(context.TODO(), fakeClient, "nonexistent", "default")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetLatestEntryPoint(t *testing.T) {
	entryPoint := &kodev1alpha2.EntryPoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-entrypoint",
			Namespace: "default",
		},
	}

	fakeClient := newFakeClient(entryPoint)
	result, err := GetLatestEntryPoint(context.TODO(), fakeClient, "test-entrypoint", "default")
	assert.NoError(t, err)
	assert.Equal(t, entryPoint, result)
}

func TestGetLatestEntryPoint_NotFound(t *testing.T) {
	fakeClient := newFakeClient()
	result, err := GetLatestEntryPoint(context.TODO(), fakeClient, "nonexistent", "default")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestObjectKeyFromConfig(t *testing.T) {
	config := CommonConfig{
		Name:      "test-name",
		Namespace: "test-namespace",
	}
	expected := types.NamespacedName{
		Name:      "test-name",
		Namespace: "test-namespace",
	}
	result := ObjectKeyFromConfig(config)
	assert.Equal(t, expected, result)
}

func TestMaskString(t *testing.T) {
	input := "password"
	expected := "********"

	result := MaskString(input)
	assert.Equal(t, expected, result)
}

func TestMaskSecretData(t *testing.T) {
	secret := &corev1.Secret{}
	secret.Data = map[string][]byte{
		"username": []byte("user"),
		"password": []byte("secretpass"),
		"token":    []byte("sometoken"),
		"secret":   []byte("supersecret"),
	}

	expected := map[string]string{
		"username": "user",
		"password": "**********",
		"token":    "sometoken",
		"secret":   "***********",
	}

	result := MaskSecretData(secret)
	assert.Equal(t, expected, result)
}

func TestMaskEnvVars(t *testing.T) {
	envs := []corev1.EnvVar{
		{Name: "USERNAME", Value: "user"},
		{Name: "PASSWORD", Value: "secretpass"},
		{Name: "TOKEN", Value: "sometoken"},
		{Name: "SECRET", Value: "supersecret"},
	}

	expected := []corev1.EnvVar{
		{Name: "USERNAME", Value: "user"},
		{Name: "PASSWORD", Value: "**********"},
		{Name: "TOKEN", Value: "sometoken"},
		{Name: "SECRET", Value: "***********"},
	}

	result := MaskEnvVars(envs)
	assert.Equal(t, expected, result)
}

func TestMaskSpec(t *testing.T) {
	spec := corev1.Container{
		Name: "test-container",
		Env: []corev1.EnvVar{
			{Name: "USERNAME", Value: "user"},
			{Name: "PASSWORD", Value: "secretpass"},
			{Name: "TOKEN", Value: "sometoken"},
			{Name: "SECRET", Value: "supersecret"},
		},
	}

	expected := corev1.Container{
		Name: "test-container",
		Env: []corev1.EnvVar{
			{Name: "USERNAME", Value: "user"},
			{Name: "PASSWORD", Value: "**********"},
			{Name: "TOKEN", Value: "sometoken"},
			{Name: "SECRET", Value: "***********"},
		},
	}

	result := MaskSpec(spec)
	assert.Equal(t, expected, result)
}

func TestGetUsernameAndPasswordFromSecret(t *testing.T) {
	secret := &corev1.Secret{}
	secret.Data = map[string][]byte{
		"username": []byte("user"),
		"password": []byte("pass"),
	}

	username, password, err := GetUsernameAndPasswordFromSecret(secret)
	assert.NoError(t, err)
	assert.Equal(t, "user", username)
	assert.Equal(t, "pass", password)
}

func TestGetUsernameAndPasswordFromSecret_NoUsername(t *testing.T) {
	secret := &corev1.Secret{}
	secret.Data = map[string][]byte{
		"password": []byte("pass"),
	}

	username, password, err := GetUsernameAndPasswordFromSecret(secret)
	assert.Error(t, err)
	assert.Equal(t, "", username)
	assert.Equal(t, "", password)
}

func TestContextWithTimeout(t *testing.T) {
	parentCtx := context.Background()
	timeoutSeconds := int64(5)

	ctx, cancel := ContextWithTimeout(parentCtx, timeoutSeconds)
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Second)
}

func TestBoolPtr(t *testing.T) {
	value := true
	ptr := BoolPtr(value)
	assert.Equal(t, &value, ptr)
	assert.Equal(t, value, *ptr)
}

func TestInt32Ptr(t *testing.T) {
	var value int32 = 42
	ptr := Int32Ptr(value)
	assert.Equal(t, &value, ptr)
	assert.Equal(t, value, *ptr)
}

func TestInt64Ptr(t *testing.T) {
	var value int64 = 64
	ptr := Int64Ptr(value)
	assert.Equal(t, &value, ptr)
	assert.Equal(t, value, *ptr)
}

func TestStringPtr(t *testing.T) {
	value := "test"
	ptr := StringPtr(value)
	assert.Equal(t, &value, ptr)
	assert.Equal(t, value, *ptr)
}

func TestMergeLabels(t *testing.T) {
	labels1 := map[string]string{"app": "test", "version": "1"}
	labels2 := map[string]string{"env": "prod", "version": "2"}
	labels3 := map[string]string{"team": "dev"}

	result := MergeLabels(labels1, labels2, labels3)

	expected := map[string]string{
		"app":     "test",
		"version": "2", // labels2 overwrites labels1
		"env":     "prod",
		"team":    "dev",
	}

	assert.Equal(t, expected, result)
}

func TestAddTypeInformationToObject(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	obj := &corev1.Pod{}

	err := AddTypeInformationToObject(scheme, obj)
	assert.NoError(t, err)

	gvk := obj.GetObjectKind().GroupVersionKind()
	expectedGVK := corev1.SchemeGroupVersion.WithKind("Pod")
	assert.Equal(t, expectedGVK, gvk)
}

func TestAddTypeInformationToObject_NilObject(t *testing.T) {
	scheme := runtime.NewScheme()
	err := AddTypeInformationToObject(scheme, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input object is nil")
}

func TestAddTypeInformationToObject_UnknownObject(t *testing.T) {
	scheme := runtime.NewScheme()
	obj := &UnknownType{} // Not registered in scheme

	err := AddTypeInformationToObject(scheme, obj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get object kinds")
}

// UnknownType is a test struct implementing runtime.Object
type UnknownType struct{}

func (u *UnknownType) GetObjectKind() schema.ObjectKind {
	return &metav1.TypeMeta{}
}

func (u *UnknownType) DeepCopyObject() runtime.Object {
	return u
}
