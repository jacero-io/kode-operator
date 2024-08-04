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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = time.Second * 60
	interval = time.Second * 1

	resourceNamespace     = "test-namespace"
	kodeResourceName      = "kode"
	kodeTemplateKind      = "KodeClusterTemplate"
	envoyProxyConfigKind  = "EnvoyProxyClusterConfig"
	envoyProxyConfigName  = "test-envoyproxyconfig"
	envoyProxyConfigImage = "envoyproxy/envoy:v1.31-latest"

	kodeTemplateNameCodeServerWithoutEnvoy = "test-kodetemplate-codeserver-without-envoy"
	kodeTemplateNameCodeServerWithEnvoy    = "test-kodetemplate-codeserver-with-envoy"
	kodeTemplateNameWebtopWithoutEnvoy     = "test-kodetemplate-webtop-without-envoy"
	kodeTemplateNameWebtopWithEnvoy        = "test-kodetemplate-webtop-with-envoy"

	kodeTemplateImageCodeServer = "lscr.io/linuxserver/code-server:latest"
	kodeTemplateImageWebtop     = "lscr.io/linuxserver/webtop:latest"
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
			Expect(k8sClient.Delete(ctx, kode)).To(Succeed()) // Expect the Kode resource to be deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		},
		Entry("code-server without Envoy Proxy", kodeTemplateNameCodeServerWithoutEnvoy, "code-server", false, 1, int32(8000), int32(8000)),
		Entry("code-server with Envoy Proxy", kodeTemplateNameCodeServerWithEnvoy, "code-server", true, 2, int32(8000), int32(3000)),
		Entry("webtop without Envoy Proxy", kodeTemplateNameWebtopWithoutEnvoy, "webtop", false, 1, int32(8000), int32(8000)),
		Entry("webtop with Envoy Proxy", kodeTemplateNameWebtopWithEnvoy, "webtop", true, 2, int32(8000), int32(3000)),
	)
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

// It("should create a PersitentVolumeClaim for the Kode resource", func() {
// 	kodeName := fmt.Sprintf("%s-pvc", kodeResourceName)
// 	kode := &kodev1alpha1.Kode{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      kodeName,
// 			Namespace: namespace.Name,
// 		},
// 		Spec: kodev1alpha1.KodeSpec{
// 			TemplateRef: kodev1alpha1.KodeTemplateReference{
// 				Kind: kodeTemplateKind,
// 				Name: kodeTemplateNameWithoutEnvoy,
// 			},
// 			Storage: kodev1alpha1.KodeStorageSpec{
// 				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
// 				Resources: corev1.VolumeResourceRequirements{
// 					Requests: corev1.ResourceList{
// 						corev1.ResourceStorage: resource.MustParse("1Gi"),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	// Ensure the Kode resource doesn't exist before creating
// 	err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
// 	if err == nil {
// 		Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
// 		}, timeout, interval).ShouldNot(Succeed())
// 	}

// 	Expect(k8sClient.Create(ctx, kode)).To(Succeed())

// 	statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
// 	createdStatefulSet := &appsv1.StatefulSet{}

// 	Eventually(func() error {
// 		return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
// 	}, timeout, interval).Should(Succeed())
// 	Expect(createdStatefulSet.Name).To(Equal(kodeName))
// 	Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))
// 	Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(1))

// 	pvcLookupKey := types.NamespacedName{Name: kodeResourceName, Namespace: namespace.Name}
// 	createdPVC := &corev1.PersistentVolumeClaim{}
// 	Eventually(func() error {
// 		return k8sClient.Get(ctx, pvcLookupKey, createdPVC)
// 	}, timeout, interval).Should(Succeed())
// 	Expect(createdPVC.Name).To(Equal(kodeResourceName))
// 	Expect(createdPVC.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
// 	Expect(createdPVC.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
// })
