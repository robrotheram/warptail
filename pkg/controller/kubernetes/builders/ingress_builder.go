package builders

import (
	"context"
	"fmt"
	"warptail/pkg/utils"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type IngressBuilder struct {
	Namespace   string
	Ingress     utils.Ingress
	Certificate utils.Certificate
	k8Client    *kubernetes.Clientset
	logger      logr.Logger
}

func NewIngressBuilder(config utils.KubernetesConfig, k8cfg *rest.Config) *IngressBuilder {
	k8Client, err := kubernetes.NewForConfig(k8cfg)
	if err != nil {
		return nil
	}
	return &IngressBuilder{
		Namespace:   config.Namespace,
		Certificate: config.Certificate,
		Ingress:     config.Ingress,
		k8Client:    k8Client,
		logger:      utils.Logger,
	}
}

func (ctrl *IngressBuilder) build(routes []utils.RouteConfig) networkingv1.Ingress {
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.Ingress.Name,
			Namespace: ctrl.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ctrl.Ingress.Class,
			Rules:            []networkingv1.IngressRule{},
			TLS:              []networkingv1.IngressTLS{},
		},
	}

	for _, route := range routes {
		if route.Type != utils.HTTP && route.Type != utils.HTTPS {
			continue
		}
		rule := networkingv1.IngressRule{
			Host: route.Domain,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: func() *networkingv1.PathType { pathType := networkingv1.PathTypePrefix; return &pathType }(),
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: ctrl.Ingress.Service,
									Port: networkingv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						},
					},
				},
			},
		}
		if route.Type == utils.HTTPS {
			tlsRule := networkingv1.IngressTLS{
				Hosts:      []string{route.Domain},
				SecretName: ctrl.Certificate.SecretName,
			}
			ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
			ingress.Spec.TLS = append(ingress.Spec.TLS, tlsRule)
		}
	}
	return ingress
}

func (ctrl *IngressBuilder) get() (*networkingv1.Ingress, error) {
	return ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Get(context.TODO(), ctrl.Ingress.Name, metav1.GetOptions{})
}

func (ctrl *IngressBuilder) delete() error {
	if _, err := ctrl.get(); err != nil {
		return nil
	}
	return ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Delete(context.TODO(), ctrl.Ingress.Name, metav1.DeleteOptions{})
}

func (ctrl *IngressBuilder) Create(routes []utils.RouteConfig) error {
	ingress := ctrl.build(routes)
	if len(ingress.Spec.Rules) == 0 {
		ctrl.logger.Info("Ingress exists, deleting it...")
		return ctrl.delete()
	}
	existingIngress, err := ctrl.get()
	if err != nil {
		ctrl.logger.Info("Ingress does not exist, creating a new one...")
		_, err := ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Create(context.TODO(), &ingress, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Ingress: %v", err)
		}
		return nil
	}
	ctrl.logger.Info("Ingress exists, updating it...")
	existingIngress.Spec = ingress.Spec
	_, err = ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Update(context.TODO(), existingIngress, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Ingress: %v", err)
	}
	return nil
}
