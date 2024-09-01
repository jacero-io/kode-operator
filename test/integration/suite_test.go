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
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	entrypointctrl "github.com/jacero-io/kode-operator/internal/controllers/entrypoint"
	kodectrl "github.com/jacero-io/kode-operator/internal/controllers/kode"
	"github.com/jacero-io/kode-operator/internal/events"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	timeout                 = time.Second * 60
	interval                = time.Millisecond * 250
	entryPointNameSubdomain = "test-entrypoint-with-subdomain"
	entryPointNamePath      = "test-entrypoint-with-path"

	resourceNamespace = "test-namespace"

	storageSize = "1Gi"

	podTemplateNameCodeServer = "podtemplate-codeserver"
	podTemplateNameWebtop     = "podtemplate-webtop"

	podTemplateImageCodeServer = "linuxserver/code-server:latest"
	podTemplateImageWebtop     = "linuxserver/webtop:debian-xfce"
)

var (
	cfg                  *rest.Config
	cancel               context.CancelFunc
	testEnv              *envtest.Environment
	k8sClient            client.Client
	mockK8sClient        *mockClient
	k8sManager           ctrl.Manager
	reconciler           *kodectrl.KodeReconciler
	entrypointReconciler *entrypointctrl.EntryPointReconciler

	ctx                   context.Context
	namespace             *corev1.Namespace
	podTemplateCodeServer *kodev1alpha2.ClusterPodTemplate
	podTemplateWebtop     *kodev1alpha2.ClusterPodTemplate

	entryPointSubdomain *kodev1alpha2.EntryPoint
	entryPointPath      *kodev1alpha2.EntryPoint
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}

type mockClient struct {
	client.Client
	deletedResources sync.Map
}

var (
	podTemplateSubdomain *kodev1alpha2.ClusterPodTemplate
	podTemplatePath      *kodev1alpha2.ClusterPodTemplate
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	ctx, cancel = context.WithCancel(context.Background())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "crd"),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kodev1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = gwapiv1.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	mockK8sClient = &mockClient{
		Client:           k8sClient,
		deletedResources: sync.Map{},
	}

	reconciler = &kodectrl.KodeReconciler{
		Client:          mockK8sClient,
		Scheme:          k8sManager.GetScheme(),
		Log:             ctrl.Log.WithName("Kode").WithName("Reconcile"),
		ResourceManager: resource.NewDefaultResourceManager(k8sClient, ctrl.Log.WithName("Kode").WithName("ResourceManager"), k8sManager.GetScheme()),
		TemplateManager: template.NewDefaultTemplateManager(k8sClient, ctrl.Log.WithName("Kode").WithName("TemplateManager")),
		CleanupManager:  cleanup.NewDefaultCleanupManager(mockK8sClient, ctrl.Log.WithName("Kode").WithName("CleanupManager")),
		StatusUpdater:   status.NewDefaultStatusUpdater(k8sClient, ctrl.Log.WithName("Kode").WithName("StatusUpdater")),
		Validator:       validation.NewDefaultValidator(),
		EventManager:    events.NewEventManager(k8sClient, ctrl.Log.WithName("Kode").WithName("EventManager"), k8sManager.GetScheme(), k8sManager.GetEventRecorderFor("kode-controller")),
	}

	entrypointReconciler = &entrypointctrl.EntryPointReconciler{
		Client:          mockK8sClient,
		Scheme:          k8sManager.GetScheme(),
		Log:             ctrl.Log.WithName("EntryPoint").WithName("Reconcile"),
		ResourceManager: resource.NewDefaultResourceManager(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("ResourceManager"), k8sManager.GetScheme()),
		TemplateManager: template.NewDefaultTemplateManager(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("TemplateManager")),
		CleanupManager:  cleanup.NewDefaultCleanupManager(mockK8sClient, ctrl.Log.WithName("EntryPoint").WithName("CleanupManager")),
		StatusUpdater:   status.NewDefaultStatusUpdater(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("StatusUpdater")),
		Validator:       validation.NewDefaultValidator(),
		EventManager:    events.NewEventManager(k8sClient, ctrl.Log.WithName("Kode").WithName("EventManager"), k8sManager.GetScheme(), k8sManager.GetEventRecorderFor("entrypoint-controller")),
	}

	// Set up the Kode controller with the manager
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Set up the EntryPoint controller with the manager
	err = entrypointReconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager
	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	// Create namespace
	By("Creating a namespace")
	namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

	// Create EntryPoint
	By("Creating a subdomain-based EntryPoint")
	entryPointSubdomain = createEntryPoint(entryPointNameSubdomain, namespace.Name, kodev1alpha2.RoutingTypeSubdomain)
	Expect(k8sClient.Create(ctx, entryPointSubdomain)).To(Succeed())

	By("Creating a path-based EntryPoint")
	entryPointPath = createEntryPoint(entryPointNamePath, namespace.Name, kodev1alpha2.RoutingTypePath)
	Expect(k8sClient.Create(ctx, entryPointPath)).To(Succeed())

	// Create PodTemplates
	By("Creating a ClusterPodTemplate with an entrypoint with subdomain routing")
	podTemplateCodeServer = createPodTemplate(podTemplateNameCodeServer, podTemplateImageCodeServer, "code-server", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, podTemplateCodeServer)).To(Succeed())

	By("Creating a ClusterPodTemplate with an entrypoint with path routing")
	podTemplateWebtop = createPodTemplate(podTemplateNameWebtop, podTemplateImageWebtop, "webtop", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, podTemplateWebtop)).To(Succeed())

	By("Creating a ClusterPodTemplate with an entrypoint with subdomain routing")
	podTemplateSubdomain = createPodTemplate("pod-template-subdomain", podTemplateImageCodeServer, "code-server", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, podTemplateSubdomain)).To(Succeed())

	By("Creating a ClusterPodTemplate with an entrypoint with path routing")
	podTemplatePath = createPodTemplate("pod-template-path", podTemplateImageCodeServer, "code-server", entryPointPath.Name, entryPointPath.Namespace)
	Expect(k8sClient.Create(ctx, podTemplatePath)).To(Succeed())
})

var _ = AfterSuite(func() {
	// Cleanup namespace
	Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())

	// Cleanup EntryPoint
	Expect(k8sClient.Delete(ctx, entryPointSubdomain)).To(Succeed())
	Expect(k8sClient.Delete(ctx, entryPointPath)).To(Succeed())

	// Cleanup PodTemplates
	Expect(k8sClient.Delete(ctx, podTemplateCodeServer)).To(Succeed())
	Expect(k8sClient.Delete(ctx, podTemplateWebtop)).To(Succeed())
	Expect(k8sClient.Delete(ctx, podTemplateSubdomain)).To(Succeed())
	Expect(k8sClient.Delete(ctx, podTemplatePath)).To(Succeed())

	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Keep these helper functions
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

func (m *mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	key := fmt.Sprintf("%T/%s/%s", obj, obj.GetNamespace(), obj.GetName())
	m.deletedResources.Store(key, true)
	return nil
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	resourceKey := fmt.Sprintf("%T/%s/%s", obj, key.Namespace, key.Name)
	_, deleted := m.deletedResources.Load(resourceKey)
	if deleted {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	return m.Client.Get(ctx, key, obj, opts...)
}

func createPodTemplate(name, image, templateType string, entryPointName string, entrypointNamespace string) *kodev1alpha2.ClusterPodTemplate {
	port := kodev1alpha2.Port(8000)

	template := &kodev1alpha2.ClusterPodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kodev1alpha2.ClusterPodTemplateSpec{
			PodTemplateSharedSpec: kodev1alpha2.PodTemplateSharedSpec{
				BaseSharedSpec: kodev1alpha2.BaseSharedSpec{
					Port: &port,
				},
				Type:  templateType,
				Image: image,
			},
		},
	}

	if entryPointName != "" {
		entrypointNamespace := kodev1alpha2.Namespace(entrypointNamespace)
		template.Spec.PodTemplateSharedSpec.EntryPointRef = &kodev1alpha2.CrossNamespaceObjectReference{
			Kind:      "EntryPoint",
			Name:      kodev1alpha2.ObjectName(entryPointName),
			Namespace: &entrypointNamespace,
		}
	}

	return template
}

func createEntryPoint(name string, namespaceName string, routingType kodev1alpha2.RoutingType) *kodev1alpha2.EntryPoint {
	return &kodev1alpha2.EntryPoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: kodev1alpha2.EntryPointSpec{
			RoutingType: routingType,
			BaseDomain:  "testing.com",
		},
	}
}

func createKode(name, namespaceName string, podTemplateName string, credentials *kodev1alpha2.CredentialsSpec, storage *kodev1alpha2.KodeStorageSpec) *kodev1alpha2.Kode {
	kode := &kodev1alpha2.Kode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: kodev1alpha2.KodeSpec{
			TemplateRef: kodev1alpha2.CrossNamespaceObjectReference{
				Kind: "ClusterPodTemplate",
				Name: kodev1alpha2.ObjectName(podTemplateName),
			},
		},
	}

	if credentials != nil {
		kode.Spec.Credentials = credentials
	}

	if storage != nil {
		kode.Spec.Storage = storage
	}

	return kode
}
