package main

import (
	"crypto/tls"
	"embed"
	"log"
	"net/http"
	"os"
	"warptail/pkg/api"
	"warptail/pkg/controller"
	"warptail/pkg/router"
	"warptail/pkg/utils"
)

//go:embed all:dashboard/dist
var ui embed.FS

var (
	config     utils.Config
	configPath = os.Getenv("CONFIG_PATH")
)

func init() {
	if len(configPath) == 0 {
		configPath = "config.yaml"
	}
	var err error
	config, err = utils.LoadConfig(configPath)
	if err != nil {
		utils.Logger.Error(err, "invalid config")
		os.Exit(1)
	}
}

func main() {
	router := router.NewRouter()
	err := router.Init(config)
	if err != nil {
		utils.Logger.Error(err, "unable to create router")
		return
	}
	router.GetTailScaleStatus()

	if utils.IsEmptyStruct(config.Kubernetes) {
		utils.Logger.Info("Starting Server")
		StartRouter(config, router)
	} else {
		utils.Logger.Info("Kubernetes Configured Starting Server")
		StartK8Router(config, router)
	}
}

func StartK8Router(cfg utils.Config, rt *router.Router) {
	defer rt.StopAll()
	if ctrl, err := controller.NewK8Controller(cfg.Kubernetes); err == nil {
		rt.Controllers = append(rt.Controllers, ctrl)
	}
	go controller.StartController(rt)
	mux := api.NewApi(rt, cfg, ui)

	addr := cfg.Application.GetHTTPAddr()
	utils.Logger.Info("Starting API on http://localhost" + addr)
	err := http.ListenAndServe(addr, mux)
	utils.Logger.Error(err, "unable to start api")
}

func StartRouter(cfg utils.Config, rt *router.Router) {
	defer rt.StopAll()
	mux := api.NewApi(rt, cfg, ui)
	if ctrl, err := controller.NewConfigController(configPath, rt); err == nil {
		rt.Controllers = append(rt.Controllers, ctrl)
	}

	if cfg.Application.UseHTTPS() {
		utils.Logger.Info("Certificates Managed by ACME")
		manager := cfg.Application.ACMEManager()
		rt.Controllers = append(rt.Controllers, controller.NewACMEContoller(manager, cfg.Application))
		go func() {
			err := http.ListenAndServe(":80", manager.HTTPHandler(mux))
			log.Fatal(err)
		}()

		srv := &http.Server{
			Addr:    cfg.Application.GetSSLAddr(),
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate:           manager.GetCertificate,
				PreferServerCipherSuites: true,
				CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
			},
		}
		err := srv.ListenAndServeTLS("", "") // Key and cert provided automatically by autocert.
		log.Fatal(err)
	} else {
		addr := cfg.Application.GetHTTPAddr()
		utils.Logger.Info("Starting API on http://localhost" + addr)
		err := http.ListenAndServe(addr, mux)
		log.Fatal(err, "unable to start api")
	}
}
