package router

import (
	"net/http"
	"warptail/pkg/utils"

	"github.com/go-logr/logr"
	"github.com/gosimple/slug"
	"tailscale.com/tsnet"
)

type Router struct {
	Services    map[string]*Service
	ts          *tsnet.Server
	Controllers []Controller
	logger      logr.Logger
}

type RouteInfo struct {
	utils.RouteConfig
	Status RouterStatus
	Stats  utils.TimeSeriesData
}

func NewRouter() *Router {
	router := &Router{
		Services: make(map[string]*Service),
		logger:   utils.Logger,
	}
	return router
}

func (r *Router) Init(config utils.Config) error {
	r.UpdateTailscale(config.Tailscale)
	for _, service := range config.Services {
		if _, err := r.Create(service); err != nil {
			return err
		}
	}
	return nil
}

func (r *Router) Reload(config utils.Config) {
	r.UpdateTailscale(config.Tailscale)

	for _, svc := range config.Services {
		if r.DoesExists(svc.Name) {
			id := slug.Make(svc.Name)
			r.Update(id, svc)
		} else {
			r.Create(svc)
		}
	}

	for key, svc := range r.Services {
		if !utils.ContainsService(svc.Name, config.Services) {
			delete(r.Services, key)
		}
	}
}

func (r *Router) DoesExists(name string) bool {
	id := slug.Make(name)
	_, ok := r.Services[id]
	return ok
}

func (r *Router) Create(svc utils.ServiceConfig) (*Service, *RouterError) {
	if r.DoesExists(svc.Name) {
		return nil, CustomError(http.StatusConflict, "service already exists unable to load config")
	}
	service := NewService(svc, r.ts)
	r.Services[service.Id] = service

	if service.Enabled {
		service.Start()
	}

	return service, nil
}

func (r *Router) All() []Service {
	svcs := []Service{}
	for _, svc := range r.Services {
		svcs = append(svcs, *svc)
	}
	return svcs
}

func (r *Router) Get(id string) (*Service, *RouterError) {
	if svc, ok := r.Services[id]; ok {
		return svc, nil
	}
	return nil, NotFoundError("service not found")
}

func (r *Router) GetHttpRoute(domain string) (*HTTPRoute, *RouterError) {
	for _, svc := range r.Services {
		for _, route := range svc.Routes {
			if route.Config().Type == utils.HTTP {
				if route.Config().Domain == domain {
					return route.(*HTTPRoute), nil
				}
			}
		}
	}
	return nil, NotFoundError("route not found")
}

func (r *Router) Update(id string, svc utils.ServiceConfig) (*Service, *RouterError) {
	existing, ok := r.Services[id]
	if !ok {
		return nil, NotFoundError("service not found")
	}
	existing.Update(svc, r.ts)
	if id != existing.Id {
		r.Services[existing.Id] = existing
		delete(r.Services, id)
	}
	return existing, nil
}

func (r *Router) Remove(id string) *RouterError {
	svc, ok := r.Services[id]
	if !ok {
		return NotFoundError("service not found")
	}
	for _, route := range svc.Routes {
		route.Stop()
	}
	delete(r.Services, id)
	return nil
}

func (r *Router) Save() {
	for _, ctrl := range r.Controllers {
		ctrl.Update(r)
	}
}

func (r *Router) StartAll() {
	for _, svc := range r.Services {
		svc.Start()
	}
}

func (r *Router) StopAll() {
	for _, svc := range r.Services {
		svc.Stop()
	}
}

func (r *Router) GetLogger() logr.Logger {
	return r.logger
}
