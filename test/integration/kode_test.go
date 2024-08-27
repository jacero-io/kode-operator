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
	"strings"
	"time"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("EntryPoint Routing Integration", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Second * 1
	)

	createEntryPoint := func(name string, routingType kodev1alpha2.RoutingType) *kodev1alpha2.EntryPoint {
		return &kodev1alpha2.EntryPoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace.Name,
			},
			Spec: kodev1alpha2.EntryPointSpec{
				RoutingType: routingType,
				BaseDomain:  "test.example.com",
			},
		}
	}

	createKodeWithEntryPoint := func(name string, templateName string) *kodev1alpha2.Kode {
		return &kodev1alpha2.Kode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace.Name,
			},
			Spec: kodev1alpha2.KodeSpec{
				TemplateRef: kodev1alpha2.CrossNamespaceObjectReference{
					Kind: "ClusterPodTemplate",
					Name: kodev1alpha2.ObjectName(templateName),
				},
				Credentials: &kodev1alpha2.CredentialsSpec{
					Username: "testuser",
					Password: "testpass",
				},
			},
		}
	}

	It("should correctly route Kode resources based on EntryPoint RoutingType", func() {
		ctx := context.Background()

		var subdomainPodTemplate *kodev1alpha2.ClusterPodTemplate
		var pathPodTemplate *kodev1alpha2.ClusterPodTemplate

		By("Creating a subdomain-based EntryPoint")
		subdomainEntryPoint := createEntryPoint("subdomain-entry", kodev1alpha2.RoutingTypeSubdomain)
		Expect(k8sClient.Create(ctx, subdomainEntryPoint)).To(Succeed())

		By("Creating a ClusterPodTemplate with an entrypoint with subdomain routing")
		subdomainPodTemplate = createPodTemplate("podtemplate-with-entrypoint-subdomain", podTemplateImageCodeServer, "code-server", subdomainEntryPoint.Name, subdomainEntryPoint.Namespace)
		Expect(k8sClient.Create(ctx, subdomainPodTemplate)).To(Succeed())

		By("Creating a Kode resource")
		subdomainKode := createKodeWithEntryPoint("kode-subdomain", subdomainPodTemplate.Name)
		Expect(k8sClient.Create(ctx, subdomainKode)).To(Succeed())

		By("Verifying subdomain routing")
		Eventually(func() bool {
			updatedKode := &kodev1alpha2.Kode{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: subdomainKode.Name, Namespace: namespace.Name}, updatedKode)
			if err != nil {
				return false
			}
			return strings.HasPrefix(string(updatedKode.Status.KodeUrl), fmt.Sprintf("http://%s.", subdomainKode.Name))
		}, timeout, interval).Should(BeTrue())

		By("Creating a path-based EntryPoint")
		pathEntryPoint := createEntryPoint("path-entry", kodev1alpha2.RoutingTypePath)
		Expect(k8sClient.Create(ctx, pathEntryPoint)).To(Succeed())

		By("Creating a ClusterPodTemplate with an entrypoint with subdomain routing")
		pathPodTemplate = createPodTemplate("podtemplate-with-entrypoint-path", podTemplateImageCodeServer, "code-server", pathEntryPoint.Name, pathEntryPoint.Namespace)
		Expect(k8sClient.Create(ctx, pathPodTemplate)).To(Succeed())

		By("Creating a Kode resource with path routing")
		pathKode := createKodeWithEntryPoint("kode-path", pathPodTemplate.Name)
		Expect(k8sClient.Create(ctx, pathKode)).To(Succeed())

		By("Verifying path routing")
		Eventually(func() bool {
			updatedKode := &kodev1alpha2.Kode{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: pathKode.Name, Namespace: namespace.Name}, updatedKode)
			if err != nil {
				return false
			}
			return strings.Contains(string(updatedKode.Status.KodeUrl), fmt.Sprintf("/%s", pathKode.Name))
		}, timeout, interval).Should(BeTrue())

		By("Cleaning up resources")
		Expect(k8sClient.Delete(ctx, subdomainKode)).To(Succeed())
		Expect(k8sClient.Delete(ctx, pathKode)).To(Succeed())
		Expect(k8sClient.Delete(ctx, subdomainPodTemplate)).To(Succeed())
		Expect(k8sClient.Delete(ctx, pathPodTemplate)).To(Succeed())
		Expect(k8sClient.Delete(ctx, subdomainEntryPoint)).To(Succeed())
		Expect(k8sClient.Delete(ctx, pathEntryPoint)).To(Succeed())

		By("Verifying all resources are deleted")
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: subdomainKode.Name, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: pathKode.Name, Namespace: namespace.Name}, &kodev1alpha2.Kode{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: subdomainPodTemplate.Name, Namespace: namespace.Name}, &kodev1alpha2.ClusterPodTemplate{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: pathPodTemplate.Name, Namespace: namespace.Name}, &kodev1alpha2.ClusterPodTemplate{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: subdomainEntryPoint.Name, Namespace: namespace.Name}, &kodev1alpha2.EntryPoint{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: pathEntryPoint.Name, Namespace: namespace.Name}, &kodev1alpha2.EntryPoint{})
		}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
	})
})
