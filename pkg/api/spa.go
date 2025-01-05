package api

import (
	"net/http"
	"os"
	"path/filepath"
	"warptail/pkg/utils"
)

type UIConfig struct {
	AuthenticationType string `json:"auth_type"`
	AuthenticationName string `json:"auth_name"`
	ReadOnly           bool   `json:"read_only"`
}
type SPAHandler struct {
	StaticPath string
	IndexPath  string
	Config     UIConfig
}

func NewUI(config utils.Config) SPAHandler {
	spa := SPAHandler{
		StaticPath: "./dashboard/dist",
		IndexPath:  "index.html",
		Config: UIConfig{
			AuthenticationType: config.Application.Authentication.Provider.Type,
			AuthenticationName: config.Application.Authentication.Provider.Name,
			ReadOnly:           utils.IsEmptyStruct(config.Kubernetes),
		},
	}
	return spa
}

func (h SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path = filepath.Join(h.StaticPath, path)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.Dir(h.StaticPath)).ServeHTTP(w, r)
}

func (h SPAHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	utils.WriteData(w, h.Config)
}
