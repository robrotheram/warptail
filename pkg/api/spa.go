package api

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"warptail/pkg/utils"
)

type UIConfig struct {
	AuthenticationType string `json:"auth_type"`
	AuthenticationName string `json:"auth_name"`
	ReadOnly           bool   `json:"read_only"`
	SiteName           string `json:"site_name,omitempty"`
	SiteLogo           string `json:"site_logo,omitempty"`
}
type SPAHandler struct {
	StaticFs   embed.FS
	StaticPath string
	IndexPath  string
	Config     UIConfig
}

func NewUI(config utils.Config, ui embed.FS) SPAHandler {
	cfg := UIConfig{
		ReadOnly: !utils.IsEmptyStruct(config.Kubernetes),
		SiteName: config.Application.SiteName,
		SiteLogo: config.Application.SiteLogo,
	}

	if config.Application.Authentication.Provider.OIDC != nil {
		cfg.AuthenticationType = "openid"
		cfg.AuthenticationName = config.Application.Authentication.Provider.OIDC.Name
	}

	spa := SPAHandler{
		StaticFs:   ui,
		StaticPath: "dashboard/dist",
		IndexPath:  "index.html",
		Config:     cfg,
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
	_, err = h.StaticFs.Open(path)
	if err == nil {
		assets, _ := fs.Sub(h.StaticFs, h.StaticPath)
		http.FileServer(http.FS(assets)).ServeHTTP(w, r)
		return
	}

	indexFile, err := h.StaticFs.Open(filepath.Join(h.StaticPath, h.IndexPath))
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	defer indexFile.Close()
	data, _ := io.ReadAll(indexFile)
	w.Write(data)
}

// assets.Open(path)
// _, err = assets.Stat(path)
// path, err := filepath.Abs(r.URL.Path)
// if err != nil {
// 	http.Error(w, err.Error(), http.StatusBadRequest)
// 	return
// }
// // path = filepath.Join(h.StaticPath, path)

// _, err = os.Stat(path)
// if os.IsNotExist(err) {
// 	http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
// 	return
// } else if err != nil {
// 	http.Error(w, err.Error(), http.StatusInternalServerError)
// 	return
// }
// http.FileServer(http.Dir(h.StaticFs)).ServeHTTP(w, r)
//}

func (h SPAHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	utils.WriteData(w, h.Config)
}
