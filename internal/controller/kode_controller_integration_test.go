//go:build integration

package controller

import (
	"context"
	"fmt"
	"time"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = time.Second * 30
	interval = time.Millisecond * 250
)

var _ = Describe("Kode Controller Integration", Ordered, func() {
	const (
		resourceNamespace                  = "test-namespace"
		kodeResourceName                   = "test-kode"
		kodeTemplateKind                   = "KodeClusterTemplate"
		kodeTemplateNameWithoutEnvoy       = "test-kodetemplate-without-envoy"
		kodeTemplateNameWithEnvoy          = "test-kodetemplate-with-envoy"
		kodeTemplateImage                  = "lscr.io/linuxserver/code-server:latest"
		envoyProxyConfigKind               = "EnvoyProxyClusterConfig"
		envoyProxyConfigName               = "test-envoyproxyconfig"
		envoyProxyConfigImage              = "envoyproxy/envoy:v1.30-latest"
		withEnvoyPort                int32 = 8000
		withoutEnvoyPort             int32 = 3000
	)

	var (
		ctx                      context.Context
		namespace                *corev1.Namespace
		kodeTemplateWithoutEnvoy *kodev1alpha1.KodeClusterTemplate
		kodeTemplateWithEnvoy    *kodev1alpha1.KodeClusterTemplate
		envoyProxyConfig         *kodev1alpha1.EnvoyProxyClusterConfig
	)

	BeforeAll(func() {
		ctx = context.Background()

		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: resourceNamespace,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		envoyProxyConfig = &kodev1alpha1.EnvoyProxyClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: envoyProxyConfigName,
			},
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

		kodeTemplateWithoutEnvoy = &kodev1alpha1.KodeClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: kodeTemplateNameWithoutEnvoy,
			},
			Spec: kodev1alpha1.KodeClusterTemplateSpec{
				SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
					Image: kodeTemplateImage,
					Type:  "code-server",
				},
			},
		}
		Expect(k8sClient.Create(ctx, kodeTemplateWithoutEnvoy)).To(Succeed())

		kodeTemplateWithEnvoy = &kodev1alpha1.KodeClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: kodeTemplateNameWithEnvoy,
			},
			Spec: kodev1alpha1.KodeClusterTemplateSpec{
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

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateWithEnvoy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, kodeTemplateWithoutEnvoy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, envoyProxyConfig)).To(Succeed())
	})

	DescribeTable("Kode resource creation",
		func(templateName string, expectedContainerCount int, port int32) {
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

			// Ensure the Kode resource doesn't exist before creating
			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
			if err == nil {
				Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
				}, timeout, interval).ShouldNot(Succeed())
			}

			Expect(k8sClient.Create(ctx, kode)).To(Succeed())

			statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
			createdStatefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
			}, timeout, interval).Should(Succeed())
			Expect(createdStatefulSet.Name).To(Equal(kodeName))
			Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))
			Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(expectedContainerCount))

			serviceLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
			createdService := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceLookupKey, createdService)
			}, timeout, interval).Should(Succeed())
			Expect(createdService.Name).To(Equal(kodeName))
			Expect(createdService.Spec.Ports).To(HaveLen(1))
			Expect(createdService.Spec.Ports[0].Port).To(Equal(int32(port)))

			// Cleanup
			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
			}, timeout, interval).ShouldNot(Succeed())
		},
		Entry("without Envoy Proxy", kodeTemplateNameWithoutEnvoy, 1, withoutEnvoyPort),
		Entry("with Envoy Proxy", kodeTemplateNameWithEnvoy, 2, withEnvoyPort),
	)

	It("should create a StatefulSet the Kode resource", func() {
		kodeName := fmt.Sprintf("%s-%s", kodeResourceName, kodeTemplateNameWithEnvoy)
		kode := &kodev1alpha1.Kode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kodeName,
				Namespace: namespace.Name,
			},
			Spec: kodev1alpha1.KodeSpec{
				TemplateRef: kodev1alpha1.KodeTemplateReference{
					Kind: kodeTemplateKind,
					Name: kodeTemplateNameWithEnvoy,
				},
			},
		}

		// Ensure the Kode resource doesn't exist before creating
		err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
		if err == nil {
			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
			}, timeout, interval).ShouldNot(Succeed())
		}

		Expect(k8sClient.Create(ctx, kode)).To(Succeed())

		statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
		createdStatefulSet := &appsv1.StatefulSet{}

		Eventually(func() error {
			return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
		}, timeout, interval).Should(Succeed())

		Expect(createdStatefulSet.Name).To(Equal(kodeName))
		Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))

		// Cleanup
		Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha1.Kode{})
		}, timeout, interval).ShouldNot(Succeed())
	})

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
})
