package router

import (
	"context"
	"fmt"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

type TailscaleStatus struct {
	Messages []string `json:"messages"`
	Version  string   `json:"version"`
	State    string   `json:"state"`
}

func LogPrintf(format string, args ...interface{}) {
	logger := utils.Logger
	formattedMessage := fmt.Sprintf(format, args...)
	logger.Info(formattedMessage)
}

func (r *Router) UpdateTailscale(config utils.TailscaleConfig) error {

	if r.ts == nil {
		r.ts = &tsnet.Server{
			AuthKey:  config.AuthKey,
			Hostname: config.Hostname,
			UserLogf: LogPrintf,
		}
	}

	if r.ts.AuthKey != config.AuthKey || r.ts.Hostname != config.Hostname {
		r.StopAll()
		r.ts.Close()
		r.ts.AuthKey = config.AuthKey
		r.ts.Hostname = config.Hostname

		r.ts = &tsnet.Server{
			AuthKey:  config.AuthKey,
			Hostname: config.Hostname,
			UserLogf: LogPrintf,
		}
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	_, err := r.ts.Up(ctx)

	return err
}

func (r *Router) GetTailScaleStatus() TailscaleStatus {
	c, _ := r.ts.LocalClient()
	status, err := c.Status(context.Background())
	if err != nil {
		return TailscaleStatus{}
	}
	return TailscaleStatus{
		Version:  status.Version,
		State:    status.BackendState,
		Messages: status.Health,
	}
}

func (r *Router) SaveTailScale(config utils.TailscaleConfig) {
	r.UpdateTailscale(config)
	r.Save()
	r.StartAll()
}

func (r *Router) GetTailScaleConfig() utils.TailscaleConfig {
	return utils.TailscaleConfig{
		AuthKey:  r.ts.AuthKey,
		Hostname: r.ts.Hostname,
	}
}
