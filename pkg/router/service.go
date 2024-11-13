package router

import (
	"fmt"
	"sync"
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

func (svc *Service) Update(config utils.ServiceConfig, server *tsnet.Server) *RouterError {
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
	Id      string               `json:"id"`
	Name    string               `json:"name"`
	Enabled bool                 `json:"enabled"`
	Routes  []RouteStatus        `json:"routes"`
	Latency time.Duration        `json:"latency,omitempty"`
	Stats   utils.TimeSeriesData `json:"stats,omitempty"`
}

type RouteStatus struct {
	utils.RouteConfig
	Status  RouterStatus  `json:"status,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

func (svc *Service) Status(full bool) ServiceStatus {
	status := ServiceStatus{
		Id:      svc.Id,
		Name:    svc.Name,
		Enabled: svc.Enabled,
		Routes:  []RouteStatus{},
		Latency: svc.HeartBeat(),
	}
	if full {
		status.Stats = utils.TimeSeriesData{
			Points: []utils.DataPoint{},
			Total:  utils.ProxyStats{},
		}
	}
	for _, routes := range svc.Routes {
		rStatus := RouteStatus{
			RouteConfig: routes.Config(),
			Status:      routes.Status(),
		}
		if full {
			rStatus.Latency = routes.Ping()
		}
		status.Stats = utils.CombineTimeSeriesData(status.Stats, routes.Stats())
		status.Routes = append(status.Routes, rStatus)
	}
	return status
}

func (svc *Service) HeartBeat() time.Duration {
	latency := time.Nanosecond
	wg := sync.WaitGroup{}
	for _, route := range svc.Routes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			latency += route.Ping()
		}()
	}
	wg.Wait()
	if latency == 0 || time.Duration(len(svc.Routes)) == 0 {
		return latency
	}
	latency = latency / time.Duration(len(svc.Routes))
	return latency
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
