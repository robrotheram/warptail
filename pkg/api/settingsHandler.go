package api

import (
	"encoding/json"
	"net/http"
	"warptail/pkg/utils"
)

func (api *api) handleTailscaleSettings(w http.ResponseWriter, r *http.Request) {
	utils.WriteData(w, api.Router.GetTailScaleConfig())
}

func (api *api) handleUpdateTailscaleSettings(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var tsc utils.TailscaleConfig
	decoder.Decode(&tsc)
	api.SaveTailScale(tsc)
	utils.WriteStatus(w, http.StatusOK)
}

func (api *api) handleUpdateTailscaleSatus(w http.ResponseWriter, r *http.Request) {
	utils.WriteData(w, api.GetTailScaleStatus())
}

func (api *api) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	var logs []string
	var err error
	switch r.URL.Query().Get("type") {
	case "access":
		logs, err = utils.RequestLogger.GetLogs("access")
	case "error":
		logs, err = utils.RequestLogger.GetLogs("error")
	case "server":
		logs = utils.GetLogs()
	default:
		utils.WriteErrorResponse(w, utils.BadReqError("do not know log type"))
		return
	}
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to get logs"))
		return
	}
	utils.WriteData(w, logs)
}
