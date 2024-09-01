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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureGateway ensures that the Gateway exists for the EntryPoint instance
func (r *EntryPointReconciler) ensureGateway(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint, config *common.EntryPointResourceConfig) (*gwapiv1.Gateway, error) {
	log := r.Log.WithName("GatewayEnsurer").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring Gateway")

	if entryPoint.Spec.GatewaySpec != nil && entryPoint.Spec.GatewaySpec.ExistingGatewayRef != nil {
		log.V(1).Info("Gateway already exists", "gateway", entryPoint.Spec.GatewaySpec.ExistingGatewayRef)
		gateway := &gwapiv1.Gateway{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: string(entryPoint.Spec.GatewaySpec.ExistingGatewayRef.Name), Namespace: entryPoint.Namespace}, gateway)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing Gateway: %w", err)
		}

		return gateway, nil
	}

	gateway := &gwapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.GatewayName,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
	}

	err := r.ResourceManager.CreateOrPatch(ctx, gateway, func() error {
		constructedGateway, err := r.constructGatewaySpec(entryPoint)
		if err != nil {
			return fmt.Errorf("failed to construct Gateway spec: %w", err)
		}

		gateway.Spec = constructedGateway.Spec
		gateway.ObjectMeta.Labels = constructedGateway.ObjectMeta.Labels

		return controllerutil.SetControllerReference(entryPoint, gateway, r.Scheme)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create or patch Gateway: %w", err)
	}

	return gateway, nil
}

// constructGateway constructs a Gateway for the EntryPoint instance
func (r *EntryPointReconciler) constructGatewaySpec(entryPoint *kodev1alpha2.EntryPoint) (*gwapiv1.Gateway, error) {
	log := r.Log.WithName("GatewayConstructor").WithValues("entrypoint", types.NamespacedName{Name: entryPoint.Name, Namespace: entryPoint.Namespace})

	gateway := &gwapiv1.Gateway{
		Spec: gwapiv1.GatewaySpec{
			GatewayClassName: *entryPoint.Spec.GatewaySpec.GatewayClassName,
			Listeners: []gwapiv1.Listener{
				{
					Name:     gwapiv1.SectionName("http"),
					Port:     80,
					Protocol: "HTTP",
				},
				{
					Name:     gwapiv1.SectionName("https"),
					Port:     443,
					Protocol: "HTTP",
					TLS:      &gwapiv1.GatewayTLSConfig{},
				},
			},
		},
	}

	log.V(1).Info("Constructed Gateway", "gateway", gateway)

	return gateway, nil
}

func (r *EntryPointReconciler) getGateway(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (*gwapiv1.Gateway, error) {
	// Implementation for fetching the Gateway associated with the EntryPoint
	// ...
	var err error
	gateway := &gwapiv1.Gateway{}

	return gateway, err
}

func (r *EntryPointReconciler) isGatewayReady(gateway *gwapiv1.Gateway) (bool, error) {
	// Implementation for checking if the Gateway is ready
	// ...
	var err error
	ready := false

	return ready, err
}

func (r *EntryPointReconciler) isSecurityPolicyReady(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (bool, error) {
	// Implementation for checking if the SecurityPolicy is ready
	// ...
	var err error
	ready := false

	return ready, err
}

func (r *EntryPointReconciler) isEnvoyPatchPolicyReady(ctx context.Context, entryPoint *kodev1alpha2.EntryPoint) (bool, error) {
	// Implementation for checking if the EnvoyPatchPolicy is ready
	// ...
	var err error
	ready := false

	return ready, err
}
