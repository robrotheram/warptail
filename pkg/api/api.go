package api

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"warptail/pkg/auth"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type apiCtx string

const SvcContext = apiCtx("service")

type api struct {
	*router.Router
	authentication *auth.Authentication
}

func NewApi(router *router.Router, config utils.Config, ui embed.FS) *chi.Mux {
	db := utils.NewDB(config)
	mux := chi.NewMux()
	api := api{
		Router: router,
	}
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	// mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Compress(5))
	mux.Use(api.proxy)
	mux.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool { return true },
		AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))

	api.authentication = auth.NewAuthentication(mux, db, config.Application.Authentication)

	mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	//Handle Metrics
	mux.Handle("/metrics", promhttp.Handler())
	go api.startMetrics()
	mux.Group(func(r chi.Router) {
		r.Use(api.authentication.DashboardMiddleware)
		r.Get("/api/services", api.handleGetRoutes)
		r.Get("/api/services/{id}", api.handleGetRoute)

	})
	mux.Group(func(r chi.Router) {
		r.Use(api.authentication.DashboardAdminMiddleware)
		r.Route("/api/settings", func(r chi.Router) {
			r.Get("/tailscale", api.handleTailscaleSettings)
			r.Post("/tailscale", api.handleUpdateTailscaleSettings)
			r.Get("/tailscale/status", api.handleUpdateTailscaleSatus)
		})

		r.Post("/api/services", api.handleCreateRoute)

		r.Put("/api/services/{id}", api.handleUpdateRoute)
		r.Delete("/api/services/{id}", api.handleDeleteRoute)
		r.Post("/api/services/{id}/stop", api.handleStopRoute)
		r.Post("/api/services/{id}/start", api.handleStartRoute)

		r.Route("/api/user", func(r chi.Router) {
			r.Get("/", api.authentication.HandleListUsers)
			r.Put("/", api.authentication.HandleCreateUsers)
			r.Post("/{id}", api.authentication.HandleUpdateUser)
			r.Delete("/{id}", api.authentication.HandleDeleteUser)
		})
	})

	spa := NewUI(config, ui)
	mux.Get("/config", spa.HandleConfig)
	mux.Get("/*", spa.ServeHTTP)

	return mux
}

func (api *api) proxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}
		if route, err := api.GetHttpRoute(host); err == nil {
			if route.Config().Private {
				api.authentication.Authenticate(w, r, route.Handle)
			} else {
				route.Handle(w, r)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
