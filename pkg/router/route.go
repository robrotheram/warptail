package router

import (
	"fmt"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

type RouterStatus string

const (
	STARTING = RouterStatus("Starting")
	RUNNING  = RouterStatus("Running")
	STOPPING = RouterStatus("Stopping")
	STOPPED  = RouterStatus("Stopped")
)

type Route interface {
	Start() error
	Stop() error
	Update(utils.RouteConfig) error
	Config() utils.RouteConfig
	Status() RouterStatus
	Stats() utils.TimeSeriesData
	Ping() time.Duration
}

func NewRoute(config utils.RouteConfig, ts *tsnet.Server) (Route, error) {
	client, err := ts.LocalClient()
	if err != nil {
		return nil, err
	}
	switch config.Type {
	case utils.UDP:
		return NewNetworkRoute(config, client), nil
	case utils.TCP:
		return NewNetworkRoute(config, client), nil
	case utils.HTTP:
		return NewHTTPRoute(config, ts), nil
	case utils.HTTPS:
		return NewHTTPRoute(config, ts), nil
	default:
		return nil, fmt.Errorf("no handler for type %s", config.Type)
	}
}
