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
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

type StateMachine struct {
	Handlers map[kodev1alpha2.Phase]StateHandler
	Client   client.Client
	Log      logr.Logger
}

func NewStateMachine(c client.Client, log logr.Logger) *StateMachine {
	return &StateMachine{
		Handlers: make(map[kodev1alpha2.Phase]StateHandler),
		Client:   c,
		Log:      log,
	}
}

func (sm *StateMachine) RegisterHandler(phase kodev1alpha2.Phase, handler StateHandler) {
	sm.Handlers[phase] = handler
}

func (sm *StateMachine) HandleState(ctx context.Context, r ReconcilerInterface, resource StateManagedResource) (ctrl.Result, error) {
	currentPhase := resource.GetPhase()
	handler, exists := sm.Handlers[currentPhase]
	if !exists {
		return ctrl.Result{}, fmt.Errorf("no handler for phase %s", currentPhase)
	}

	nextPhase, result, err := handler(ctx, r, resource)
	if err != nil {
		sm.Log.Error(err, "Error handling phase", "current_phase", currentPhase)
		return ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, err
	}

	if nextPhase != currentPhase {
		resource.SetPhase(nextPhase)
		if err := resource.UpdateStatus(ctx, sm.Client); err != nil {
			sm.Log.Error(err, "Unable to update resource status")
			return ctrl.Result{RequeueAfter: r.GetReconcileInterval()}, err
		}
		sm.Log.Info("Updated resource phase", "resource", fmt.Sprintf("%T", resource), "new_phase", nextPhase)
	}

	return result, nil
}
