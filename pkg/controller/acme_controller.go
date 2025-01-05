package controller

import (
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"golang.org/x/crypto/acme/autocert"
)

type ACMEController struct {
	manager *autocert.Manager
	domains []string
}

func NewACMEContoller(manager *autocert.Manager, cfg utils.ApplicationConfig) *ACMEController {
	return &ACMEController{
		manager: manager,
		domains: []string{cfg.Acme.PortalDomain},
	}
}

func (ctrl *ACMEController) Update(router *router.Router) {
	domains := []string{}
	for _, svc := range router.Services {
		for _, route := range svc.Routes {
			cfg := route.Config()
			if cfg.Type == utils.HTTPS {
				domains = append(domains, cfg.Domain)
			}
		}
	}
	ctrl.manager.HostPolicy = autocert.HostWhitelist(domains...)
}
