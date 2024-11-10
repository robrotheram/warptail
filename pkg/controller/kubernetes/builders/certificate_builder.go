package builders

import (
	"context"
	"fmt"
	"warptail/pkg/utils"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type CertifcationBuilder struct {
	Namespace   string
	Certificate utils.Certificate
	cmclient    *cmclientset.Clientset
	logger      logr.Logger
}

func NewCertifcationBuilder(config utils.KubernetesConfig, k8cfg *rest.Config) *CertifcationBuilder {
	cmclient, err := cmclientset.NewForConfig(k8cfg)
	if err != nil {
		return nil
	}
	return &CertifcationBuilder{
		Namespace:   config.Namespace,
		Certificate: config.Certificate,
		cmclient:    cmclient,
		logger:      utils.Logger,
	}
}

func (ctrl *CertifcationBuilder) build(routes []utils.RouteConfig) certmanagerv1.Certificate {
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
func (ctrl *CertifcationBuilder) get() (*certmanagerv1.Certificate, error) {
	return ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Get(context.TODO(), ctrl.Certificate.Name, metav1.GetOptions{})
}

func (ctrl *CertifcationBuilder) delete() error {
	if _, err := ctrl.get(); err != nil {
		return nil
	}
	return ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Delete(context.TODO(), ctrl.Certificate.Name, metav1.DeleteOptions{})
}

func (ctrl *CertifcationBuilder) Create(routes []utils.RouteConfig) error {
	certificate := ctrl.build(routes)
	if len(certificate.Spec.DNSNames) == 0 {
		ctrl.logger.Info("certificate exists, deleting it...")
		return ctrl.delete()
	}
	existingCertificate, err := ctrl.get()
	if err != nil {
		ctrl.logger.Info("Certficate does not exist, creating a new one...")
		_, err := ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Create(context.TODO(), &certificate, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Certficate: %v", err)
		}
		return nil
	}
	ctrl.logger.Info("Certficate exists, updating it...")
	existingCertificate.Spec = certificate.Spec
	_, err = ctrl.cmclient.CertmanagerV1().Certificates(ctrl.Namespace).Update(context.TODO(), existingCertificate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Ingress: %v", err)
	}
	return nil
}
