// // The tshello server demonstrates how to use Tailscale as a library.
package main

import (
	"log"
	"os"
	"warptail/pkg/api"
	"warptail/pkg/controller"
	"warptail/pkg/router"
	"warptail/pkg/utils"
)

var configPath = os.Getenv("CONFIG_PATH")
var Router *router.Router
var config utils.Config

func buildControllers(cfg utils.Config) []router.Controller {
	ctrls := []router.Controller{}
	if ctrl, err := controller.NewK8Controller(cfg.Kubernetes); err == nil {
		ctrls = append(ctrls, ctrl)
	}
	if ctrl, err := controller.NewConfigController(configPath, Router); err == nil {
		ctrls = append(ctrls, ctrl)
	}
	return ctrls
}

func init() {
	if len(configPath) == 0 {
		configPath = "config.yaml"
	}
	config = utils.LoadConfig(configPath)
	Router = router.NewRouter()
	Router.Controllers = buildControllers(config)
}

func main() {
	err := Router.Init(config)
	if err != nil {
		log.Fatalf("Unable to Start router %w", err)
	}
	Router.StartAll()
	defer Router.StopAll()
	server := api.NewApi(Router, config.Dasboard)
	server.Start(config.Dasboard.Port)
}
