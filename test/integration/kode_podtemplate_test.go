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

package integration

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jacero-io/kode-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

var _ = Describe("Kode Controller PodTemplate Integration", Ordered, func() {

	DescribeTable("Kode resource creation",
		func(templateName string, templateType string, expectedContainerCount int, exposePort int32) {

			kodeName := fmt.Sprintf("%s-%s", "kode-standard", templateName)
			username := "abc"
			password := "123"
			statefulSetName := kodeName
			storageClassName := "standard"

			credentials := &kodev1alpha2.CredentialsSpec{
				Username: username,
				Password: password,
			}

			storage := &kodev1alpha2.KodeStorageSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: &corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(storageSize),
					},
				},
				StorageClassName: &storageClassName,
			}

			kode := createKode(kodeName, namespace.Name, templateName, credentials, storage)

			// Create Kode resource
			Expect(k8sClient.Create(ctx, kode)).To(Succeed())

			// Check StatefulSet
			statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
			createdStatefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
			}, timeout, interval).Should(Succeed())

			Expect(createdStatefulSet.Name).To(Equal(kodeName))
			Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(expectedContainerCount))

			if templateType == "code-server" {
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(podTemplateImageCodeServer))
				// Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "DEFAULT_WORKSPACE", Value: "/config/workspace"}))
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "USERNAME", Value: "abc"}))
			} else if templateType == "webtop" {
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(podTemplateImageWebtop))
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "CUSTOM_USER", Value: "abc"}))
			}

			// Check PersistentVolumeClaim
			pvcName := fmt.Sprintf("%s-pvc", kodeName)
			pvcLookupKey := types.NamespacedName{Name: pvcName, Namespace: namespace.Name}
			createdPVC := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, pvcLookupKey, createdPVC)
			}, timeout, interval).Should(Succeed())

			Expect(createdPVC.Name).To(Equal(pvcName))
			Expect(createdPVC.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
			Expect(createdPVC.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse(storageSize)))
			Expect(createdPVC.Spec.StorageClassName).NotTo(BeNil())
			Expect(*createdPVC.Spec.StorageClassName).To(Equal(storageClassName))

			// Check if the PVC is mounted in the StatefulSet
			volumeMount := corev1.VolumeMount{
				Name:      common.KodeVolumeStorageName,
				MountPath: "/config",
			}
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(volumeMount))

			volume := corev1.Volume{
				Name: common.KodeVolumeStorageName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}
			Expect(createdStatefulSet.Spec.Template.Spec.Volumes).To(ContainElement(volume))

			// Check Service
			serviceName := fmt.Sprintf("%s-svc", kodeName)
			serviceLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace.Name}
			createdService := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())

			Expect(createdService.Name).To(Equal(serviceName))
			Expect(createdService.Spec.Ports).To(HaveLen(1))
			Expect(createdService.Spec.Ports[0].Port).To(Equal(exposePort))

			// Cleanup
			By("Deleting the Kode resource")
			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())

			By("Waiting for Kode resource to be deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue(), "Failed to delete Kode resource")

			By("Ensuring all related resources are cleaned up")
			Eventually(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
				defer cancel()

				// Check if StatefulSet is deleted
				err := waitForResourceDeletion(ctx, mockK8sClient, &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: namespace.Name},
				}, time.Minute*5)
				if err != nil {
					return fmt.Errorf("StatefulSet deletion error: %v", err)
				}

				// Check if Service is deleted
				err = waitForResourceDeletion(ctx, mockK8sClient, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: namespace.Name},
				}, time.Minute*5)
				if err != nil {
					return fmt.Errorf("Service deletion error: %v", err)
				}

				// Check if PVC is deleted
				err = waitForResourceDeletion(ctx, mockK8sClient, &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: pvcName, Namespace: namespace.Name},
				}, time.Minute*5)
				if err != nil {
					return fmt.Errorf("PersistentVolumeClaim deletion error: %v", err)
				}

				return nil
			}, time.Minute*5, time.Second).Should(Succeed(), "Failed to clean up all resources")
		},
		Entry("code-server", podTemplateNameCodeServer, "code-server", 1, int32(8000)),
		Entry("webtop", podTemplateNameWebtop, "webtop", 1, int32(8000)),
	)

	It("should not create a PersistentVolumeClaim when storage is not specified", func() {
		username := "abc"
		password := "123"
		kodeName := "kode-no-storage"

		credentials := &kodev1alpha2.CredentialsSpec{
			Username: username,
			Password: password,
		}

		// Create Kode resource
		kode := createKode(kodeName, namespace.Name, podTemplateNameCodeServer, credentials, nil) // No storage
		Expect(k8sClient.Create(ctx, kode)).To(Succeed())

		// Check StatefulSet
		statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
		createdStatefulSet := &appsv1.StatefulSet{}
		Eventually(func() error {
			return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
		}, timeout, interval).Should(Succeed())

		// Ensure no PVC-related volume or volumeMount exists
		Expect(createdStatefulSet.Spec.Template.Spec.Volumes).NotTo(ContainElement(HaveField("Name", common.KodeVolumeStorageName)))
		Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts).NotTo(ContainElement(HaveField("Name", common.KodeVolumeStorageName)))

		// Check that PVC doesn't exist
		pvcName := fmt.Sprintf("%s-pvc", kodeName)
		pvcLookupKey := types.NamespacedName{Name: pvcName, Namespace: namespace.Name}
		Eventually(func() error {
			return k8sClient.Get(ctx, pvcLookupKey, &corev1.PersistentVolumeClaim{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

		// Cleanup
		Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	It("should create the secret with default username when credentials are not specified", func() {
		kodeName := "kode-default-credentials"

		// Create Kode resource
		kode := createKode(kodeName, namespace.Name, podTemplateNameCodeServer, nil, nil) // No credentials
		Expect(k8sClient.Create(ctx, kode)).To(Succeed())

		// Check Secret
		secretName := kode.GetSecretName()
		secretLookupKey := types.NamespacedName{Name: secretName, Namespace: namespace.Name}
		createdSecret := &corev1.Secret{}
		Eventually(func() error {
			return k8sClient.Get(ctx, secretLookupKey, createdSecret)
		}, timeout, interval).Should(Succeed())

		Expect(createdSecret.Name).To(Equal(secretName))
		Expect(createdSecret.Data).To(HaveKeyWithValue("username", []byte(common.Username)))

		// Cleanup
		Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})

var _ = Describe("Kode Controller Update Integration", func() {
	Context("When updating credentials", func() {
		It("Should update the Kode resource and Secret", func() {
			kodeName := "kode-update-test"
			username := "abc"
			password := "123"

			credentials := &kodev1alpha2.CredentialsSpec{
				Username: username,
				Password: password,
			}

			// Create initial Kode resource
			initialKode := createKode(kodeName, namespace.Name, podTemplateNameCodeServer, credentials, nil)
			Expect(k8sClient.Create(ctx, initialKode)).To(Succeed())

			kodeNamespacedName := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}

			// Wait for initial resources to be created
			Eventually(func() error {
				return k8sClient.Get(ctx, kodeNamespacedName, &appsv1.StatefulSet{})
			}, timeout, interval).Should(Succeed())

			// Update Kode resource
			updatedKode := &kodev1alpha2.Kode{}
			Expect(k8sClient.Get(ctx, kodeNamespacedName, updatedKode)).To(Succeed())

			updatedUsername := "updated-user"
			updatedPassword := "updated-pass"

			updatedKode.Spec.Credentials.Username = updatedUsername
			updatedKode.Spec.Credentials.Password = updatedPassword

			Expect(k8sClient.Update(ctx, updatedKode)).To(Succeed())

			// Verify Kode resource is updated
			refreshedKode := &kodev1alpha2.Kode{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, kodeNamespacedName, refreshedKode)
				if err != nil {
					fmt.Printf("Error getting Kode: %v\n", err)
					return false
				}
				return refreshedKode.Spec.Credentials.Username == updatedUsername
			}, timeout, interval).Should(BeTrue(), "Kode resource should be updated with new credentials")

			// verify Secret is updated
			kodeSecret := &kodev1alpha2.Kode{}
			err := k8sClient.Get(ctx, kodeNamespacedName, kodeSecret)
			Expect(err).NotTo(HaveOccurred())

			secretNamespacedName := types.NamespacedName{Name: kodeSecret.GetSecretName(), Namespace: namespace.Name}
			refreshedSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretNamespacedName, refreshedSecret)
				if err != nil {
					fmt.Printf("Error getting Secret: %v\n", err)
					return false
				}
				return string(refreshedSecret.Data["username"]) == updatedUsername
			}, timeout, interval).Should(BeTrue(), "Secret should be updated with new credentials")

			// Cleanup
			Expect(k8sClient.Delete(ctx, updatedKode)).To(Succeed())

			// Verify cleanup
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
			}, timeout, interval).Should(MatchError(ContainSubstring("not found")), "Kode resource should be deleted")
		})
	})
})
