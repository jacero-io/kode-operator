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
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureGateway ensures that the Gateway exists for the EntryPoint instance
func (r *EntryPointReconciler) ensureGateway(ctx context.Context, config *common.EntryPointResourceConfig, entrypoint *kodev1alpha2.ClusterEntryPoint) error {
    log := r.Log.WithName("GatewayEnsurer").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

    ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
    defer cancel()

    log.V(1).Info("Ensuring Gateway")

    gateway := &gwapiv1.Gateway{
        ObjectMeta: metav1.ObjectMeta{
            Name:      config.CommonConfig.Name,
            Namespace: config.CommonConfig.Namespace,
            Labels:   config.CommonConfig.Labels,
        },
    }

    err := r.ResourceManager.CreateOrPatch(ctx, gateway, func() error {
        constructedGateway, err := r.constructGatewaySpec(config)
        if err != nil {
            return fmt.Errorf("failed to construct Gateway spec: %v", err)
        }

        gateway.Spec = constructedGateway.Spec
        gateway.ObjectMeta.Labels = constructedGateway.ObjectMeta.Labels

        return controllerutil.SetControllerReference(entrypoint, gateway, r.Scheme)
    })

    if err != nil {
        return fmt.Errorf("failed to create or patch Gateway: %v", err)
    }

    return nil
}

// constructGateway constructs a Gateway for the EntryPoint instance
func (r *EntryPointReconciler) constructGatewaySpec(config *common.EntryPointResourceConfig) (*gwapiv1.Gateway, error) {
    log := r.Log.WithName("GatewayConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

    gateway := &gwapiv1.Gateway{
        ObjectMeta: metav1.ObjectMeta{
            Name:      config.CommonConfig.Name,
            Namespace: config.CommonConfig.Namespace,
            Labels:   config.CommonConfig.Labels,
        },
        Spec: gwapiv1.GatewaySpec{
            GatewayClassName: config.GatewayClassName,
        },
    }

    log.V(1).Info("Constructed Gateway", "gateway", gateway)

    return gateway, nil
}