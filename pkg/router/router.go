package router

import (
	"context"
	"net/http"
	"warptail/pkg/utils"

	"github.com/gosimple/slug"
	"tailscale.com/tsnet"
)

var ServiceNotFoundError = utils.NotFoundError("service not found")

type Router struct {
	Services    map[string]*Service
	ts          *tsnet.Server
	Controllers []Controller
}

type RouteInfo struct {
	utils.RouteConfig
	Status RouterStatus
	Stats  utils.TimeSeriesData
}

func NewRouter() *Router {
	router := &Router{
		Services:    make(map[string]*Service),
		Controllers: []Controller{},
	}
	return router
}

func (r *Router) Init(config utils.Config) error {
	err := r.UpdateTailscale(config.Tailscale)
	if err != nil {
		return err
	}
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

func (r *Router) Create(svc utils.ServiceConfig) (*Service, *utils.RouterError) {
	if r.DoesExists(svc.Name) {
		return nil, utils.CustomError(http.StatusConflict, "service already exists unable to load config")
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

func (r *Router) Get(id string) (*Service, *utils.RouterError) {
	if svc, ok := r.Services[id]; ok {
		return svc, nil
	}
	return nil, ServiceNotFoundError
}

func (r *Router) GetHttpRoute(domain string) (*HTTPRoute, *utils.RouterError) {
	for _, svc := range r.Services {
		for _, route := range svc.Routes {
			if route.Config().Type == utils.HTTP || route.Config().Type == utils.HTTPS {
				if route.Config().Domain == domain {
					return route.(*HTTPRoute), nil
				}
			}
		}
	}
	return nil, utils.NotFoundError("route not found")
}

func (r *Router) Update(id string, svc utils.ServiceConfig) (*Service, *utils.RouterError) {
	existing, ok := r.Services[id]
	if !ok {
		return nil, ServiceNotFoundError
	}
	existing.Update(svc, r.ts)
	if id != existing.Id {
		r.Services[existing.Id] = existing
		delete(r.Services, id)
	}
	return existing, nil
}

func (r *Router) Remove(id string) *utils.RouterError {
	svc, ok := r.Services[id]
	if !ok {
		return ServiceNotFoundError
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

func (r *Router) GetPeers() ([]TailscalePeers, *utils.RouterError) {
	c, _ := r.ts.LocalClient()
	status, err := c.Status(context.Background())
	if err != nil {
		return []TailscalePeers{}, utils.CustomError(http.StatusInternalServerError, "unable to get tailscale status")
	}
	nodes := []TailscalePeers{}
	seen := make(map[string]bool)
	for _, peer := range status.Peer {
		if len(peer.TailscaleIPs) == 0 {
			continue
		}
		ip := peer.TailscaleIPs[0].String()
		key := peer.HostName + ":" + ip
		if seen[key] {
			continue
		}
		seen[key] = true
		nodes = append(nodes, TailscalePeers{
			HostName: peer.HostName,
			IP:       ip,
		})
	}
	return nodes, nil
}
