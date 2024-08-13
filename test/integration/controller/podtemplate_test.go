// test/integration/controller/kode_integration_test.go

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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 60
	interval = time.Second * 1

	resourceNamespace = "test-namespace"
	kodeResourceName  = "kode"
	podTemplateKind   = "ClusterPodTemplate"
	storageSize       = "1Gi"

	podTemplateNameCodeServer = "podtemplate-codeserver"
	podTemplateNameWebtop     = "podtemplate-webtop"

	entryPointName = "entrypoint"

	podTemplateImageCodeServer = "linuxserver/code-server:latest"
	podTemplateImageWebtop     = "linuxserver/webtop:debian-xfce"
)

func createPodTemplate(name, image, templateType string) *kodev1alpha2.ClusterPodTemplate {
	template := &kodev1alpha2.ClusterPodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kodev1alpha2.ClusterPodTemplateSpec{
			PodTemplateSharedSpec: kodev1alpha2.PodTemplateSharedSpec{
				BaseSharedSpec: kodev1alpha2.BaseSharedSpec{
					Port: 8000,
				},
				Type:  templateType,
				Image: image,
			},
		},
	}

	return template
}

func waitForResourceDeletion(ctx context.Context, c *mockClient, obj client.Object, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

var _ = Describe("Kode Controller Integration", Ordered, func() {

	var (
		ctx                   context.Context
		namespace             *corev1.Namespace
		podTemplateCodeServer *kodev1alpha2.ClusterPodTemplate
		podTemplateWebtop     *kodev1alpha2.ClusterPodTemplate
		entryPoint            *kodev1alpha2.EntryPoint
	)

	BeforeAll(func() {
		ctx = context.Background()

		// Create namespace
		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create EntryPoint
		entryPoint = &kodev1alpha2.EntryPoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      entryPointName,
				Namespace: resourceNamespace,
			},
			Spec: kodev1alpha2.EntryPointSpec{
				EntryPointSharedSpec: kodev1alpha2.EntryPointSharedSpec{
					RoutingType: "domain",
					URL:         "kode.example.com",
				},
			},
		}
		Expect(k8sClient.Create(ctx, entryPoint)).To(Succeed())

		// Create PodTemplates
		podTemplateCodeServer = createPodTemplate(podTemplateNameCodeServer, podTemplateImageCodeServer, "code-server")
		podTemplateWebtop = createPodTemplate(podTemplateNameWebtop, podTemplateImageWebtop, "webtop")

		Expect(k8sClient.Create(ctx, podTemplateCodeServer)).To(Succeed())
		Expect(k8sClient.Create(ctx, podTemplateWebtop)).To(Succeed())
	})

	AfterAll(func() {
		// Cleanup resources
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		Expect(k8sClient.Delete(ctx, podTemplateCodeServer)).To(Succeed())
		Expect(k8sClient.Delete(ctx, podTemplateWebtop)).To(Succeed())
		Expect(k8sClient.Delete(ctx, entryPoint)).To(Succeed())
	})

	DescribeTable("Kode resource creation",
		func(templateName string, templateType string, expectedContainerCount int, exposePort int32) {
			kodeName := fmt.Sprintf("%s-%s", kodeResourceName, templateName)
			statefulSetName := kodeName
			storageClassName := "standard"
			kode := &kodev1alpha2.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeName,
					Namespace: namespace.Name,
				},
				Spec: kodev1alpha2.KodeSpec{
					TemplateRef: kodev1alpha2.KodeTemplateReference{
						Kind: podTemplateKind,
						Name: templateName,
					},
					Credentials: kodev1alpha2.CredentialsSpec{
						Username: "abc",
						Password: "123",
					},
					Storage: kodev1alpha2.KodeStorageSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(storageSize),
							},
						},
						StorageClassName: &storageClassName,
					},
				},
			}

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
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "DEFAULT_WORKSPACE", Value: "/config/workspace"}))
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
		kodeName := "kode-no-storage"
		kode := &kodev1alpha2.Kode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kodeName,
				Namespace: namespace.Name,
			},
			Spec: kodev1alpha2.KodeSpec{
				TemplateRef: kodev1alpha2.KodeTemplateReference{
					Kind: podTemplateKind,
					Name: podTemplateNameCodeServer,
				},
				// No storage specification
			},
		}

		// Create Kode resource
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
})
