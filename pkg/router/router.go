package router

import (
	"net/http"
	"sync"
	"warptail/pkg/utils"

	"github.com/gosimple/slug"
)

var ServiceNotFoundError = utils.NotFoundError("service not found")

type Router struct {
	Services    map[string]*Service
	backend     Backend
	networkMu   sync.Mutex
	Controllers []Controller
	mu          sync.RWMutex
	ready       bool
}

type RouteInfo struct {
	utils.RouteConfig
	Status RouterStatus
	Stats  utils.TimeSeriesData
}

func NewRouter() *Router {
	return NewRouterWithBackend(&tailscaleBackend{})
}

// NewRouterWithBackend accepts an externally managed network backend.
// Its lifecycle belongs to the caller. NewRouter uses embedded Tailscale.
func NewRouterWithBackend(backend Backend) *Router {
	router := &Router{
		Services:    make(map[string]*Service),
		backend:     backend,
		Controllers: []Controller{},
		ready:       false,
	}
	return router
}

// IsReady returns true if the router has completed initialization
func (r *Router) IsReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ready
}

// SetReady marks the router as ready or not ready
func (r *Router) SetReady(ready bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ready = ready
}

func (r *Router) Init(config utils.Config) error {
	err := r.initBackend(config.Tailscale)
	if err != nil {
		return err
	}
	for _, service := range config.Services {
		if _, err := r.Create(service); err != nil {
			return err
		}
	}
	r.SetReady(true)
	utils.Logger.Info("Router initialization complete")
	return nil
}

func (r *Router) Reload(config utils.Config) error {
	if err := r.initBackend(config.Tailscale); err != nil {
		return err
	}
	for _, svc := range config.Services {
		if r.DoesExists(svc.Name) {
			id := slug.Make(svc.Name)
			if _, err := r.Update(id, svc); err != nil {
				return err
			}
		} else {
			if _, err := r.Create(svc); err != nil {
				return err
			}
		}
	}

	// Collect keys to delete while holding lock, then delete
	r.mu.Lock()
	keysToDelete := []string{}
	for key, svc := range r.Services {
		if !utils.ContainsService(svc.Name, config.Services) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	for _, key := range keysToDelete {
		if svc, ok := r.Services[key]; ok {
			for _, route := range svc.Routes {
				route.Stop()
			}
			delete(r.Services, key)
		}
	}
	r.ready = true
	r.mu.Unlock()
	return nil
}

func (r *Router) DoesExists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id := slug.Make(name)
	_, ok := r.Services[id]
	return ok
}

func (r *Router) Create(svc utils.ServiceConfig) (*Service, *utils.RouterError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := slug.Make(svc.Name)
	if _, ok := r.Services[id]; ok {
		return nil, utils.CustomError(http.StatusConflict, "service already exists unable to load config")
	}
	service, err := NewService(svc, r.backend)
	if err != nil {
		return nil, utils.BadReqError(err.Error())
	}

	if service.Enabled {
		if err := service.Start(); err != nil {
			return nil, utils.CustomError(http.StatusInternalServerError, err.Error())
		}
	}

	r.Services[service.Id] = service
	return service, nil
}

func (r *Router) All() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svcs := []Service{}
	for _, svc := range r.Services {
		svcs = append(svcs, *svc)
	}
	return svcs
}

func (r *Router) Get(id string) (*Service, *utils.RouterError) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if svc, ok := r.Services[id]; ok {
		return svc, nil
	}
	return nil, ServiceNotFoundError
}

func (r *Router) GetHttpRoute(domain string) (*HTTPRoute, *utils.RouterError) {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.Services[id]
	if !ok {
		return nil, ServiceNotFoundError
	}
	if err := existing.Update(svc, r.backend); err != nil {
		return nil, err
	}
	if id != existing.Id {
		r.Services[existing.Id] = existing
		delete(r.Services, id)
	}
	return existing, nil
}

func (r *Router) Remove(id string) *utils.RouterError {
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, svc := range r.Services {
		if svc.Enabled {
			if err := svc.Start(); err != nil {
				utils.Logger.Error(err, "Unable to start service", "service", svc.Name)
			}
		}
	}
}

func (r *Router) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, svc := range r.Services {
		for _, route := range svc.Routes {
			route.Stop()
		}
	}
}
