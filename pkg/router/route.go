package router

import (
	"fmt"
	"warptail/pkg/utils"
)

func NewRoute(config utils.RouteConfig, ts Backend) (Route, error) {
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
