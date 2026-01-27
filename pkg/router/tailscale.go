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
	AuthURL   string           `json:"auth_url,omitempty"`
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()
	_, err := r.ts.Up(ctx)
	if err != nil {
		return err
	}

	// Wait for Tailscale to be fully authenticated and running
	return r.WaitForTailscale(ctx)
}

// WaitForTailscale blocks until Tailscale backend state is "Running" or context times out
func (r *Router) WaitForTailscale(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for tailscale to authenticate: %w", ctx.Err())
		case <-ticker.C:
			c, err := r.ts.LocalClient()
			if err != nil {
				continue
			}
			status, err := c.Status(ctx)
			if err != nil {
				continue
			}
			if status.BackendState == "Running" {
				utils.Logger.Info("Tailscale connected successfully", "hostname", r.ts.Hostname)
				return nil
			}
			utils.Logger.V(1).Info("Waiting for Tailscale authentication", "state", status.BackendState)
		}
	}
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
		AuthURL:   status.AuthURL,
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

func GetTailScaleServerIp(ts *tsnet.Server) (string, error) {
	client, err := ts.LocalClient()
	if err != nil {
		return "", err
	}
	status, err := client.Status(context.Background())
	if err != nil {
		return "", err
	}
	ip := status.Self.TailscaleIPs[0].String()
	return ip, nil
}
