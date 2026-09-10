package router

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
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

// Routes retain this adapter when settings replace the underlying tsnet server.
type tailscaleBackend struct {
	mu sync.RWMutex
	ts *tsnet.Server
}

func (b *tailscaleBackend) server() (*tsnet.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ts == nil {
		return nil, fmt.Errorf("Tailscale is not initialized")
	}
	return b.ts, nil
}
func (b *tailscaleBackend) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	ts, err := b.server()
	if err != nil {
		return nil, err
	}
	return ts.Dial(ctx, network, address)
}
func (b *tailscaleBackend) Ping(ctx context.Context, address string) (time.Duration, error) {
	ts, err := b.server()
	if err != nil {
		return -1, err
	}
	client, err := ts.LocalClient()
	if err != nil {
		return -1, err
	}
	ip, err := netip.ParseAddr(address)
	if err != nil {
		status, statusErr := client.Status(ctx)
		if statusErr != nil {
			return -1, statusErr
		}
		for _, peer := range status.Peer {
			if peer != nil && len(peer.TailscaleIPs) > 0 && (strings.EqualFold(peer.HostName, address) || strings.EqualFold(strings.TrimSuffix(peer.DNSName, "."), strings.TrimSuffix(address, "."))) {
				ip = peer.TailscaleIPs[0]
				break
			}
		}
		if !ip.IsValid() {
			return -1, fmt.Errorf("no Tailscale peer for %q", address)
		}
	}
	result, err := client.Ping(ctx, ip, tailcfg.PingTSMP)
	if err != nil {
		return -1, err
	}
	if result.Err != "" {
		return -1, fmt.Errorf("Tailscale ping: %s", result.Err)
	}
	return time.Duration(result.LatencySeconds * float64(time.Second)), nil
}
func LogPrintf(format string, args ...any) { utils.Logger.Info(fmt.Sprintf(format, args...)) }

func (r *Router) initBackend(config utils.TailscaleConfig) error {
	if _, ok := r.backend.(*tailscaleBackend); ok {
		return r.UpdateTailscale(config)
	}
	return nil
}
func (r *Router) UpdateTailscale(config utils.TailscaleConfig) error {
	r.networkMu.Lock()
	defer r.networkMu.Unlock()
	backend, ok := r.backend.(*tailscaleBackend)
	if !ok {
		return fmt.Errorf("Tailscale settings are unavailable for this backend")
	}
	old, _ := backend.server()
	changed := old == nil || old.AuthKey != config.AuthKey || old.Hostname != config.Hostname
	if changed {
		r.SetReady(false)
		r.mu.Lock()
		for _, svc := range r.Services {
			for _, route := range svc.Routes {
				route.Stop()
			}
		}
		if old != nil {
			old.Close()
		}
		backend.mu.Lock()
		backend.ts = &tsnet.Server{AuthKey: config.AuthKey, Hostname: config.Hostname, UserLogf: LogPrintf}
		backend.mu.Unlock()
		r.mu.Unlock()
	}
	ts, _ := backend.server()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// Up already waits for authentication and usable network addresses.
	if _, err := ts.Up(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, svc := range r.Services {
		if svc.Enabled {
			if err := svc.Start(); err != nil {
				return err
			}
		}
	}
	return nil
}
func (r *Router) tailscaleStatus(ctx context.Context) (*tsnet.Server, *ipnstate.Status, error) {
	backend, ok := r.backend.(*tailscaleBackend)
	if !ok {
		return nil, nil, fmt.Errorf("Tailscale is not the active backend")
	}
	ts, err := backend.server()
	if err != nil {
		return nil, nil, err
	}
	client, err := ts.LocalClient()
	if err != nil {
		return nil, nil, err
	}
	status, err := client.Status(ctx)
	return ts, status, err
}
func (r *Router) GetTailScaleStatus() TailscaleStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts, status, err := r.tailscaleStatus(ctx)
	if err != nil {
		return TailscaleStatus{Peers: []TailscalePeers{}}
	}
	nodes := []TailscalePeers{}
	for _, peer := range status.Peer {
		if peer == nil || len(peer.TailscaleIPs) == 0 {
			continue
		}
		nodes = append(nodes, TailscalePeers{Id: string(peer.ID), Name: peer.DNSName,
			HostName: peer.HostName, IP: peer.TailscaleIPs[0].String(), LastSeen: peer.LastSeen,
			Os: peer.OS, Online: peer.Online})
	}
	result := TailscaleStatus{HostName: ts.Hostname, Version: status.Version,
		State: status.BackendState, Peers: nodes, AuthURL: status.AuthURL}
	if status.Self != nil {
		result.KeyExpiry = status.Self.KeyExpiry
	}
	return result
}
func (r *Router) GetPeers() ([]TailscalePeers, *utils.RouterError) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, status, err := r.tailscaleStatus(ctx)
	if err != nil {
		return nil, utils.CustomError(http.StatusServiceUnavailable, err.Error())
	}
	nodes := []TailscalePeers{}
	seen := make(map[string]bool)
	for _, peer := range status.Peer {
		if peer == nil || len(peer.TailscaleIPs) == 0 {
			continue
		}
		ip := peer.TailscaleIPs[0].String()
		key := peer.HostName + ":" + ip
		if seen[key] {
			continue
		}
		seen[key] = true
		nodes = append(nodes, TailscalePeers{HostName: peer.HostName, IP: ip})
	}
	return nodes, nil
}
func (r *Router) SaveTailScale(config utils.TailscaleConfig) error {
	if err := r.UpdateTailscale(config); err != nil {
		return err
	}
	r.SetReady(true)
	r.Save()
	return nil
}
func (r *Router) GetTailScaleConfig() utils.TailscaleConfig {
	backend, ok := r.backend.(*tailscaleBackend)
	if !ok {
		return utils.TailscaleConfig{}
	}
	ts, err := backend.server()
	if err != nil {
		return utils.TailscaleConfig{}
	}
	return utils.TailscaleConfig{AuthKey: ts.AuthKey, Hostname: ts.Hostname}
}
