package main

import (
	"os"
	"warptail/pkg/api"
	"warptail/pkg/controller"
	"warptail/pkg/router"
	"warptail/pkg/utils"
)

var (
	config     utils.Config
	configPath = os.Getenv("CONFIG_PATH")
)

func init() {
	if len(configPath) == 0 {
		configPath = "config.yaml"
	}
	config = utils.LoadConfig(configPath)
}

func setupControllers(cfg utils.Config, rt *router.Router) {
	rt.Controllers = []router.Controller{}
	if !utils.IsEmptyStruct(cfg.Kubernetes) {
		if ctrl, err := controller.NewK8Controller(cfg.Kubernetes); err == nil {
			rt.Controllers = append(rt.Controllers, ctrl)
		}
	} else {
		if ctrl, err := controller.NewConfigController(configPath, rt); err == nil {
			rt.Controllers = append(rt.Controllers, ctrl)
		}
	}
}

func main() {
	Router := router.NewRouter()
	Router.Init(config)
	setupControllers(config, Router)
	Router.StartAll()
	defer Router.StopAll()
	if !utils.IsEmptyStruct(config.Kubernetes) {
		go controller.StartController(Router)
	}
	server := api.NewApi(Router, config)
	server.Start(config.Dasboard.Port)
}
