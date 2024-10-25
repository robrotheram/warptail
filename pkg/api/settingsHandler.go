package api

import (
	"encoding/json"
	"net/http"
	"warptail/pkg/utils"
)

func (api *api) handleTailscaleSettings(w http.ResponseWriter, r *http.Request) {
	WriteData(w, api.Router.GetTailScaleConfig())
}

func (api *api) handleUpdateTailscaleSettings(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var tsc utils.TailscaleConfig
	decoder.Decode(&tsc)
	api.SaveTailScale(tsc)
	WriteStatus(w, http.StatusOK)
}
