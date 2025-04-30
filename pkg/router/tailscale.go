package router

import (
	"context"
	"fmt"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

type TailscaleStatus struct {
	Version   string           `json:"version"`
	State     string           `json:"state"`
	Peers     []TailscalePeers `json:"nodes"`
	HostName  string           `json:"hostname"`
	KeyExpiry *time.Time       `json:"key_expiry"`
}

type TailscalePeers struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	HostName  string    `json:"hostname"`
	IP        string    `json:"ip"`
	LastSeen  time.Time `json:"last_seen"`
	Online    bool      `json:"online"`
	Os        string    `json:"os"`
	KeyExpiry time.Time `json:"key_expiry"`
}

func LogPrintf(format string, args ...any) {
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

	nodes := []TailscalePeers{}
	for _, peer := range status.Peer {
		nodes = append(nodes, TailscalePeers{
			Id:       string(peer.ID),
			Name:     peer.DNSName,
			HostName: peer.HostName,
			IP:       peer.TailscaleIPs[0].String(),
			LastSeen: peer.LastSeen,
			Os:       peer.OS,
			Online:   peer.Online,
		})
	}

	return TailscaleStatus{
		HostName:  r.ts.Hostname,
		KeyExpiry: status.Self.KeyExpiry,
		Version:   status.Version,
		State:     status.BackendState,
		Peers:     nodes,
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
