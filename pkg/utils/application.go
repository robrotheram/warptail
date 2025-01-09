package utils

import (
	"fmt"

	"golang.org/x/crypto/acme/autocert"
)

type ApplicationConfig struct {
	Port           int                  `yaml:"port"`
	Authentication AuthenticationConfig `yaml:"authentication"`
	SiteName       string               `yaml:"site_name,omitempty"`
	SiteLogo       string               `yaml:"site_logo,omitempty"`
	Acme           struct {
		Enabled         bool   `yaml:"enabled"`
		SslPort         int    `yaml:"ssl_port"`
		CertificatesDir string `yaml:"certificates_dir"`
		PortalDomain    string `yaml:"portal_domain"`
	} `yaml:"acme"`
}

func (app *ApplicationConfig) GetHTTPAddr() string {
	return fmt.Sprintf(":%d", app.Port)
}

func (app *ApplicationConfig) GetSSLAddr() string {
	return fmt.Sprintf(":%d", app.Acme.SslPort)
}

func (app *ApplicationConfig) UseHTTPS() bool {
	return !IsEmptyStruct(app.Acme) && app.Acme.Enabled
}

func (app *ApplicationConfig) ACMEManager() *autocert.Manager {
	return &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(app.Acme.CertificatesDir),
	}
}
