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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
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

		It("should create a StatefulSet and Service for the resource", func() {
			By("reconciling the created resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the StatefulSet has been created")
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, statefulSet)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(statefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal("lscr.io/linuxserver/code-server:latest"))

			By("checking if the Service has been created")
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, service)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8443)))
		})

		It("should update the StatefulSet when the Kode resource is updated", func() {
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

			By("checking if the StatefulSet has been updated")
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, statefulSet)
				return statefulSet.Spec.Template.Spec.Containers[0].Image
			}, time.Second*5, time.Millisecond*500).Should(Equal("lscr.io/linuxserver/code-server:latest"))
		})

		It("should create a PVC for the resource if specified", func() {
			By("updating the Kode resource to specify storage")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			kode.Spec.Storage = &kodev1alpha1.StorageSpec{
				Name: "test-pvc",
				Size: "1Gi",
			}
			Expect(k8sClient.Update(ctx, kode)).To(Succeed())

			By("reconciling the updated resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the PVC has been created")
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-pvc", Namespace: resourceNamespace}, pvc)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
		})

		It("should update the PVC when the Kode resource is updated", func() {
			By("updating the Kode resource to specify storage")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			kode.Spec.Storage = &kodev1alpha1.StorageSpec{
				Name: "test-pvc",
				Size: "1Gi",
			}
			Expect(k8sClient.Update(ctx, kode)).To(Succeed())

			By("reconciling the updated resource")
			controllerReconciler := &KodeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the PVC has been created")
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-pvc", Namespace: resourceNamespace}, pvc)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))

			By("updating the Kode resource to change the PVC size")
			kode.Spec.Storage.Size = "2Gi"
			Expect(k8sClient.Update(ctx, kode)).To(Succeed())

			By("reconciling the updated resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the PVC has been updated")
			Eventually(func() error {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pvc", Namespace: resourceNamespace}, pvc)
				if pvc.Spec.Resources.Requests[corev1.ResourceStorage] != resource.MustParse("2Gi") {
					return fmt.Errorf("PVC size not updated")
				}
				return nil
			}, time.Second*5, time.Millisecond*500).Should(Succeed())
		})
	})
})
