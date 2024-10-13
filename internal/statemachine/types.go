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

package statemachine

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/template"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

// StateManagedResource defines the interface for resources managed by the state machine
type StateManagedResource interface {
	GetName() string
	GetNamespace() string
	GetPhase() kodev1alpha2.Phase
	SetPhase(phase kodev1alpha2.Phase)
	UpdateStatus(ctx context.Context, c client.Client) error
	SetCondition(conditionType constant.ConditionType, status metav1.ConditionStatus, reason, message string)
	GetCondition(conditionType constant.ConditionType) *metav1.Condition
	DeleteCondition(conditionType constant.ConditionType)
}

// ReconcilerInterface defines the common interface for reconcilers
type ReconcilerInterface interface {
	GetClient() client.Client
	GetScheme() *runtime.Scheme
	GetLog() logr.Logger
	GetEventRecorder() event.EventManager
	GetResourceManager() resource.ResourceManager
	GetTemplateManager() template.TemplateManager
	GetCleanupManager() cleanup.CleanupManager
	GetIsTestEnvironment() bool
	GetReconcileInterval() time.Duration
	GetLongReconcileInterval() time.Duration
}

// StateHandler defines the function signature for state handlers
type StateHandler func(ctx context.Context, r ReconcilerInterface, resource StateManagedResource) (kodev1alpha2.Phase, ctrl.Result, error)
