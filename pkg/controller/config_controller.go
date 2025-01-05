package controller

import (
	"os"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

type ConfigCtrl struct {
	path     string
	watcher  *fsnotify.Watcher
	router   *router.Router
	lastHash [16]byte
}

func NewConfigController(path string, router *router.Router) (*ConfigCtrl, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctrl := ConfigCtrl{
		path:     path,
		watcher:  watcher,
		router:   router,
		lastHash: utils.ConfigHash(path),
	}
	ctrl.Watch()
	return &ctrl, nil
}
func (ctrl *ConfigCtrl) Save(config utils.Config) {
	b, err := yaml.Marshal(config)
	if err != nil {
		utils.Logger.Error(err, "unable to marshal config")
	}
	err = os.WriteFile(ctrl.path, b, os.ModeDir)
	if err != nil {
		utils.Logger.Error(err, "unable to write config")
	}
	utils.Logger.Info("config saved")
}

func (ctrl *ConfigCtrl) Update(router *router.Router) {
	svcs := []utils.ServiceConfig{}
	for _, svc := range router.Services {
		config := utils.ServiceConfig{
			Name:    svc.Name,
			Enabled: svc.Enabled,
			Routes:  []utils.RouteConfig{},
		}
		for _, route := range svc.Routes {
			config.Routes = append(config.Routes, route.Config())
		}
		svcs = append(svcs, config)
	}
	config := utils.LoadConfig(ctrl.path)
	config.Tailscale = router.GetTailScaleConfig()
	ctrl.lastHash = utils.ConfigHash(ctrl.path)
	config.Services = svcs
	ctrl.Save(config)
}

func (ctrl *ConfigCtrl) Watch() error {
	go func() {
		for {
			select {
			case event, ok := <-ctrl.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					currentHash := utils.ConfigHash(ctrl.path)
					if currentHash != ctrl.lastHash {
						utils.Logger.Info("file modified by an external source", "source", event.Name)
						ctrl.lastHash = currentHash
						config := utils.LoadConfig(ctrl.path)
						ctrl.router.Reload(config)
					}
				}
			case err, ok := <-ctrl.watcher.Errors:
				if !ok {
					return
				}
				utils.Logger.Error(err, "unable watch for config changes")
			}
		}
	}()
	return ctrl.watcher.Add(ctrl.path)
}
