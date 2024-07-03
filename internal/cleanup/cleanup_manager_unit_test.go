//go:build unit

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

package cleanup

import (
	"context"
	"testing"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/kode/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockCleanupManager is a mock implementation of CleanupManager
type MockCleanupManager struct {
	mock.Mock
}

func (m *MockCleanupManager) Cleanup(ctx context.Context, kode *kodev1alpha1.Kode) error {
	args := m.Called(ctx, kode)
	return args.Error(0)
}

// TestCleanupManager_Cleanup tests the Cleanup method of MockCleanupManager
func TestCleanupManager_Cleanup(t *testing.T) {
	// Create a mock CleanupManager
	mockCleanupManager := new(MockCleanupManager)

	// Create a test Kode instance
	kode := &kodev1alpha1.Kode{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-kode",
			Namespace:  "default",
			Finalizers: []string{common.FinalizerName},
		},
		Spec: kodev1alpha1.KodeSpec{
			Storage: kodev1alpha1.KodeStorageSpec{},
		},
	}

	// Set up expectations on the mock CleanupManager
	mockCleanupManager.On("Cleanup", mock.Anything, kode).Return(nil)

	// Call the Cleanup method
	err := mockCleanupManager.Cleanup(context.Background(), kode)

	// Assert that no error occurred
	assert.NoError(t, err)

	// Assert that the Cleanup method was called with the correct arguments
	mockCleanupManager.AssertCalled(t, "Cleanup", mock.Anything, kode)
}

func TestCleanupManager_Cleanup_WithStorage(t *testing.T) {
	// Create a mock CleanupManager
	mockCleanupManager := new(MockCleanupManager)

	// Create a test Kode instance with storage
	kode := &kodev1alpha1.Kode{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-kode",
			Namespace:  "default",
			Finalizers: []string{common.FinalizerName},
		},
		Spec: kodev1alpha1.KodeSpec{
			Storage: kodev1alpha1.KodeStorageSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}

	// Set up expectations on the mock CleanupManager
	mockCleanupManager.On("Cleanup", mock.Anything, kode).Return(nil)

	// Call the Cleanup method
	err := mockCleanupManager.Cleanup(context.Background(), kode)

	// Assert that no error occurred
	assert.NoError(t, err)

	// Assert that the Cleanup method was called with the correct arguments
	mockCleanupManager.AssertCalled(t, "Cleanup", mock.Anything, kode)
}
