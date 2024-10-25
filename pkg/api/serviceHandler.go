package api

import (
	"encoding/json"
	"net/http"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
)

func (api *api) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	status := []router.ServiceStatus{}
	for _, svc := range api.All() {
		status = append(status, svc.Status(false))
	}
	WriteData(w, status)
}

func (api *api) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	WriteData(w, service.Status(true))
}

func (api *api) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var config utils.ServiceConfig
	decoder.Decode(&config)
	svc, err := api.Create(config)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	api.Save()
	writeResponse(w, http.StatusCreated, svc.Status(false))
}

func (api *api) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	decoder := json.NewDecoder(r.Body)
	var svc utils.ServiceConfig
	decoder.Decode(&svc)

	service, err := api.Update(id, svc)
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	api.Save()
	WriteData(w, service.Status(true))
}

func (api *api) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := api.Remove(chi.URLParam(r, "id")); err != nil {
		WriteErrorResponse(w, err)
		return
	}
	api.Save()
	WriteStatus(w, http.StatusOK)
}

func (api *api) handleStartRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	service.Start()
	api.Save()
	WriteData(w, service.Status(true))
}

func (api *api) handleStopRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		WriteErrorResponse(w, err)
		return
	}
	service.Stop()
	api.Save()
	WriteData(w, service.Status(true))
}
