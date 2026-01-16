package builders

import (
	"context"
	"fmt"
	"warptail/pkg/utils"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LoadbalancerBuilder struct {
	Namespace    string
	Loadbalancer utils.Loadbalancer
	k8Client     *kubernetes.Clientset
	logger       logr.Logger
}

func NewLoadbalancerBuilder(config utils.KubernetesConfig, k8cfg *rest.Config) *LoadbalancerBuilder {
	k8Client, err := kubernetes.NewForConfig(k8cfg)
	if err != nil {
		return nil
	}
	return &LoadbalancerBuilder{
		Namespace:    config.Namespace,
		Loadbalancer: config.Loadbalancer,
		k8Client:     k8Client,
		logger:       utils.Logger,
	}
}

func (ctrl *LoadbalancerBuilder) build(routes []utils.RouteConfig, existingPorts []corev1.ServicePort) corev1.Service {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.Loadbalancer.Name,
			Namespace: ctrl.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "warptail",
			},
			Ports: []corev1.ServicePort{},
		},
	}

	// Build a map of existing ports to preserve nodePort assignments
	existingPortMap := make(map[string]int32)
	for _, p := range existingPorts {
		key := fmt.Sprintf("%s-%d", p.Protocol, p.Port)
		existingPortMap[key] = p.NodePort
	}

	for _, route := range routes {
		if route.Type != utils.TCP && route.Type != utils.UDP {
			continue
		}
		protocol := corev1.ProtocolTCP
		if route.Type == utils.UDP {
			protocol = corev1.ProtocolUDP
		}
		port := corev1.ServicePort{
			Name:       fmt.Sprintf("%s-%d", string(route.Type), route.Port),
			Port:       int32(route.Port),
			TargetPort: intstr.FromInt(route.Port),
			Protocol:   protocol,
		}
		// Preserve existing nodePort if available to avoid conflicts
		key := fmt.Sprintf("%s-%d", protocol, route.Port)
		if nodePort, ok := existingPortMap[key]; ok {
			port.NodePort = nodePort
		}
		service.Spec.Ports = append(service.Spec.Ports, port)
	}
	return service
}

func (ctrl *LoadbalancerBuilder) get() (*corev1.Service, error) {
	return ctrl.k8Client.CoreV1().Services(ctrl.Namespace).Get(context.TODO(), ctrl.Loadbalancer.Name, metav1.GetOptions{})
}

func (ctrl *LoadbalancerBuilder) delete() error {
	if _, err := ctrl.get(); err != nil {
		return nil
	}
	return ctrl.k8Client.CoreV1().Services(ctrl.Namespace).Delete(context.TODO(), ctrl.Loadbalancer.Name, metav1.DeleteOptions{})
}

func (ctrl *LoadbalancerBuilder) Create(routes []utils.RouteConfig) error {
	existingService, err := ctrl.get()
	var existingPorts []corev1.ServicePort
	if err == nil {
		existingPorts = existingService.Spec.Ports
	}

	service := ctrl.build(routes, existingPorts)
	if len(service.Spec.Ports) == 0 {
		ctrl.logger.Info("Service exists, deleting it...")
		return ctrl.delete()
	}

	if err != nil {
		ctrl.logger.Info("Service does not exist, creating a new one...")
		_, err := ctrl.k8Client.CoreV1().Services(ctrl.Namespace).Create(context.TODO(), &service, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Service: %v", err)
		}
		return nil
	}
	ctrl.logger.Info("Service exists, updating it...")
	existingService.Spec = service.Spec
	_, err = ctrl.k8Client.CoreV1().Services(ctrl.Namespace).Update(context.TODO(), existingService, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Service: %v", err)
	}
	return nil
}
