package api

import (
	"embed"
	"io/fs"
	"net/http"
	"warptail/pkg/utils"
)

type UIConfig struct {
	AuthenticationType string `json:"auth_type"`
	AuthenticationName string `json:"auth_name"`
	ReadOnly           bool   `json:"read_only"`
}
type SPAHandler struct {
	StaticFs   embed.FS
	StaticPath string
	IndexPath  string
	Config     UIConfig
}

func NewUI(config utils.Config, ui embed.FS) SPAHandler {
	spa := SPAHandler{
		StaticFs:   ui,
		StaticPath: "./dashboard/dist",
		IndexPath:  "index.html",
		Config: UIConfig{
			AuthenticationType: config.Application.Authentication.Provider.Type,
			AuthenticationName: config.Application.Authentication.Provider.Name,
			ReadOnly:           !utils.IsEmptyStruct(config.Kubernetes),
		},
	}
	return spa
}

func (h SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assets, _ := fs.Sub(h.StaticFs, "dashboard/dist")
	fs := http.FileServer(http.FS(assets))

	// path, err := filepath.Abs(r.URL.Path)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// // path = filepath.Join(h.StaticPath, path)
	fs.ServeHTTP(w, r)

	// _, err = os.Stat(path)
	// if os.IsNotExist(err) {
	// 	http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
	// 	return
	// } else if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// http.FileServer(http.Dir(h.StaticFs)).ServeHTTP(w, r)
}

func (h SPAHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	utils.WriteData(w, h.Config)
}
