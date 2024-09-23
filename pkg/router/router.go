package router

import (
	"fmt"
	"sync"
	"wormtail/pkg/utils"

	"github.com/google/uuid"
	"tailscale.com/tsnet"
)

type RouterStatus string

const (
	STARTING = RouterStatus("Starting")
	RUNNING  = RouterStatus("Running")
	STOPPED  = RouterStatus("Stopped")
)

type Route interface {
	Start() error
	Stop() error
	Update(utils.RouteConfig) error
	Config() utils.RouteConfig
	Status() RouterStatus
	Stats() utils.TimeSeriesData
}

type Router struct {
	routes map[string]Route
	ts     *tsnet.Server
	wg     sync.WaitGroup
}

type RouteInfo struct {
	utils.RouteConfig
	Status RouterStatus
	Stats  utils.TimeSeriesData
}

func NewRouter(config utils.Config) (*Router, error) {
	router := &Router{
		routes: make(map[string]Route),
		wg:     sync.WaitGroup{},
	}
	s := new(tsnet.Server)
	s.Hostname = config.Tailscale.Hostname
	router.ts = s
	for _, route := range config.Routes {
		router.AddRoute(route)
	}
	return router, nil
}

func (r *Router) Close() {
	r.ts.Close()
}

func (r *Router) AddRoute(config utils.RouteConfig) (Route, error) {
	defer r.save()
	if len(config.Id) == 0 {
		config.Id = uuid.NewString()
	}
	client, _ := r.ts.LocalClient()
	switch config.Type {
	case utils.TCP:
		r.routes[config.Id] = NewTCPRoute(config, client)
	case utils.HTTP:
		r.routes[config.Id] = NewHTTPRoute(config, r.ts)
	default:
		return nil, fmt.Errorf("no handler for type %s", config.Type)
	}
	return r.routes[config.Id], nil
}

func (r *Router) UpdateRoute(route utils.RouteConfig) {
	defer r.save()
	r.DeleteRoute(route.Id)
	r.AddRoute(route)
	r.StopRoute(route.Id)
}

func (r *Router) DeleteRoute(Id string) {
	defer r.save()
	r.StopRoute(Id)
	delete(r.routes, Id)
}

func (r *Router) GetRoute(Id string) Route {
	if route, ok := r.routes[Id]; ok {
		return route
	}
	return nil
}

func (r *Router) GetRouteByName(name string) (Route, error) {
	for _, route := range r.routes {
		if route.Config().Name == name {
			return route, nil
		}
	}
	return &TCPRoute{}, fmt.Errorf("no route found")
}

func (r *Router) save() {
	routes := []utils.RouteConfig{}
	for _, route := range r.routes {
		routes = append(routes, route.Config())
	}
	utils.SaveRoutes(routes)
}

func (r *Router) Get(name string) (RouteInfo, error) {
	if route, ok := r.routes[name]; ok {
		return RouteInfo{
			RouteConfig: route.Config(),
			Status:      route.Status(),
			Stats:       route.Stats(),
		}, nil
	}
	return RouteInfo{}, fmt.Errorf("route %s not found", name)
}

func (r *Router) GetAll() []RouteInfo {
	routes := []RouteInfo{}
	for key := range r.routes {
		r, _ := r.Get(key)
		routes = append(routes, r)
	}
	return routes
}

func (r *Router) StartRoute(name string) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.routes[name].Start()
	}()
}

func (r *Router) StopRoute(Id string) {
	r.routes[Id].Stop()
}

func (r *Router) StartAll() {
	for name := range r.routes {
		r.StartRoute(name)
	}
}