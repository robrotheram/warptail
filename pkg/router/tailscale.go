package router

import (
	"fmt"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

func LogPrintf(format string, args ...interface{}) {
	logger := utils.Logger
	formattedMessage := fmt.Sprintf(format, args...)
	logger.Info(formattedMessage)
}

func (r *Router) UpdateTailscale(config utils.TailscaleConfig) {

	if r.ts == nil {
		r.ts = new(tsnet.Server)
		r.ts.AuthKey = config.AuthKey
		r.ts.Hostname = config.Hostname
		r.ts.UserLogf = LogPrintf
		r.ts.Start()
		return
	}

	if r.ts.AuthKey != config.AuthKey || r.ts.Hostname != config.Hostname {
		r.StopAll()
		r.ts.Close()
		r.ts = new(tsnet.Server)
		r.ts.AuthKey = config.AuthKey
		r.ts.Hostname = config.Hostname
		r.ts.Start()
	}
}

func (r *Router) SaveTailScale(config utils.TailscaleConfig) {
	r.UpdateTailscale(config)
	r.Save()
}

func (r *Router) GetTailScaleConfig() utils.TailscaleConfig {
	return utils.TailscaleConfig{
		AuthKey:  r.ts.AuthKey,
		Hostname: r.ts.Hostname,
	}
}
