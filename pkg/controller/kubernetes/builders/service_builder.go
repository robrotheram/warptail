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

func (ctrl *LoadbalancerBuilder) build(routes []utils.RouteConfig) corev1.Service {
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

	for _, route := range routes {
		if route.Type != utils.TCP && route.Type != utils.UDP {
			continue
		}
		port := corev1.ServicePort{
			Port:       int32(route.Port),
			TargetPort: intstr.FromInt(route.Port),
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
	service := ctrl.build(routes)
	if len(service.Spec.Ports) == 0 {
		ctrl.logger.Info("Service exists, deleting it...")
		return ctrl.delete()
	}

	existingService, err := ctrl.get()
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
