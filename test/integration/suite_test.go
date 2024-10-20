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
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	"github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/template"
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

	containerTemplateNameCodeServer = "containertemplate-codeserver"
	containerTemplateNameWebtop     = "containertemplate-webtop"

	containerTemplateImageCodeServer = "linuxserver/code-server:latest"
	containerTemplateImageWebtop     = "linuxserver/webtop:debian-xfce"
)

var (
	cfg                  *rest.Config
	cancel               context.CancelFunc
	testEnv              *envtest.Environment
	k8sClient            client.Client
	k8sManager           ctrl.Manager
	reconciler           *kodectrl.KodeReconciler
	entrypointReconciler *entrypointctrl.EntryPointReconciler

	ctx                         context.Context
	namespace                   *corev1.Namespace
	containerTemplateCodeServer *kodev1alpha2.ClusterContainerTemplate
	containerTemplateWebtop     *kodev1alpha2.ClusterContainerTemplate

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
	containerTemplateSubdomain *kodev1alpha2.ClusterContainerTemplate
	containerTemplatePath      *kodev1alpha2.ClusterContainerTemplate
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
	err = storagev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// Set up StorageClass
	err = setupStorageClass(ctx, k8sClient)
	Expect(err).ToNot(HaveOccurred())

	reconciler = &kodectrl.KodeReconciler{
		Client:            k8sClient,
		Scheme:            k8sManager.GetScheme(),
		Log:               ctrl.Log.WithName("Kode").WithName("Reconcile"),
		Resource:          resource.NewDefaultResourceManager(k8sClient, ctrl.Log.WithName("Kode").WithName("ResourceManager"), k8sManager.GetScheme()),
		Template:          template.NewDefaultTemplateManager(k8sClient, ctrl.Log.WithName("Kode").WithName("TemplateManager")),
		CleanupManager:    cleanup.NewDefaultCleanupManager(k8sClient, ctrl.Log.WithName("Kode").WithName("CleanupManager")),
		EventManager:      event.NewEventManager(k8sClient, ctrl.Log.WithName("Kode").WithName("EventManager"), k8sManager.GetScheme(), k8sManager.GetEventRecorderFor("kode-controller")),
		IsTestEnvironment: true,
	}

	entrypointReconciler = &entrypointctrl.EntryPointReconciler{
		Client:            k8sClient,
		Scheme:            k8sManager.GetScheme(),
		Log:               ctrl.Log.WithName("EntryPoint").WithName("Reconcile"),
		Resource:          resource.NewDefaultResourceManager(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("ResourceManager"), k8sManager.GetScheme()),
		Template:          template.NewDefaultTemplateManager(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("TemplateManager")),
		CleanupManager:    cleanup.NewDefaultCleanupManager(k8sClient, ctrl.Log.WithName("EntryPoint").WithName("CleanupManager")),
		EventManager:      event.NewEventManager(k8sClient, ctrl.Log.WithName("Kode").WithName("EventManager"), k8sManager.GetScheme(), k8sManager.GetEventRecorderFor("entrypoint-controller")),
		IsTestEnvironment: true,
	}

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = entrypointReconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	// Create namespace
	namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceNamespace}}
	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

	// Create EntryPoints
	entryPointSubdomain = createEntryPoint(entryPointNameSubdomain, namespace.Name, kodev1alpha2.RoutingTypeSubdomain)
	Expect(k8sClient.Create(ctx, entryPointSubdomain)).To(Succeed())

	entryPointPath = createEntryPoint(entryPointNamePath, namespace.Name, kodev1alpha2.RoutingTypePath)
	Expect(k8sClient.Create(ctx, entryPointPath)).To(Succeed())

	// Create ContainerTemplates
	containerTemplateCodeServer = createContainerTemplate(containerTemplateNameCodeServer, containerTemplateImageCodeServer, "code-server", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, containerTemplateCodeServer)).To(Succeed())

	containerTemplateWebtop = createContainerTemplate(containerTemplateNameWebtop, containerTemplateImageWebtop, "webtop", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, containerTemplateWebtop)).To(Succeed())

	containerTemplateSubdomain = createContainerTemplate("pod-template-subdomain", containerTemplateImageCodeServer, "code-server", entryPointSubdomain.Name, entryPointSubdomain.Namespace)
	Expect(k8sClient.Create(ctx, containerTemplateSubdomain)).To(Succeed())

	containerTemplatePath = createContainerTemplate("pod-template-path", containerTemplateImageCodeServer, "code-server", entryPointPath.Name, entryPointPath.Namespace)
	Expect(k8sClient.Create(ctx, containerTemplatePath)).To(Succeed())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

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

func setupStorageClass(ctx context.Context, k8sClient client.Client) error {
	volumeBindingMode := storagev1.VolumeBindingImmediate
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "standard",
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		VolumeBindingMode: &volumeBindingMode,
	}

	err := k8sClient.Create(ctx, storageClass)
	if err != nil {
		return err
	}

	return nil
}

func createContainerTemplate(name, image, templateType string, entryPointName string, entrypointNamespace string) *kodev1alpha2.ClusterContainerTemplate {
	port := kodev1alpha2.Port(8000)

	template := &kodev1alpha2.ClusterContainerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kodev1alpha2.ClusterContainerTemplateSpec{
			ContainerTemplateSharedSpec: kodev1alpha2.ContainerTemplateSharedSpec{
				CommonSpec: kodev1alpha2.CommonSpec{
					Port: port,
				},
				Type:  templateType,
				Image: image,
			},
		},
	}

	if entryPointName != "" {
		entrypointNamespace := kodev1alpha2.Namespace(entrypointNamespace)
		template.Spec.ContainerTemplateSharedSpec.EntryPointRef = &kodev1alpha2.CrossNamespaceObjectReference{
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

func createKode(name, namespaceName string, containerTemplateName string, credentials *kodev1alpha2.CredentialsSpec, storage *kodev1alpha2.KodeStorageSpec) *kodev1alpha2.Kode {
	kode := &kodev1alpha2.Kode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: kodev1alpha2.KodeSpec{
			TemplateRef: kodev1alpha2.CrossNamespaceObjectReference{
				Kind: "ClusterContainerTemplate",
				Name: kodev1alpha2.ObjectName(containerTemplateName),
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
