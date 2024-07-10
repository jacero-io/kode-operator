// internal/status/status_updater.go

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

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
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

func (u *defaultStatusUpdater) UpdateStatus(ctx context.Context, phase string, conditions []metav1.Condition, lastError string, lastErrorTime *metav1.Time) error {
	// Not implemented
	return nil
}

func (u *defaultStatusUpdater) UpdateKodeStatus(ctx context.Context,
	config *common.KodeResourcesConfig,
	phase kodev1alpha1.KodePhase,
	conditions []metav1.Condition,
	lastError string,
	lastErrorTime *metav1.Time) error {
	log := u.log.WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of Kode
		var latestKode kodev1alpha1.Kode
		if err := u.client.Get(ctx, types.NamespacedName{Name: config.Kode.Name, Namespace: config.Kode.Namespace}, &latestKode); err != nil {
			log.Error(err, "Failed to get latest version of Kode")
			return err
		}

		// Update the status
		latestKode.Status.Phase = phase
		latestKode.Status.Conditions = conditions
		latestKode.Status.LastError = lastError
		latestKode.Status.LastErrorTime = lastErrorTime

		// Try to update
		if err := u.client.Status().Update(ctx, &latestKode); err != nil {
			log.Error(err, "Failed to update Kode status")
			return err
		}

		// Update was successful, update the config's Kode with the latest version
		config.Kode = latestKode

		return nil
	})
}

func (u *defaultStatusUpdater) UpdateEntryPointsStatus(ctx context.Context, config *common.EntryPointResourceConfig, phase kodev1alpha1.EntryPointPhase, conditions []metav1.Condition, lastError string, lastErrorTime *metav1.Time) error {
	log := u.log.WithValues("entryPoint", client.ObjectKeyFromObject(&config.EntryPoint))

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of EntryPoint
		var latestEntryPoint kodev1alpha1.EntryPoint
		if err := u.client.Get(ctx, types.NamespacedName{Name: config.EntryPoint.Name, Namespace: config.EntryPoint.Namespace}, &latestEntryPoint); err != nil {
			log.Error(err, "Failed to get latest version of EntryPoint")
			return err
		}

		// Update the status
		latestEntryPoint.Status.Phase = phase
		latestEntryPoint.Status.Conditions = conditions
		latestEntryPoint.Status.LastError = lastError
		latestEntryPoint.Status.LastErrorTime = lastErrorTime

		// Try to update
		if err := u.client.Status().Update(ctx, &latestEntryPoint); err != nil {
			log.Error(err, "Failed to update EntryPoint status")
			return err
		}

		// Update was successful, update the config's EntryPoint with the latest version
		config.EntryPoint = latestEntryPoint

		return nil
	})
}
