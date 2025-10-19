package main

import (
	"context"
	"crypto/tls"
	"embed"
	"log"
	"net/http"
	"os"
	"warptail/pkg/api"
	"warptail/pkg/cmd"
	"warptail/pkg/controller"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/urfave/cli/v3"
)

//go:embed all:dashboard/dist
var ui embed.FS

var (
	version = "dev"
)

func main() {
	cmd := &cli.Command{
		Name:    "warptail",
		Usage:   "Tailscale proxy service",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:   "install",
				Usage:  "install warptail as a systemd service",
				Action: cmd.InstallService,
			},
			{
				Name:   "uninstall",
				Usage:  "complete a task on the list",
				Action: cmd.UninstallService,
			},
			{
				Name:   "update",
				Usage:  "update warptail to the latest release",
				Action: cmd.Update,
			},
		},
		Action: ApplicationCmd,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func ApplicationCmd(ctx context.Context, cmd *cli.Command) error {
	config, err := utils.LoadConfig(utils.ConfigPath)
	if err != nil {
		return err
	}
	router := router.NewRouter()
	go router.Init(config)

	if utils.IsEmptyStruct(config.Kubernetes) {
		utils.Logger.Info("Starting Server")
		return StartRouter(config, router)
	} else {
		utils.Logger.Info("Kubernetes Configured Starting Server")
		return StartK8Router(config, router)
	}

}

func StartK8Router(cfg utils.Config, rt *router.Router) error {
	defer rt.StopAll()
	if ctrl, err := controller.NewK8Controller(cfg.Kubernetes); err == nil {
		rt.Controllers = append(rt.Controllers, ctrl)
	}
	go controller.StartController(rt)
	mux := api.NewApi(rt, cfg, ui)

	addr := cfg.Application.GetHTTPAddr()
	utils.Logger.Info("Starting API on http://localhost" + addr)
	return http.ListenAndServe(addr, mux)
}

func StartRouter(cfg utils.Config, rt *router.Router) error {
	defer rt.StopAll()
	mux := api.NewApi(rt, cfg, ui)
	if ctrl, err := controller.NewConfigController(utils.ConfigPath, rt); err == nil {
		rt.Controllers = append(rt.Controllers, ctrl)
	}

	if cfg.UseHTTPS() {
		utils.Logger.Info("Certificates Managed by ACME")
		manager := cfg.CertificateManager.ACMEManager()
		rt.Controllers = append(rt.Controllers, controller.NewACMEContoller(manager, cfg.CertificateManager))
		go func() {
			err := http.ListenAndServe(":80", manager.HTTPHandler(mux))
			log.Fatal(err)
		}()

		srv := &http.Server{
			Addr:    cfg.CertificateManager.GetSSLAddr(),
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate:           manager.GetCertificate,
				PreferServerCipherSuites: true,
				CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
			},
		}
		return srv.ListenAndServeTLS("", "") // Key and cert provided automatically by autocert.
	} else {
		addr := cfg.Application.GetHTTPAddr()
		utils.Logger.Info("Starting API on http://localhost" + addr)
		return http.ListenAndServe(addr, mux)
	}
}
