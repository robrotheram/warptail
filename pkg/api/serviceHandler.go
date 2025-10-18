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

	// Check if we have no services and if Tailscale needs authentication
	if len(status) == 0 {
		if err := api.checkTailscaleAuth(); err != nil {
			utils.WriteErrorResponse(w, utils.BadReqError("NeedsLogin: "+err.Error()))
			return
		}
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

func (api *api) handleGetTailscaleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := api.Router.GetPeers()
	if err != nil {
		// Check if error is related to Tailscale authentication
		if isAuthError(err) {
			utils.WriteErrorResponse(w, utils.BadReqError("NeedsLogin: "+err.Error()))
			return
		}
		utils.WriteErrorResponse(w, err)
		return
	}
	utils.WriteData(w, nodes)
}

// Helper function to check if Tailscale needs authentication
func (api *api) checkTailscaleAuth() error {
	_, err := api.Router.GetPeers()
	return err
}

// Helper function to determine if an error is authentication-related
func isAuthError(err error) bool {
	errMsg := err.Error()
	return contains(errMsg, "not logged in") ||
		contains(errMsg, "needs login") ||
		contains(errMsg, "authentication required") ||
		contains(errMsg, "offline") ||
		contains(errMsg, "not authenticated")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOfSubstring(s, substr) != -1)))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
