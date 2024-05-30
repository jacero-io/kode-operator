package controller

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Kode Controller", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		testEnv    *envtest.Environment
		k8sClient  client.Client
		k8sManager ctrl.Manager
		reconciler *KodeReconciler
	)

	BeforeEach(func() {
		By("bootstrapping test environment")
		ctx, cancel = context.WithCancel(context.Background())
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		err = kodev1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		k8sClient = k8sManager.GetClient()
		Expect(k8sClient).ToNot(BeNil())

		reconciler = &KodeReconciler{
			Client: k8sClient,
			Scheme: k8sManager.GetScheme(),
			Log:    ctrl.Log.WithName("controllers").WithName("Kode"),
		}

		err = reconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(ctx)).To(Succeed())
		}()
	})

	AfterEach(func() {
		cancel()
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

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
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind: "KodeTemplate",
						Name: "test-kodetemplate",
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
			err := k8sClient.Create(ctx, kode)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind KodeTemplate")
			kodeTemplate := &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-kodetemplate",
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					Image:       "lscr.io/linuxserver/code-server:latest",
					ServicePort: 8443,
					Password:    "password",
					EnvoyProxyTemplateRef: kodev1alpha1.EnvoyProxyTemplateReference{
						Kind: "EnvoyProxyTemplate",
						Name: "test-envoyproxytemplate",
					},
				},
			}
			err = k8sClient.Create(ctx, kodeTemplate)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind EnvoyProxyTemplate")
			envoyProxyTemplate := &kodev1alpha1.EnvoyProxyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-envoyproxytemplate",
				},
				Spec: kodev1alpha1.EnvoyProxyTemplateSpec{
					Image: "envoyproxy/envoy:v1.30-latest",
					HTTPFilters: []kodev1alpha1.HTTPFilter{
						{
							Name: "envoy.filters.http.router",
							TypedConfig: kodev1alpha1.TypedConfig{
								Type: "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
							},
						},
					},
				},
			}
			err = k8sClient.Create(ctx, envoyProxyTemplate)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("deleting the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())

			By("deleting the custom resource for the Kind KodeTemplate")
			kodeTemplate := &kodev1alpha1.KodeTemplate{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-kodetemplate"}, kodeTemplate)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, kodeTemplate)).To(Succeed())

			By("deleting the custom resource for the Kind EnvoyProxyTemplate")
			envoyProxyTemplate := &kodev1alpha1.EnvoyProxyTemplate{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-envoyproxytemplate"}, envoyProxyTemplate)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, envoyProxyTemplate)).To(Succeed())
		})

		It("should create a Deployment for the Kode resource", func() {
			By("checking if the Deployment has been created")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, deployment)
				if err != nil {
					return false
				}
				return true
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("lscr.io/linuxserver/code-server:latest"))
		})

		It("should create a Service for the Kode resource", func() {
			By("checking if the Service has been created")
			service := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, service)
				if err != nil {
					return false
				}
				return true
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8443)))
		})

		It("should create a PersistentVolumeClaim for the Kode resource if storage is defined", func() {
			By("checking if the PersistentVolumeClaim has been created")
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-pvc", Namespace: resourceNamespace}, pvc)
				if err != nil {
					return false
				}
				return true
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
		})

		It("should update the PersistentVolumeClaim when the storage specification changes", func() {
			By("updating the Kode resource to change the storage specification")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			Expect(err).NotTo(HaveOccurred())

			kode.Spec.Storage = kodev1alpha1.KodeStorageSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			}

			err = k8sClient.Update(ctx, kode)
			Expect(err).NotTo(HaveOccurred())

			By("checking if the PersistentVolumeClaim has been updated")
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-pvc", Namespace: resourceNamespace}, pvc)
				if err != nil {
					return false
				}
				return pvc.Spec.Resources.Requests[corev1.ResourceStorage] == resource.MustParse("2Gi")
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})

		It("should handle missing required fields gracefully", func() {
			By("creating the custom resource with missing required fields")
			invalidKode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-resource",
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind: "KodeTemplate",
						Name: "missing-template",
					},
				},
			}
			err := k8sClient.Create(ctx, invalidKode)
			Expect(err).To(HaveOccurred())
		})

		It("should handle an invalid image name", func() {
			By("creating the custom resource with an invalid image name")
			invalidImageKode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-image-resource",
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind: "KodeTemplate",
						Name: "invalid-image-template",
					},
				},
			}
			err := k8sClient.Create(ctx, invalidImageKode)
			Expect(err).NotTo(HaveOccurred())

			By("creating the KodeTemplate with an invalid image name")
			invalidImageKodeTemplate := &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-image-template",
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					Image:       "invalid-image-name",
					ServicePort: 8443,
					Password:    "password",
				},
			}
			err = k8sClient.Create(ctx, invalidImageKodeTemplate)
			Expect(err).NotTo(HaveOccurred())

			By("checking if the Deployment has been created")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "invalid-image-resource", Namespace: resourceNamespace}, deployment)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("invalid-image-name"))
		})

		It("should handle an invalid port number", func() {
			By("creating the custom resource with an invalid port number")
			invalidPortKode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-port-resource",
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind: "KodeTemplate",
						Name: "invalid-port-template",
					},
				},
			}
			err := k8sClient.Create(ctx, invalidPortKode)
			Expect(err).To(HaveOccurred())
		})
	})
})
