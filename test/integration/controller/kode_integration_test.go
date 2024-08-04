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

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 120
	interval = time.Second * 1

	resourceNamespace     = "test-namespace"
	kodeResourceName      = "kode"
	kodeTemplateKind      = "KodeClusterTemplate"
	envoyProxyConfigKind  = "EnvoyProxyClusterConfig"
	envoyProxyConfigName  = "test-envoyproxyconfig"
	envoyProxyConfigImage = "envoyproxy/envoy:v1.31-latest"
	storageSize           = "1Gi"

	kodeTemplateNameCodeServerWithoutEnvoy = "test-kodetemplate-codeserver-without-envoy"
	kodeTemplateNameCodeServerWithEnvoy    = "test-kodetemplate-codeserver-with-envoy"
	kodeTemplateNameWebtopWithoutEnvoy     = "test-kodetemplate-webtop-without-envoy"
	kodeTemplateNameWebtopWithEnvoy        = "test-kodetemplate-webtop-with-envoy"

	kodeTemplateImageCodeServer = "linuxserver/code-server:latest"
	kodeTemplateImageWebtop     = "linuxserver/webtop:debian-xfce"
)

var _ = Describe("Kode Controller Integration", Ordered, func() {

	var (
		ctx                                context.Context
		namespace                          *corev1.Namespace
		envoyProxyConfig                   *kodev1alpha1.EnvoyProxyClusterConfig
		kodeTemplateCodeServerWithoutEnvoy *kodev1alpha1.KodeClusterTemplate
		kodeTemplateCodeServerWithEnvoy    *kodev1alpha1.KodeClusterTemplate
		kodeTemplateWebtopWithoutEnvoy     *kodev1alpha1.KodeClusterTemplate
		kodeTemplateWebtopWithEnvoy        *kodev1alpha1.KodeClusterTemplate
	)

	BeforeAll(func() {
		ctx = context.Background()

		// Create namespace
		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create EnvoyProxyConfig
		envoyProxyConfig = &kodev1alpha1.EnvoyProxyClusterConfig{
			ObjectMeta: metav1.ObjectMeta{Name: envoyProxyConfigName},
			Spec: kodev1alpha1.EnvoyProxyClusterConfigSpec{
				SharedEnvoyProxyConfigSpec: kodev1alpha1.SharedEnvoyProxyConfigSpec{
					Image: envoyProxyConfigImage,
					HTTPFilters: []kodev1alpha1.HTTPFilter{{
						Name: "filter1",
						TypedConfig: runtime.RawExtension{Raw: []byte(`{
                            "@type":"type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz",
                            "with_request_body":{"max_request_bytes":8192,"allow_partial_message":true},
                            "failure_mode_allow":false,
                            "grpc_service":{"envoy_grpc":{"cluster_name":"ext_authz_server"},"timeout":"0.5s"},
                            "transport_api_version":"v3"
                        }`)},
					}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, envoyProxyConfig)).To(Succeed())

		// Create KodeClusterTemplates
		kodeTemplateCodeServerWithoutEnvoy = createKodeClusterTemplate(kodeTemplateNameCodeServerWithoutEnvoy, kodeTemplateImageCodeServer, "code-server", false)
		kodeTemplateCodeServerWithEnvoy = createKodeClusterTemplate(kodeTemplateNameCodeServerWithEnvoy, kodeTemplateImageCodeServer, "code-server", true)
		kodeTemplateWebtopWithoutEnvoy = createKodeClusterTemplate(kodeTemplateNameWebtopWithoutEnvoy, kodeTemplateImageWebtop, "webtop", false)
		kodeTemplateWebtopWithEnvoy = createKodeClusterTemplate(kodeTemplateNameWebtopWithEnvoy, kodeTemplateImageWebtop, "webtop", true)

		Expect(k8sClient.Create(ctx, kodeTemplateCodeServerWithoutEnvoy)).To(Succeed())
		Expect(k8sClient.Create(ctx, kodeTemplateCodeServerWithEnvoy)).To(Succeed())
		Expect(k8sClient.Create(ctx, kodeTemplateWebtopWithoutEnvoy)).To(Succeed())
		Expect(k8sClient.Create(ctx, kodeTemplateWebtopWithEnvoy)).To(Succeed())
	})

	AfterAll(func() {
		// Cleanup resources
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		Expect(k8sClient.Delete(ctx, envoyProxyConfig)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateCodeServerWithoutEnvoy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateCodeServerWithEnvoy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateWebtopWithoutEnvoy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateWebtopWithEnvoy)).To(Succeed())
	})

	DescribeTable("Kode resource creation",
		func(templateName string, templateType string, withEnvoy bool, expectedContainerCount int, exposePort int32, containerPort int32) {
			kodeName := fmt.Sprintf("%s-%s", kodeResourceName, templateName)
			statefulSetName := kodeName
			storageClassName := "standard"
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeName,
					Namespace: namespace.Name,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind: kodeTemplateKind,
						Name: templateName,
					},
					Storage: kodev1alpha1.KodeStorageSpec{
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
			// Ensure the Kode resource doesn't exist before creating
			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
			if err == nil {
				Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
				}, timeout, interval).ShouldNot(Succeed())
			}

			// Create Kode resource
			Expect(k8sClient.Create(ctx, kode)).To(Succeed()) // Expect the creation to succeed

			// Check StatefulSet
			statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
			createdStatefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
			}, timeout, interval).Should(Succeed())

			Expect(createdStatefulSet.Name).To(Equal(kodeName))                                                         // Expect the name to be set to the kode name
			Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(expectedContainerCount))                // Except the container count to be 1 or 2 based on the template
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(containerPort)) // Expect the container port to be set to 3000 with envoy and 8000 without envoy

			if templateType == "code-server" {
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImageCodeServer))                                                 // Expect the image to be set to the template image
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "DEFAULT_WORKSPACE", Value: "/config/workspace"})) // Expect the default workspace to be set
			} else if templateType == "webtop" {
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImageWebtop))                                 // Expect the image to be set to the template image
				Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "CUSTOM_USER", Value: "abc"})) // Expect the custom user to be set
			}

			if withEnvoy {
				Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(ContainElement(HaveField("Name", "envoy-proxy"))) // Expect the envoy proxy container to be present
			}

			// Check PersistentVolumeClaim
			pvcName := fmt.Sprintf("%s-pvc", kodeName)
			pvcLookupKey := types.NamespacedName{Name: pvcName, Namespace: namespace.Name}
			createdPVC := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, pvcLookupKey, createdPVC)
			}, timeout, interval).Should(Succeed())

			// After creating the PVC, update its status to Bound
			createdPVC.Status.Phase = corev1.ClaimBound
			Expect(k8sClient.Status().Update(ctx, createdPVC)).To(Succeed())

			Expect(createdPVC.Name).To(Equal(pvcName))                                                                    // Expect the name to be set to the kode name + "pvc"
			Expect(createdPVC.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))                                  // Expect the access mode to be set to ReadWriteOnce
			Expect(createdPVC.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse(storageSize))) // Expect the storage size to be set to 1Gi
			Expect(createdPVC.Spec.StorageClassName).NotTo(BeNil())                                                       // Expect the storage class name to be set
			Expect(*createdPVC.Spec.StorageClassName).To(Equal(storageClassName))                                         // Expect the storage class name to be set to the specified storage class name'
			Expect(createdPVC.Status.Phase).To(Or(Equal(corev1.ClaimPending), Equal(corev1.ClaimBound)))                  // Expect the PVC to be in Pending or Bound phase

			// Check if the PVC is mounted in the StatefulSet
			volumeMount := corev1.VolumeMount{
				Name:      common.KodeVolumeStorageName,
				MountPath: "/config", // Adjust this path if necessary
			}
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(volumeMount)) // Expect the volume mount to be set to the PVC

			volume := corev1.Volume{
				Name: common.KodeVolumeStorageName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}
			Expect(createdStatefulSet.Spec.Template.Spec.Volumes).To(ContainElement(volume)) // Expect the volume to be set to the PVC

			// Check Service
			serviceName := fmt.Sprintf("%s-svc", kodeName)
			serviceLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace.Name}
			createdService := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())

			Expect(createdService.Name).To(Equal(serviceName))              // Expect the name to be set to the kode name + "svc"
			Expect(createdService.Spec.Ports).To(HaveLen(1))                // Expect the service to have 1 port
			Expect(createdService.Spec.Ports[0].Port).To(Equal(exposePort)) // Expect the service port to be set to the template port. Defaults to 8000

			// Cleanup
			By("Deleting the Kode resource")
			Expect(k8sClient.Delete(ctx, kode)).To(Succeed()) // Expect the deletion to succeed

			By("Waiting for Kode resource to be deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
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
		Entry("code-server without Envoy Proxy", kodeTemplateNameCodeServerWithoutEnvoy, "code-server", false, 1, int32(8000), int32(8000)),
		Entry("code-server with Envoy Proxy", kodeTemplateNameCodeServerWithEnvoy, "code-server", true, 2, int32(8000), int32(3000)),
		Entry("webtop without Envoy Proxy", kodeTemplateNameWebtopWithoutEnvoy, "webtop", false, 1, int32(8000), int32(8000)),
		Entry("webtop with Envoy Proxy", kodeTemplateNameWebtopWithEnvoy, "webtop", true, 2, int32(8000), int32(3000)),
	)

	It("should not create a PersistentVolumeClaim when storage is not specified", func() {
		kodeName := "kode-no-storage"
		kode := &kodev1alpha1.Kode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kodeName,
				Namespace: namespace.Name,
			},
			Spec: kodev1alpha1.KodeSpec{
				TemplateRef: kodev1alpha1.KodeTemplateReference{
					Kind: kodeTemplateKind,
					Name: kodeTemplateNameCodeServerWithoutEnvoy,
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
			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})

func createKodeClusterTemplate(name, image, templateType string, withEnvoy bool) *kodev1alpha1.KodeClusterTemplate {
	template := &kodev1alpha1.KodeClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kodev1alpha1.KodeClusterTemplateSpec{
			SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
				Image: image,
				Type:  templateType,
			},
		},
	}

	if withEnvoy {
		template.Spec.EnvoyConfigRef = kodev1alpha1.EnvoyConfigReference{
			Kind: envoyProxyConfigKind,
			Name: envoyProxyConfigName,
		}
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
