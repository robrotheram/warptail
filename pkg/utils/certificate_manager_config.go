package utils

import (
	"fmt"

	"golang.org/x/crypto/acme/autocert"
)

type CertificateManagerConfig struct {
	Enabled         bool   `yaml:"enabled"`
	SslPort         int    `yaml:"ssl_port"`
	CertificatesDir string `yaml:"certificates_dir"`
	PortalDomain    string `yaml:"portal_domain"`
}

func (app *CertificateManagerConfig) GetSSLAddr() string {
	return fmt.Sprintf(":%d", app.SslPort)
}

func (app *CertificateManagerConfig) ACMEManager() *autocert.Manager {
	return &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(app.CertificatesDir),
	}
}
