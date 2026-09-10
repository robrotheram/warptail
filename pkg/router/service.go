package router

import (
	"fmt"
	"time"
	"warptail/pkg/utils"

	"github.com/gosimple/slug"
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

func NewService(config utils.ServiceConfig, backend Backend) (*Service, error) {
	routes := []Route{}
	for _, cfg := range config.Routes {
		route, err := NewRoute(cfg, backend)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return &Service{Id: slug.Make(config.Name), Name: config.Name, Enabled: config.Enabled, Routes: routes}, nil
}

func (svc *Service) Update(config utils.ServiceConfig, backend Backend) (result *utils.RouterError) {
	// Construct all new routes before changing the running service.
	routes := []Route{}
	for _, cfg := range config.Routes {
		route, err := containsRoute(svc.Routes, cfg)
		if err != nil {
			route, err = NewRoute(cfg, backend)
			if err != nil {
				return utils.BadReqError(err.Error())
			}
		}
		routes = append(routes, route)
	}
	previous := *svc
	previousConfigs := make([]utils.RouteConfig, len(svc.Routes))
	for i, route := range svc.Routes {
		previousConfigs[i] = route.Config()
	}
	defer func() {
		if result == nil {
			return
		}
		for _, route := range routes {
			route.Stop()
		}
		*svc = previous
		for i, route := range svc.Routes {
			route.Stop()
			route.Update(previousConfigs[i])
		}
		if svc.Enabled {
			if err := svc.Start(); err != nil {
				result.Message += "; restoring previous routes: " + err.Error()
			}
		}
	}()
	if !config.Enabled {
		for _, route := range svc.Routes {
			route.Stop()
		}
	}
	for _, route := range svc.Routes {
		if _, err := containsRoute(routes, route.Config()); err != nil {
			route.Stop()
		}
	}
	for i, route := range routes {
		if err := route.Update(config.Routes[i]); err != nil {
			return utils.CustomError(500, err.Error())
		}
	}
	svc.Name = config.Name
	svc.Id = slug.Make(config.Name)
	svc.Enabled = config.Enabled
	svc.Routes = routes
	if svc.Enabled {
		if err := svc.Start(); err != nil {
			return utils.CustomError(500, err.Error())
		}
	}
	return nil
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

func (svc *Service) Start() error {
	for _, route := range svc.Routes {
		if err := route.Start(); err != nil {
			for _, started := range svc.Routes {
				started.Stop()
			}
			return fmt.Errorf("start service %q: %w", svc.Name, err)
		}
	}
	svc.Enabled = true
	return nil
}
