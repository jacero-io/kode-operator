//go:build integration

// internal/controller/suite_test.go

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
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/emil-jacero/kode-operator/internal/cleanup"
	"github.com/emil-jacero/kode-operator/internal/repository"
	"github.com/emil-jacero/kode-operator/internal/resource"
	"github.com/emil-jacero/kode-operator/internal/status"
	"github.com/emil-jacero/kode-operator/internal/template"
	"github.com/emil-jacero/kode-operator/internal/validation"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	ctx        context.Context
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	k8sClient  client.Client
	k8sManager ctrl.Manager
	reconciler *KodeReconciler
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kode Controller Suite")
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

	err = kodev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	reconciler = &KodeReconciler{
		Client:          k8sClient,
		Scheme:          k8sManager.GetScheme(),
		Log:             ctrl.Log.WithName("controllers").WithName("Kode").WithName("Reconcile"),
		Repo:            repository.NewDefaultRepository(k8sClient),
		ResourceManager: resource.NewDefaultResourceManager(k8sClient, ctrl.Log.WithName("controllers").WithName("Kode").WithName("ResourceManager")),
		TemplateManager: template.NewDefaultTemplateManager(k8sClient, ctrl.Log.WithName("controllers").WithName("Kode").WithName("TemplateManager")),
		CleanupManager:  cleanup.NewDefaultCleanupManager(k8sClient, ctrl.Log.WithName("controllers").WithName("Kode").WithName("CleanupManager")),
		StatusUpdater:   status.NewDefaultStatusUpdater(k8sClient, ctrl.Log.WithName("controllers").WithName("Kode").WithName("StatusUpdater")),
		Validator:       validation.NewDefaultValidator(),
	}

	// Set up the controller with the manager
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager
	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
