package utils

import (
	"fmt"
)

type ApplicationConfig struct {
	Port     int    `yaml:"port"`
	SiteName string `yaml:"site_name,omitempty"`
	SiteLogo string `yaml:"site_logo,omitempty"`
}

func (app *ApplicationConfig) GetHTTPAddr() string {
	return fmt.Sprintf(":%d", app.Port)
}

// func (app *ApplicationConfig) GetSSLAddr() string {
// 	return fmt.Sprintf(":%d", app.Acme.SslPort)
// }

// func (app *ApplicationConfig) UseHTTPS() bool {
// 	return !IsEmptyStruct(app.Acme) && app.Acme.Enabled
// }

// func (app *ApplicationConfig) ACMEManager() *autocert.Manager {
// 	return &autocert.Manager{
// 		Prompt: autocert.AcceptTOS,
// 		Cache:  autocert.DirCache(app.Acme.CertificatesDir),
// 	}
// }
