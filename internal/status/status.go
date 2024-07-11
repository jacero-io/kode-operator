// internal/status/status.go

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

package status

import (
	"context"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusUpdater defines the interface for updating the status of a Kode and entry point resources.
type StatusUpdater interface {
	UpdateKodeStatus(ctx context.Context, config *common.KodeResourcesConfig, phase kodev1alpha1.KodePhase, conditions []metav1.Condition, lastError string, lastErrorTime *metav1.Time) error
	UpdateEntryPointsStatus(ctx context.Context, config *common.EntryPointResourceConfig, phase kodev1alpha1.EntryPointPhase, conditions []metav1.Condition, lastError string, lastErrorTime *metav1.Time) error
}
