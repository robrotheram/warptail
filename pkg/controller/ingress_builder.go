package controller

import (
	"context"
	"fmt"
	"warptail/pkg/utils"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctrl *K8Controller) buildIngress(routes []utils.RouteConfig) networkingv1.Ingress {
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
		if route.Type != utils.HTTP {
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
		tlsRule := networkingv1.IngressTLS{
			Hosts:      []string{route.Domain},
			SecretName: ctrl.Certificate.SecretName,
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
		ingress.Spec.TLS = append(ingress.Spec.TLS, tlsRule)
	}
	return ingress
}

func (ctrl *K8Controller) getIngress() (*networkingv1.Ingress, error) {
	return ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Get(context.TODO(), ctrl.Ingress.Name, metav1.GetOptions{})
}

func (ctrl *K8Controller) deleteIngress() error {
	if _, err := ctrl.getIngress(); err == nil {
		return nil
	}
	return ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Delete(context.TODO(), ctrl.Ingress.Name, metav1.DeleteOptions{})
}

func (ctrl *K8Controller) createIngress(routes []utils.RouteConfig) error {
	ingress := ctrl.buildIngress(routes)
	if len(ingress.Spec.Rules) == 0 {
		fmt.Println("Ingress exists, deleting it...")
		return ctrl.deleteIngress()
	}
	existingIngress, err := ctrl.getIngress()
	if err != nil {
		fmt.Println("Ingress does not exist, creating a new one...")
		_, err := ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Create(context.TODO(), &ingress, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Ingress: %v", err)
		}
		return nil
	}
	fmt.Println("Ingress exists, updating it...")
	existingIngress.Spec = ingress.Spec
	_, err = ctrl.k8Client.NetworkingV1().Ingresses(ctrl.Namespace).Update(context.TODO(), existingIngress, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Ingress: %v", err)
	}
	return nil
}
