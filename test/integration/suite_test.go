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

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	controller "github.com/jacero-io/kode-operator/internal/controllers/kode"
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
	timeout  = time.Second * 60
	interval = time.Second * 1
)

var (
	cfg           *rest.Config
	cancel        context.CancelFunc
	testEnv       *envtest.Environment
	k8sClient     client.Client
	mockK8sClient *mockClient
	k8sManager    ctrl.Manager
	reconciler    *controller.KodeReconciler

	ctx                   context.Context
	namespace             *corev1.Namespace
	podTemplateCodeServer *kodev1alpha2.ClusterPodTemplate
	podTemplateWebtop     *kodev1alpha2.ClusterPodTemplate
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}

type mockClient struct {
	client.Client
	deletedResources sync.Map
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	ctx, cancel = context.WithCancel(context.Background())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kodev1alpha2.AddToScheme(scheme.Scheme)
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

	reconciler = &controller.KodeReconciler{
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

	// Set up the controller with the manager
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager
	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	// Create namespace
	namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

	// // Create EntryPoint
	// entryPoint = &kodev1alpha2.EntryPoint{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "test-entrypoint",
	// 		Namespace: namespace.Name,
	// 	},
	// 	Spec: kodev1alpha2.EntryPointSpec{
	// 		RoutingType: "subdomain",
	// 		BaseDomain:  "kode.example.com",
	// 	},
	// }
	// Expect(k8sClient.Create(ctx, entryPoint)).To(Succeed())

	// Create PodTemplates
	podTemplateCodeServer = createPodTemplate(podTemplateNameCodeServer, podTemplateImageCodeServer, "code-server", "", "")
	podTemplateWebtop = createPodTemplate(podTemplateNameWebtop, podTemplateImageWebtop, "webtop", "", "")

	Expect(k8sClient.Create(ctx, podTemplateCodeServer)).To(Succeed())
	Expect(k8sClient.Create(ctx, podTemplateWebtop)).To(Succeed())
})

var _ = AfterSuite(func() {
	// Cleanup resources
	Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	Expect(k8sClient.Delete(ctx, podTemplateCodeServer)).To(Succeed())
	Expect(k8sClient.Delete(ctx, podTemplateWebtop)).To(Succeed())
	// Expect(k8sClient.Delete(ctx, entryPoint)).To(Succeed())

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
