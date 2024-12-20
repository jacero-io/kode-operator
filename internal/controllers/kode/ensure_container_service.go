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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/resourcev1"
	"github.com/jacero-io/kode-operator/internal/statemachine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureService ensures that the Service exists for the Kode instance
func ensureService(ctx context.Context, r statemachine.ReconcilerInterface, resource resourcev1.ResourceManager, kode *kodev1alpha2.Kode, config *common.KodeResourceConfig) error {
	log := r.GetLog().WithName("ServiceEnsurer").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring Service")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.GetServiceName(),
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
	}

	_, err := resource.CreateOrPatch(ctx, service, func() error {
		constructedService, err := constructServiceSpec(r, config)
		if err != nil {
			return fmt.Errorf("failed to construct Service spec: %v", err)
		}

		service.Spec = constructedService.Spec
		service.ObjectMeta.Labels = constructedService.ObjectMeta.Labels

		return controllerutil.SetControllerReference(kode, service, r.GetScheme())
	})

	if err != nil {
		return fmt.Errorf("failed to create or patch Service: %v", err)
	}

	return nil
}

// constructService constructs a Service for the Kode instance
func constructServiceSpec(r statemachine.ReconcilerInterface, config *common.KodeResourceConfig) (*corev1.Service, error) {
	log := r.GetLog().WithName("ServiceConstructor").WithValues("kode", common.ObjectKeyFromConfig(config.CommonConfig))

	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: config.CommonConfig.Labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(config.Port),
				TargetPort: intstr.FromInt(int(config.Port)),
			}},
		},
	}

	log.V(1).Info("Service object constructed", "Service", service, "Spec", service.Spec)

	return service, nil
}
