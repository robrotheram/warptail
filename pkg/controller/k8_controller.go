package controller

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"warptail/pkg/controller/kubernetes/builders"
	v1 "warptail/pkg/controller/kubernetes/v1"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
}

type K8Controller struct {
	CertBuilder         *builders.CertifcationBuilder
	IngressBuilder      *builders.IngressBuilder
	LoadbalancerBuilder *builders.LoadbalancerBuilder
}

func getK8Config() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, err
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user home dir: %v", err)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to find kubernetes config: %v", err)
	}
	return kubeConfig, nil
}

func getCurrentNamespace() (string, error) {
	namespaceFilePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	// Check if the file exists
	if _, err := os.Stat(namespaceFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("namespace file does not exist: %v", err)
	}

	// Read the namespace from the file
	namespaceBytes, err := os.ReadFile(namespaceFilePath)
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %v", err)
	}

	return string(namespaceBytes), nil
}

func NewK8Controller(cfg utils.KubernetesConfig) (*K8Controller, error) {
	config, err := getK8Config()
	if err != nil {
		return nil, err
	}
	if len(cfg.Namespace) == 0 {
		namespace, err := getCurrentNamespace()
		if err != nil {
			cfg.Namespace = namespace
		}
	}
	return &K8Controller{
		CertBuilder:         builders.NewCertifcationBuilder(cfg, config),
		IngressBuilder:      builders.NewIngressBuilder(cfg, config),
		LoadbalancerBuilder: builders.NewLoadbalancerBuilder(cfg, config),
	}, nil
}

func (ctrl *K8Controller) Update(router *router.Router) {
	routes := []utils.RouteConfig{}
	for _, svc := range router.Services {
		for _, route := range svc.Routes {
			routes = append(routes, route.Config())
		}
	}
	if err := ctrl.LoadbalancerBuilder.Create(routes); err != nil {
		log.Printf("K8 Service Error: %v", err)
	}
	if err := ctrl.IngressBuilder.Create(routes); err != nil {
		log.Printf("K8 Ingress Error: %v", err)
	}
	if err := ctrl.CertBuilder.Create(routes); err != nil {
		log.Printf("K8 Certificate Error: %v", err)
	}
}

func StartController(router *router.Router) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   false,
		LeaderElectionID: "90da48b4.warptail.exceptionerror.io",
	})
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: false})))

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&v1.WarpTailServiceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Router: router,
	}).SetupWithManager(mgr); err != nil {
		slog.Error("unable to setup contoller", "error", err.Error())
		setupLog.Error(err, "unable to create controller", "controller", "WarpTailService")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	mgr.Start(context.Background())
}
