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

package entrypoint

import (
	"context"
	"fmt"

	// egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (r *EntryPointReconciler) checkGatewayResources(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, config *common.EntryPointResourceConfig) (bool, error) {
	log := r.Log.WithName("GatewayChecker").WithValues("entrypoint", client.ObjectKeyFromObject(entryPoint), "phase", entryPoint.Status.Phase)

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Checking Gateway")

	gateway := &gwapiv1.Gateway{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: config.GatewayName, Namespace: config.GatewayNamespace}, gateway)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Gateway not found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get existing Gateway: %w", err)
	}
	log.V(1).Info("Gateway found")

	log.V(1).Info("All resources are ready")
	return true, nil
}
