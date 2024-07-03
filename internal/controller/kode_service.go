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

package controller

import (
	"context"
	"fmt"

	"github.com/emil-jacero/kode-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureService ensures that the Service exists for the Kode instance
func (r *KodeReconciler) ensureService(ctx context.Context, config *common.KodeResourcesConfig) error {
	log := r.Log.WithName("ServiceEnsurer").WithValues("kode", client.ObjectKeyFromObject(&config.Kode))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.Info("Ensuring Service exists")

	service := r.constructService(config)
	if err := controllerutil.SetControllerReference(&config.Kode, service, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	err := r.ResourceManager.Ensure(ctx, service)
	if err != nil {
		log.Error(err, "Failed to ensure Service")
		return fmt.Errorf("failed to ensure Service: %w", err)
	}

	log.Info("Successfully ensured Service")
	return nil
}

// constructService constructs a Service for the Kode instance
func (r *KodeReconciler) constructService(config *common.KodeResourcesConfig) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Kode.Name,
			Namespace: config.Kode.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: config.Labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       config.ExternalServicePort,
				TargetPort: intstr.FromInt(int(config.ExternalServicePort)),
			}},
		},
	}
	return service
}
