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
	utils.WriteData(w, status)
}

func (api *api) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	utils.WriteData(w, service.Status(true))
}

func (api *api) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var config utils.ServiceConfig
	decoder.Decode(&config)
	svc, err := api.Create(config)
	if err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	api.Save()
	utils.WriteResponse(w, http.StatusCreated, svc.Status(false))
}

func (api *api) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	decoder := json.NewDecoder(r.Body)
	var svc utils.ServiceConfig
	decoder.Decode(&svc)

	service, err := api.Update(id, svc)
	if err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	api.Save()
	utils.WriteData(w, service.Status(true))
}

func (api *api) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := api.Remove(chi.URLParam(r, "id")); err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	api.Save()
	utils.WriteStatus(w, http.StatusOK)
}

func (api *api) handleStartRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	service.Start()
	api.Save()
	utils.WriteData(w, service.Status(true))
}

func (api *api) handleStopRoute(w http.ResponseWriter, r *http.Request) {
	service, err := api.Router.Get(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteErrorResponse(w, err)
		return
	}
	service.Stop()
	api.Save()
	utils.WriteData(w, service.Status(true))
}
