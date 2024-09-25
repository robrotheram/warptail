package kubeController

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"warptail/pkg/utils"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	cmclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
)

const SecretName = "warptail-certificate"

type K8Controller struct {
	k8Client     *kubernetes.Clientset
	cmclient     *cmclientset.Clientset
	namespace    string
	serviceName  string
	ingressClass string
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

func NewK8Controller(cfg utils.K8Config) (*K8Controller, error) {
	config, err := getK8Config()
	if err != nil {
		return nil, err
	}
	k8client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	cmclient, err := cmclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespace, err := getCurrentNamespace()
	if err != nil {
		namespace = cfg.Namespace
	}

	return &K8Controller{
		k8Client:     k8client,
		cmclient:     cmclient,
		namespace:    namespace,
		serviceName:  cfg.ServiceName,
		ingressClass: cfg.IngressClass,
	}, nil
}

func (ctrl *K8Controller) Update(routes []utils.RouteConfig) {
	if err := ctrl.createService(routes); err != nil {
		log.Printf("K8 Service Error: %v", err)
	}
	if err := ctrl.createIngress(routes); err != nil {
		log.Printf("K8 Ingress Error: %v", err)
	}
	if err := ctrl.createCertificate(routes); err != nil {
		log.Printf("K8 Certificate Error: %v", err)
	}
}
