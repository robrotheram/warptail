package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type apiCtx string

const SvcContext = apiCtx("service")

type api struct {
	*router.Router
	Mux    *chi.Mux
	config utils.DashboardConfig
}

func WriteErrorResponse(w http.ResponseWriter, err *router.RouterError) {
	writeResponse(w, err.StatusCode, err)
}

func WriteData(w http.ResponseWriter, data any) {
	writeResponse(w, http.StatusOK, data)
}

func WriteStatus(w http.ResponseWriter, statusCode int) {
	writeResponse(w, statusCode, nil)
}

func writeResponse(w http.ResponseWriter, statusCode int, data any) {
	// Set content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Set custom status code
	w.WriteHeader(statusCode)

	// Marshal the response data into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		http.Error(w, "Error processing the response", http.StatusInternalServerError)
		return
	}

	// Write JSON to the ResponseWriter
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Error writing response: %v", err)
		http.Error(w, "Error writing the response", http.StatusInternalServerError)
	}
}

func NewApi(router *router.Router, config utils.DashboardConfig) *api {
	api := api{
		Router: router,
		Mux:    chi.NewRouter(),
		config: config,
	}
	// Add middlewares
	api.Mux.Use(middleware.RequestID)
	api.Mux.Use(middleware.RealIP)
	api.Mux.Use(middleware.Logger)
	api.Mux.Use(middleware.Recoverer)
	api.Mux.Use(middleware.Compress(5))

	api.Mux.Use(api.proxy)

	api.Mux.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool { return true },
		AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))

	api.Mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	if config.Enabled {
		spa := SPAHandler{
			StaticPath: "./dashboard/dist",
			IndexPath:  "index.html",
		}

		api.Mux.Get("/*", spa.ServeHTTP)
		api.Mux.Post("/auth/login", api.loginHandler)
		api.Mux.Group(func(r chi.Router) {
			r.Use(TokenAuthMiddleware)
			r.Route("/api/settings", func(r chi.Router) {
				r.Get("/tailscale", api.handleTailscaleSettings)
				r.Post("/tailscale", api.handleUpdateTailscaleSettings)
			})
			r.Route("/api/services", func(r chi.Router) {
				r.Get("/", api.handleGetRoutes)
				r.Post("/", api.handleCreateRoute)
			})
			r.Route("/api/services/{id}", func(r chi.Router) {
				r.Get("/", api.handleGetRoute)
				r.Put("/", api.handleUpdateRoute)
				r.Delete("/", api.handleDeleteRoute)
				r.Post("/stop", api.handleStopRoute)
				r.Post("/start", api.handleStartRoute)
			})
		})
	}
	return &api
}

func (api *api) proxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}
		if route, err := api.GetHttpRoute(host); err == nil {
			route.Handle(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *api) Start(addr string) {
	log.Println("Starting API on http://localhost" + addr)
	log.Println(http.ListenAndServe(addr, api.Mux))
}
