package main

import (
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/cleanup"
	entrypointcontroller "github.com/jacero-io/kode-operator/internal/controllers/entrypoint"
	kodecontroller "github.com/jacero-io/kode-operator/internal/controllers/kode"
	"github.com/jacero-io/kode-operator/internal/events"
	"github.com/jacero-io/kode-operator/internal/resource"
	"github.com/jacero-io/kode-operator/internal/status"
	"github.com/jacero-io/kode-operator/internal/template"
	"github.com/jacero-io/kode-operator/internal/validation"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kodev1alpha2.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&logLevel, "log-level", "info", "Log level for development (debug, info, warn, error, dpanic, panic, fatal)")
	flag.Parse()

	var loggerConfig zap.Config
	env := os.Getenv("ENV")
	if env == "production" {
		loggerConfig = zap.NewProductionConfig()
	} else {
		loggerConfig = zap.NewDevelopmentConfig()
		level := zap.NewAtomicLevel()
		err := level.UnmarshalText([]byte(logLevel))
		if err != nil {
			setupLog.Error(err, "unable to parse log level")
			os.Exit(1)
		}
		loggerConfig.Level = level
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		setupLog.Error(err, "unable to build logger")
		os.Exit(1)
	}
	ctrl.SetLogger(zapr.NewLogger(logger))

	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8d0abd8d.jacero.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// printKnownTypes(mgr.GetScheme())

	if err = (&kodecontroller.KodeReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Log:               ctrl.Log.WithName("Kode").WithName("Reconcile"),
		ResourceManager:   resource.NewDefaultResourceManager(mgr.GetClient(), ctrl.Log.WithName("Kode").WithName("ResourceManager"), scheme),
		TemplateManager:   template.NewDefaultTemplateManager(mgr.GetClient(), ctrl.Log.WithName("Kode").WithName("TemplateManager")),
		CleanupManager:    cleanup.NewDefaultCleanupManager(mgr.GetClient(), ctrl.Log.WithName("Kode").WithName("CleanupManager")),
		Validator:         validation.NewDefaultValidator(),
		EventManager:      events.NewEventManager(mgr.GetClient(), ctrl.Log.WithName("Kode").WithName("EventManager"), mgr.GetScheme(), mgr.GetEventRecorderFor("kode-controller")),
		IsTestEnvironment: false,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kode")
		os.Exit(1)
	}

	if err = (&entrypointcontroller.EntryPointReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Log:             ctrl.Log.WithName("EntryPoint").WithName("Reconcile"),
		ResourceManager: resource.NewDefaultResourceManager(mgr.GetClient(), ctrl.Log.WithName("EntryPoint").WithName("ResourceManager"), scheme),
		TemplateManager: template.NewDefaultTemplateManager(mgr.GetClient(), ctrl.Log.WithName("EntryPoint").WithName("TemplateManager")),
		CleanupManager:  cleanup.NewDefaultCleanupManager(mgr.GetClient(), ctrl.Log.WithName("EntryPoint").WithName("CleanupManager")),
		StatusUpdater:   status.NewDefaultStatusUpdater(mgr.GetClient(), ctrl.Log.WithName("EntryPoint").WithName("StatusUpdater")),
		Validator:       validation.NewDefaultValidator(),
		EventManager:    events.NewEventManager(mgr.GetClient(), ctrl.Log.WithName("EntryPoint").WithName("EventManager"), mgr.GetScheme(), mgr.GetEventRecorderFor("entrypoint-controller")),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EntryPoint")
		os.Exit(1)
		for gvk := range mgr.GetScheme().AllKnownTypes() {
			setupLog.V(1).Info("Known type in manager scheme", "gvk", gvk)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func printKnownTypes(scheme *runtime.Scheme) {
	for gvk := range scheme.AllKnownTypes() {
		setupLog.V(1).Info("Known type in scheme", "gvk", gvk)
	}
}
