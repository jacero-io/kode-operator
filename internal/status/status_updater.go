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

package status

import (
	"context"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultStatusUpdater struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultStatusUpdater(client client.Client, log logr.Logger) StatusUpdater {
	return &defaultStatusUpdater{
		client: client,
		log:    log,
	}
}

func (u *defaultStatusUpdater) UpdateStatus(ctx context.Context, kode *kodev1alpha1.Kode, phase kodev1alpha1.KodePhase, conditions []metav1.Condition, lastError string, lastErrorTime *metav1.Time) error {
	log := u.log.WithValues("kode", client.ObjectKeyFromObject(kode))

	kode.Status.Phase = phase
	kode.Status.Conditions = conditions
	kode.Status.LastError = lastError
	kode.Status.LastErrorTime = lastErrorTime

	if err := u.client.Status().Update(ctx, kode); err != nil {
		log.Error(err, "Failed to update Kode status")
		return err
	}

	return nil
}
