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

package events

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventReason defines the reason for an event
type EventReason string

// EventType defines the type of an event
type EventType string

const (
	// Event types
	EventTypeNormal  EventType = "Normal"
	EventTypeWarning EventType = "Warning"

	// Event reasons
	ReasonCreated    EventReason = "Created"
	ReasonUpdated    EventReason = "Updated"
	ReasonFailed     EventReason = "Failed"
	ReasonReconciled EventReason = "Reconciled"
)

// EventManager defines the interface for managing events
type EventManager interface {
	Record(ctx context.Context, object runtime.Object, eventtype EventType, reason EventReason, message string) error
}

// defaultEventManager implements EventManager
type defaultEventManager struct {
	Client        client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// NewEventManager creates a new EventManager
func NewEventManager(client client.Client, log logr.Logger, scheme *runtime.Scheme, eventRecorder record.EventRecorder) EventManager {
	return &defaultEventManager{
		Client:        client,
		Log:           log,
		Scheme:        scheme,
		EventRecorder: eventRecorder,
	}
}

// Record creates and records an event
func (e *defaultEventManager) Record(ctx context.Context, object runtime.Object, eventtype EventType, reason EventReason, message string) error {
	metaObj, ok := object.(metav1.Object)
	if !ok {
		return fmt.Errorf("object does not implement metav1.Object")
	}

	log := e.Log.WithValues(
		"kind", object.GetObjectKind().GroupVersionKind().Kind,
		"name", metaObj.GetName(),
		"namespace", metaObj.GetNamespace(),
	)

	e.EventRecorder.Event(object, string(eventtype), string(reason), message)

	log.V(1).Info("Successfully created event",
		"object", object.GetObjectKind().GroupVersionKind().Kind,
		"namespace", metaObj.GetNamespace(),
		"name", metaObj.GetName(),
		"type", eventtype,
		"reason", reason)

	return nil
}
