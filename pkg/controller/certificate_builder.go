package controller

import (
	"context"
	"fmt"
	"warptail/pkg/utils"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctrl *K8Controller) buildCertificate(routes []utils.RouteConfig) certmanagerv1.Certificate {
	DNSNames := []string{}
	for _, route := range routes {
		DNSNames = append(DNSNames, route.Domain)
	}

	return certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrl.Certificate.Name,
			Namespace: ctrl.Namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: ctrl.Certificate.SecretName,
			DNSNames:   DNSNames,
			IssuerRef: cmmeta.ObjectReference{
				Name: "letsencrypt-prod",
				Kind: "ClusterIssuer",
			},
		},
	}
}
func (ctrl *K8Controller) getCertificate() (*certmanagerv1.Certificate, error) {
	return ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Get(context.TODO(), ctrl.Certificate.Name, metav1.GetOptions{})
}

func (ctrl *K8Controller) deleteCertificate() error {
	if _, err := ctrl.getCertificate(); err == nil {
		return nil
	}
	return ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Delete(context.TODO(), ctrl.Certificate.Name, metav1.DeleteOptions{})
}

func (ctrl *K8Controller) createCertificate(routes []utils.RouteConfig) error {
	certificate := ctrl.buildCertificate(routes)
	if len(certificate.Spec.DNSNames) == 0 {
		fmt.Println("Certificate exists, deleting it...")
		return ctrl.deleteCertificate()
	}
	existingCertificate, err := ctrl.getCertificate()
	if err != nil {
		fmt.Println("Certficate does not exist, creating a new one...")
		_, err := ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Create(context.TODO(), &certificate, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Certficate: %v", err)
		}
		return nil
	}
	fmt.Println("Certficate exists, updating it...")
	existingCertificate.Spec = certificate.Spec
	_, err = ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Update(context.TODO(), existingCertificate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Ingress: %v", err)
	}
	return nil
}
