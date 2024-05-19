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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Kode Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					Image:       "lscr.io/linuxserver/code-server:latest",
					ServicePort: 8443,
					Password:    "password",
				},
			}
			err := k8sClient.Create(ctx, kode)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("deleting the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
		})

		It("should create a Deployment and Service for the resource", func() {
			By("reconciling the created resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the Deployment has been created")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, deployment)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("lscr.io/linuxserver/code-server:latest"))

			By("checking if the Service has been created")
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, service)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8443)))
		})

		It("should update the Deployment when the Kode resource is updated", func() {
			By("reconciling the created resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("updating the Kode resource")
			kode := &kodev1alpha1.Kode{}
			err = k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			kode.Spec.Image = "lscr.io/linuxserver/code-server:latest"
			Expect(k8sClient.Update(ctx, kode)).To(Succeed())

			By("checking if the Deployment has been updated")
			deployment := &appsv1.Deployment{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, deployment)
				return deployment.Spec.Template.Spec.Containers[0].Image
			}, time.Second*5, time.Millisecond*500).Should(Equal("lscr.io/linuxserver/code-server:latest"))
		})

		It("should delete the Deployment and Service when the Kode resource is deleted", func() {
			By("reconciling the created resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("deleting the Kode resource")
			kode := &kodev1alpha1.Kode{}
			err = k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())

			By("checking if the Deployment has been deleted")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, deployment)
				return errors.IsNotFound(err)
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())

			By("checking if the Service has been deleted")
			service := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, service)
				return errors.IsNotFound(err)
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})
	})
})
