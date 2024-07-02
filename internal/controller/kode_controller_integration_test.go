//go:build integration

// internal/controller/kode_controller_integration_test.go

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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = time.Second * 30
	interval = time.Millisecond * 250
)

var _ = Describe("Kode Controller Integration", func() {
	const (
		resourceNamespace      = "test-namespace"
		kodeResourceName       = "test-kode"
		kodeTemplateKind       = "KodeTemplate"
		kodeTemplateName       = "test-kodetemplate"
		kodeTemplateImage      = "lscr.io/linuxserver/code-server:latest"
		envoyProxyConfigKind   = "EnvoyProxyConfig"
		envoyProxyConfigName   = "test-envoyproxyconfig"
		envoyProxyConfigImage  = "envoyproxy/envoy:v1.30-latest"
		envoyProxyConfigFilter = `{
			"@type":"type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz",
			"with_request_body":{
				"max_request_bytes":8192,
				"allow_partial_message":true
			},
			"failure_mode_allow":false,
			"grpc_service":{
				"envoy_grpc":{
					"cluster_name":"ext_authz_server"
				},
				"timeout":"0.5s"
			},
			"transport_api_version":"v3"
		}`
	)

	Context("When reconciling a Kode resource", func() {
		var (
			namespace                *corev1.Namespace
			kodeTemplateWithoutEnvoy *kodev1alpha1.KodeTemplate
			kodeTemplateWithEnvoy    *kodev1alpha1.KodeTemplate
			envoyProxyConfig         *kodev1alpha1.EnvoyProxyConfig
		)
		BeforeEach(func() {
			ctx = context.Background()
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			envoyProxyConfig = &kodev1alpha1.EnvoyProxyConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      envoyProxyConfigName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.EnvoyProxyConfigSpec{
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

			kodeTemplateWithoutEnvoy = &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeTemplateName + "without-envoy",
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
						Image: kodeTemplateImage,
						Type:  "code-server",
					},
				},
			}
			Expect(k8sClient.Create(ctx, kodeTemplateWithoutEnvoy)).To(Succeed())

			kodeTemplateWithEnvoy = &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeTemplateName + "with-envoy",
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
						EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
							Kind: envoyProxyConfigKind,
							Name: envoyProxyConfigName,
						},
						Image: kodeTemplateImage,
						Type:  "code-server",
					},
				},
			}
			Expect(k8sClient.Create(ctx, kodeTemplateWithEnvoy)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})

		It("should create a StatefulSet for the Kode resource without Envoy Proxy", func() {
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeResourceName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind:      kodeTemplateKind,
						Name:      kodeTemplateName + "without-envoy",
						Namespace: resourceNamespace,
					},
					Storage: kodev1alpha1.KodeStorageSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, kode)).To(Succeed())

			statefulSetLookupKey := types.NamespacedName{Name: kodeResourceName, Namespace: resourceNamespace}
			createdStatefulSet := &appsv1.StatefulSet{}

			Eventually(func() error {
				return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
			}, timeout, interval).Should(Succeed())

			Expect(createdStatefulSet.Name).To(Equal(kodeResourceName))
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))
		})

		It("should create a StatefulSet for the Kode resource with Envoy Proxy", func() {
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeResourceName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind:      kodeTemplateKind,
						Name:      kodeTemplateName + "with-envoy",
						Namespace: resourceNamespace,
					},
					Storage: kodev1alpha1.KodeStorageSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, kode)).To(Succeed())

			statefulSetLookupKey := types.NamespacedName{Name: kodeResourceName, Namespace: resourceNamespace}
			createdStatefulSet := &appsv1.StatefulSet{}

			Eventually(func() error {
				return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
			}, timeout, interval).Should(Succeed())

			Expect(createdStatefulSet.Name).To(Equal(kodeResourceName))
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))
		})

		// It("should create a Service for the Kode resource", func() {
		// 	kode := &kodev1alpha1.Kode{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      kodeResourceName,
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.KodeSpec{
		// 			TemplateRef: kodev1alpha1.KodeTemplateReference{
		// 				Kind:      kodeTemplateKind,
		// 				Name:      kodeTemplateName,
		// 				Namespace: resourceNamespace,
		// 			},
		// 		},
		// 	}
		// 	Expect(k8sClient.Create(ctx, kode)).To(Succeed())

		// 	serviceLookupKey := types.NamespacedName{Name: kodeResourceName, Namespace: resourceNamespace}
		// 	createdService := &corev1.Service{}

		// 	Eventually(func() error {
		// 		return k8sClient.Get(ctx, serviceLookupKey, createdService)
		// 	}, timeout, interval).Should(Succeed())

		// 	Expect(createdService.Name).To(Equal(kodeResourceName))
		// 	Expect(createdService.Spec.Ports).To(HaveLen(1))
		// 	Expect(createdService.Spec.Ports[0].Port).To(Equal(int32(3000)))
		// })

		// Add more integration tests here...
	})
})
