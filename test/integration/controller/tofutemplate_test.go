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

// import (
// 	tofuv1alpha2 "github.com/flux-iac/tofu-controller/api/v1alpha2"
// 	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// const (
// 	tofuTemplateKind = "ClusterTofuTemplate"
// 	tofuTemplateName = "random-template"
// )

// func createTofuTemplate(name string) *kodev1alpha2.ClusterTofuTemplate {
// 	template := &kodev1alpha2.ClusterTofuTemplate{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: name,
// 		},
// 		Spec: kodev1alpha2.ClusterTofuTemplateSpec{
// 			TofuSharedSpec: kodev1alpha2.TofuSharedSpec{
// 				TofuSpec: tofuv1alpha2.TerraformSpec{
// 					Destroy: false,
// 					Path:    "path/to/terraform",
// 					SourceRef: tofuv1alpha2.CrossNamespaceSourceReference{
// 						Kind:      "GitRepository",
// 						Name:      "random-git-repo",
// 						Namespace: resourceNamespace,
// 					},
// 				},
// 			},
// 		},
// 	}

// 	return template
// }

// var _ = Describe("Kode Controller TofuTemplate Integration", Ordered, func() {

// 	var (
// 		ctx           context.Context
// 		namespace     *corev1.Namespace
// 		tofuTemplate1 *kodev1alpha2.ClusterTofuTemplate
// 		entryPoint    *kodev1alpha2.EntryPoint
// 	)

// 	BeforeAll(func() {
// 		ctx = context.Background()

// 		// Create namespace
// 		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
// 		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

// 		// Create EntryPoint
// 		entryPoint = &kodev1alpha2.EntryPoint{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      entryPointName,
// 				Namespace: resourceNamespace,
// 			},
// 			Spec: kodev1alpha2.EntryPointSpec{
// 				EntryPointSharedSpec: kodev1alpha2.EntryPointSharedSpec{
// 					RoutingType: "domain",
// 					URL:         "kode.example.com",
// 				},
// 			},
// 		}
// 		Expect(k8sClient.Create(ctx, entryPoint)).To(Succeed())

// 		// Create PodTemplates
// 		tofuTemplate1 = createTofuTemplate(podTemplateNameCodeServer)

// 		Expect(k8sClient.Create(ctx, tofuTemplate1)).To(Succeed())
// 	})

// 	AfterAll(func() {
// 		// Cleanup resources
// 		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
// 		Expect(k8sClient.Delete(ctx, tofuTemplate1)).To(Succeed())
// 		Expect(k8sClient.Delete(ctx, entryPoint)).To(Succeed())
// 	})

// 	DescribeTable("Kode resource creation",
// 		func(templateName string, templateType string, expectedContainerCount int, exposePort int32) {
// 			kodeName := fmt.Sprintf("%s-%s", kodeResourceName, templateName)
// 			kode := &kodev1alpha2.Kode{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      kodeName,
// 					Namespace: namespace.Name,
// 				},
// 				Spec: kodev1alpha2.KodeSpec{
// 					TemplateRef: kodev1alpha2.KodeTemplateReference{
// 						Kind: podTemplateKind,
// 						Name: templateName,
// 					},
// 					Credentials: kodev1alpha2.CredentialsSpec{
// 						Username: "abc",
// 						Password: "123",
// 					},
// 				},
// 			}

// 			// Create Kode resource
// 			Expect(k8sClient.Create(ctx, kode)).To(Succeed())

// 			// Check VM in OpenStack

// 			// Cleanup
// 			By("Deleting the Kode resource")
// 			Expect(k8sClient.Delete(ctx, kode)).To(Succeed())

// 			By("Waiting for Kode resource to be deleted")
// 			Eventually(func() bool {
// 				err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
// 				return errors.IsNotFound(err)
// 			}, timeout, interval).Should(BeTrue(), "Failed to delete Kode resource")

// 			By("Ensuring all related resources are cleaned up")
// 			Eventually(func() error {
// 				// ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
// 				// defer cancel()

// 				// Check if VM is deleted

// 				return nil
// 			}, time.Minute*5, time.Second).Should(Succeed(), "Failed to clean up all resources")
// 		},
// 		Entry("code-server", podTemplateNameCodeServer, "code-server", 1, int32(8000)),
// 		Entry("webtop", podTemplateNameWebtop, "webtop", 1, int32(8000)),
// 	)

// 	It("should not create a PersistentVolumeClaim when storage is not specified", func() {
// 		kodeName := "kode-no-storage"
// 		kode := &kodev1alpha2.Kode{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      kodeName,
// 				Namespace: namespace.Name,
// 			},
// 			Spec: kodev1alpha2.KodeSpec{
// 				TemplateRef: kodev1alpha2.KodeTemplateReference{
// 					Kind: podTemplateKind,
// 					Name: podTemplateNameCodeServer,
// 				},
// 				// No storage specification
// 			},
// 		}

// 		// Create Kode resource
// 		Expect(k8sClient.Create(ctx, kode)).To(Succeed())

// 		// Check StatefulSet
// 		statefulSetLookupKey := types.NamespacedName{Name: kodeName, Namespace: namespace.Name}
// 		createdStatefulSet := &appsv1.StatefulSet{}
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, statefulSetLookupKey, createdStatefulSet)
// 		}, timeout, interval).Should(Succeed())

// 		// Ensure no PVC-related volume or volumeMount exists
// 		Expect(createdStatefulSet.Spec.Template.Spec.Volumes).NotTo(ContainElement(HaveField("Name", common.KodeVolumeStorageName)))
// 		Expect(createdStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts).NotTo(ContainElement(HaveField("Name", common.KodeVolumeStorageName)))

// 		// Check that PVC doesn't exist
// 		pvcName := fmt.Sprintf("%s-pvc", kodeName)
// 		pvcLookupKey := types.NamespacedName{Name: pvcName, Namespace: namespace.Name}
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, pvcLookupKey, &corev1.PersistentVolumeClaim{})
// 		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

// 		// Cleanup
// 		Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
// 		Eventually(func() bool {
// 			err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeName, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
// 			return errors.IsNotFound(err)
// 		}, timeout, interval).Should(BeTrue())
// 	})
// })
