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

package kode

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

func (r *KodeReconciler) handleTemplateFetchError(ctx context.Context, kode *kodev1alpha2.Kode, err error) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{
		{
			Type:    "TemplateFetchFailed",
			Status:  metav1.ConditionTrue,
			Reason:  "TemplateNotFound",
			Message: fmt.Sprintf("Failed to fetch template: %v", err),
		},
	}, nil, "", nil); err != nil {
		log.Error(err, "Failed to update status for template fetch failure")
	}
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
}

func (r *KodeReconciler) handleResourceCheckError(ctx context.Context, kode *kodev1alpha2.Kode, err error) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{
		{
			Type:    "ResourceCheckFailed",
			Status:  metav1.ConditionTrue,
			Reason:  "FailedToCheckResources",
			Message: fmt.Sprintf("Failed to check resource readiness: %v", err),
		},
	}, nil, "", nil); err != nil {
		log.Error(err, "Failed to update status for resource check failure")
	}
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
}

func (r *KodeReconciler) handleResourceCreationError(ctx context.Context, kode *kodev1alpha2.Kode, err error) (ctrl.Result, error) {
	log := r.Log.WithValues("kode", client.ObjectKeyFromObject(kode))
	if err := r.StatusUpdater.UpdateStatusKode(ctx, kode, kodev1alpha2.KodePhaseFailed, []metav1.Condition{
		{
			Type:    common.ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  "FailedToCreateResources",
			Message: fmt.Sprintf("Failed to create pod resources: %v", err),
		},
		{
			Type:    common.ConditionTypeAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "FailedToCreateResources",
			Message: fmt.Sprintf("Failed to create pod resources: %v", err),
		},
		{
			Type:    common.ConditionTypeProgressing,
			Status:  metav1.ConditionFalse,
			Reason:  "FailedToCreateResources",
			Message: fmt.Sprintf("Failed to create pod resources: %v", err),
		},
		{
			Type:    common.ConditionTypeError,
			Status:  metav1.ConditionTrue,
			Reason:  "FailedToCreateResources",
			Message: fmt.Sprintf("Failed to create pod resources: %v", err),
		},
	}, nil, "", nil); err != nil {
		log.Error(err, "Failed to update status for resource creation failure")
	}
	return r.transitionTo(ctx, kode, kodev1alpha2.KodePhaseFailed)
}
