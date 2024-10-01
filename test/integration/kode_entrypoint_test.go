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
// 	"context"

// 	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	appsv1 "k8s.io/api/apps/v1"
// 	"k8s.io/apimachinery/pkg/types"
// )

// var _ = Describe("EntryPoint Routing Integration", func() {

// 	It("should correctly route Kode resources based on EntryPoint RoutingType", func() {
// 		ctx := context.Background()

// 		var kodeSubdomain *kodev1alpha2.Kode
// 		var kodePath *kodev1alpha2.Kode

// 		username := "abc"
// 		password := "123"

// 		credentials := &kodev1alpha2.CredentialsSpec{
// 			Username: username,
// 			Password: password,
// 		}

// 		// Create Kode resources
// 		By("Creating a Kode resource with subdomain routing")
// 		kodeSubdomain = createKode("kode-subdomain", namespace.Name, containerTemplateSubdomain.Name, credentials, nil)
// 		Expect(k8sClient.Create(ctx, kodeSubdomain)).To(Succeed())

// 		By("Creating a Kode resource with path routing")
// 		kodePath = createKode("kode-path", namespace.Name, containerTemplatePath.Name, credentials, nil)
// 		Expect(k8sClient.Create(ctx, kodePath)).To(Succeed())

// 		// Check StatefulSet
// 		statefulSetLookupKeySubdomain := types.NamespacedName{Name: kodeSubdomain.Name, Namespace: namespace.Name}
// 		createdStatefulSet := &appsv1.StatefulSet{}
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, statefulSetLookupKeySubdomain, createdStatefulSet)
// 		}, timeout, interval).Should(Succeed())

// 		Expect(createdStatefulSet.Name).To(Equal(kodeSubdomain.Name))
// 		Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(1))

// 		statefulSetLookupKeyPath := types.NamespacedName{Name: kodePath.Name, Namespace: namespace.Name}
// 		createdStatefulSet = &appsv1.StatefulSet{}
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, statefulSetLookupKeyPath, createdStatefulSet)
// 		}, timeout, interval).Should(Succeed())

// 		Expect(createdStatefulSet.Name).To(Equal(kodePath.Name))
// 		Expect(createdStatefulSet.Spec.Template.Spec.Containers).To(HaveLen(1))

// 		// Verify subdomain routing
// 		// By("Verifying subdomain routing")
// 		// Eventually(func() bool {
// 		// 	updatedKode := &kodev1alpha2.Kode{}
// 		// 	err := k8sClient.Get(ctx, types.NamespacedName{Name: kodeSubdomain.Name, Namespace: namespace.Name}, updatedKode)
// 		// 	if err != nil {
// 		// 		return false
// 		// 	}
// 		// 	return strings.HasPrefix(string(updatedKode.Status.KodeUrl), fmt.Sprintf("http://%s.", kodeSubdomain.Name))
// 		// }, timeout, interval).Should(BeTrue())

// 		// Verify path routing
// 		// By("Verifying path routing")
// 		// Eventually(func() bool {
// 		// 	updatedKode := &kodev1alpha2.Kode{}
// 		// 	err := k8sClient.Get(ctx, types.NamespacedName{Name: kodePath.Name, Namespace: namespace.Name}, updatedKode)
// 		// 	if err != nil {
// 		// 		return false
// 		// 	}
// 		// 	return strings.Contains(string(updatedKode.Status.KodeUrl), fmt.Sprintf("/%s", kodePath.Name))
// 		// }, timeout, interval).Should(BeTrue())

// 		By("Cleaning up resources")
// 		Expect(k8sClient.Delete(ctx, kodeSubdomain)).To(Succeed())
// 		Expect(k8sClient.Delete(ctx, kodePath)).To(Succeed())

// 		By("Verifying all resources are deleted")
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, types.NamespacedName{Name: kodeSubdomain.Name, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
// 		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
// 		Eventually(func() error {
// 			return k8sClient.Get(ctx, types.NamespacedName{Name: kodePath.Name, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
// 		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
// 	})
// })
