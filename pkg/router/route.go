package router

import (
	"fmt"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

func NewRoute(config utils.RouteConfig, ts *tsnet.Server) (Route, error) {
	switch config.Type {
	case utils.UDP:
		return NewUDPRoute(config, ts), nil
	case utils.TCP:
		return NewTCPRoute(config, ts), nil
	case utils.HTTP:
		return NewHTTPRoute(config, ts), nil
	case utils.HTTPS:
		return NewHTTPRoute(config, ts), nil
	default:
		return nil, fmt.Errorf("no handler for type %s", config.Type)
	}
}
