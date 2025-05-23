package api

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"
	"warptail/pkg/auth"
	botprotect "warptail/pkg/botProtect"
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
	botProtect     *botprotect.BotChallenge
}

func NewApi(router *router.Router, config utils.Config, ui embed.FS) *chi.Mux {
	db := utils.NewDB(config)
	mux := chi.NewMux()
	api := api{
		Router: router,
	}
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Compress(5))

	mux.Use(api.proxy)
	mux.Use(utils.RequestLogger.Middleware)

	mux.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool { return true },
		AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))

	api.authentication = auth.NewAuthentication(mux, db, config.Application.Authentication)
	api.botProtect = botprotect.NewBotChallenge(mux, config.Application.Authentication)

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
			r.Get(("/logs"), api.handleGetLogs)
		})
		r.Get("/api/tailsale/nodes", api.handleGetTailscaleNodes)
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
		route, err := api.GetHttpRoute(host)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if route.Config().Private {
			authenticated := false
			api.authentication.Authenticate(w, r, func(w http.ResponseWriter, r *http.Request) {
				authenticated = true
			})
			if !authenticated {
				return
			}
		}
		if route.Config().BotProtect && route.Config().Type == utils.HTTPS {
			// Only set isProxy context for the actual backend call, not for the challenge page
			api.botProtect.Middleware(w, r, func(w http.ResponseWriter, r *http.Request) {
				r = r.WithContext(context.WithValue(r.Context(), "isProxy", true))
				route.Handle(w, r)
			})
		} else {
			r = r.WithContext(context.WithValue(r.Context(), "isProxy", true))
			route.Handle(w, r)
		}
	})
}
