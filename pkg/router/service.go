package router

import (
	"fmt"
	"time"
	"warptail/pkg/utils"

	"github.com/gosimple/slug"
	"tailscale.com/tsnet"
)

func containsRoute(routes []Route, config utils.RouteConfig) (Route, error) {
	for _, route := range routes {
		if utils.RouteComparison(route.Config(), config) {
			return route, nil
		}
	}
	return nil, fmt.Errorf("route not found")
}

type Service struct {
	Id      string
	Name    string
	Enabled bool
	Routes  []Route
}

func NewService(config utils.ServiceConfig, server *tsnet.Server) *Service {
	routes := []Route{}
	for _, cfg := range config.Routes {
		if route, err := NewRoute(cfg, server); err == nil {
			routes = append(routes, route)
		}
	}
	return &Service{
		Id:      slug.Make(config.Name),
		Name:    config.Name,
		Enabled: config.Enabled,
		Routes:  routes,
	}
}

func (svc *Service) Update(config utils.ServiceConfig, server *tsnet.Server) *utils.RouterError {
	if svc.Name != config.Name {
		svc.Name = config.Name
		svc.Id = slug.Make(config.Name)
	}

	svc.Enabled = config.Enabled
	if !svc.Enabled {
		svc.Stop()
	}

	existingRoutes := []Route{}
	newRoutes := []utils.RouteConfig{}
	for _, cfg := range config.Routes {
		if route, err := containsRoute(svc.Routes, cfg); err == nil {
			route.Update(cfg)
			existingRoutes = append(existingRoutes, route)
		} else {
			newRoutes = append(newRoutes, cfg)
		}
	}
	svc.pruneRoutes(existingRoutes)
	svc.updateNewRoutes(existingRoutes, newRoutes, server)
	return nil
}

func (svc *Service) pruneRoutes(existingRoutes []Route) {
	for _, route := range svc.Routes {
		if _, err := containsRoute(existingRoutes, route.Config()); err != nil {
			if svc.Enabled {
				route.Stop()
			}
		}
	}
}

func (svc *Service) updateNewRoutes(existingRoutes []Route, newRoutes []utils.RouteConfig, server *tsnet.Server) {
	for _, cfg := range newRoutes {
		if route, err := NewRoute(cfg, server); err == nil {
			if svc.Enabled {
				route.Start()
			}
			existingRoutes = append(existingRoutes, route)
		}
	}
	svc.Routes = existingRoutes
	if svc.Enabled {
		for _, route := range svc.Routes {
			if route.Status() != RUNNING {
				route.Start()
			}
		}
	}
}

type ServiceStatus struct {
	Id      string        `json:"id"`
	Name    string        `json:"name"`
	Enabled bool          `json:"enabled"`
	Routes  []RouteStatus `json:"routes"`
	Latency int64         `json:"latency,omitempty"`
}

type RouteStatus struct {
	utils.RouteConfig
	Status  RouterStatus         `json:"status,omitempty"`
	Latency int64                `json:"latency,omitempty"`
	Stats   utils.TimeSeriesData `json:"stats,omitempty"`
}

func (svc *Service) Status(full bool) ServiceStatus {
	status := ServiceStatus{
		Id:      svc.Id,
		Name:    svc.Name,
		Enabled: svc.Enabled,
		Routes:  []RouteStatus{},
	}
	totalLatency := time.Duration(0)

	for _, routes := range svc.Routes {
		rStatus := RouteStatus{
			RouteConfig: routes.Config(),
			Status:      routes.Status(),
			Latency:     routes.Ping().Nanoseconds(),
		}
		if full {
			rStatus.Stats = routes.Stats()
		}
		totalLatency += routes.Ping()
		status.Routes = append(status.Routes, rStatus)
	}
	if len(svc.Routes) > 0 {
		status.Latency = totalLatency.Nanoseconds() / int64(len(svc.Routes))
	}
	return status
}

func (svc *Service) Stop() {
	for _, route := range svc.Routes {
		route.Stop()
	}
	svc.Enabled = false
}

func (svc *Service) Start() {
	for _, route := range svc.Routes {
		route.Start()
	}
	svc.Enabled = true
}
