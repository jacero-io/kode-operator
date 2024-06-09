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
	"reflect"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureService ensures that the Service exists for the Kode instance
func (r *KodeReconciler) ensureService(ctx context.Context, kode *kodev1alpha1.Kode, labels map[string]string, kodeTemplate *kodev1alpha1.KodeTemplate) error {
	log := r.Log.WithName("ensureService")

	log.Info("Ensuring Service exists", "Namespace", kode.Namespace, "Name", kode.Name)

	service, err := r.getOrCreateService(ctx, kode, labels, kodeTemplate)
	if err != nil {
		log.Error(err, "Failed to get or create Service", "Namespace", kode.Namespace, "Name", kode.Name)
		return err
	}

	if err := r.updateServiceIfNecessary(ctx, service); err != nil {
		log.Error(err, "Failed to update Service if necessary", "Namespace", service.Namespace, "Name", service.Name)
		return err
	}

	log.Info("Successfully ensured Service", "Namespace", kode.Namespace, "Name", kode.Name)

	return nil
}

// getOrCreateService gets or creates a Service for the Kode instance
func (r *KodeReconciler) getOrCreateService(ctx context.Context, kode *kodev1alpha1.Kode, labels map[string]string, kodeTemplate *kodev1alpha1.KodeTemplate) (*corev1.Service, error) {
	log := r.Log.WithName("getOrCreateService")
	service := r.constructService(kode, labels, kodeTemplate)

	if err := controllerutil.SetControllerReference(kode, service, r.Scheme); err != nil {
		return nil, err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Service", "Namespace", service.Namespace, "Name", service.Name)
			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create Service", "Namespace", service.Namespace, "Name", service.Name)
				return nil, err
			}
			log.Info("Service created", "Namespace", service.Namespace, "Name", service.Name)
		} else {
			return nil, err
		}
	}

	return service, nil
}

// constructService constructs a Service for the Kode instance
func (r *KodeReconciler) constructService(kode *kodev1alpha1.Kode, labels map[string]string, kodeTemplate *kodev1alpha1.KodeTemplate) *corev1.Service {
	log := r.Log.WithName("constructService")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       kodeTemplate.Spec.Port,
				TargetPort: intstr.FromInt(int(kodeTemplate.Spec.Port)),
			}},
		},
	}
	logServiceManifest(log, service)
	return service
}

// updateServiceIfNecessary updates the Service if the desired state is different from the existing state
func (r *KodeReconciler) updateServiceIfNecessary(ctx context.Context, service *corev1.Service) error {
	log := r.Log.WithName("updateServiceIfNecessary")
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		log.Info("Updating Service", "Namespace", found.Namespace, "Name", found.Name)
		return r.Update(ctx, found)
	}
	return nil
}
