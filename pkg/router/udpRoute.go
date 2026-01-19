package router

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"warptail/pkg/utils"

	tailscale "tailscale.com/tsnet"
)

const (
	udpBufferSize        = 65535
	udpSessionTimeout    = 30 * time.Second
	udpHeartbeatInterval = 5 * time.Second
)

// udpSession represents a client session for UDP NAT traversal.
// Each client gets its own session with a dedicated backend connection
// to maintain consistent source ports for protocols like QUIC.
type udpSession struct {
	clientAddr net.Addr
	lastSeen   atomic.Value // stores time.Time
}

// UDPRoute handles UDP traffic proxying through Tailscale.
// It maintains per-client sessions to preserve connection identity
// for stateful UDP protocols like QUIC.
type UDPRoute struct {
	config utils.RouteConfig
	client *tailscale.Server
	data   *utils.TimeSeries

	mu         sync.RWMutex
	status     RouterStatus
	listener   net.PacketConn
	remote     net.PacketConn
	remoteAddr *net.UDPAddr
	tsNodeAddr string

	quit chan struct{}
	wg   sync.WaitGroup

	sessions sync.Map

	latency   time.Duration
	latencyMu sync.RWMutex
}

func NewUDPRoute(config utils.RouteConfig, client *tailscale.Server) *UDPRoute {
	return &UDPRoute{
		config: config,
		data:   utils.NewTimeSeries(time.Second, 1000),
		status: STOPPED,
		client: client,
	}
}

func (route *UDPRoute) Status() RouterStatus {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.status
}

func (route *UDPRoute) Config() utils.RouteConfig {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.config
}

func (route *UDPRoute) Stats() utils.TimeSeriesData {
	return route.data.Data
}

func (route *UDPRoute) Update(config utils.RouteConfig) error {
	route.Stop()
	route.mu.Lock()
	route.config = config
	route.mu.Unlock()
	return route.Start()
}

func (route *UDPRoute) Stop() error {
	route.mu.Lock()
	if route.status != RUNNING {
		route.mu.Unlock()
		return fmt.Errorf("route not running")
	}
	route.status = STOPPING

	// Close connections to unblock readers
	if route.listener != nil {
		route.listener.Close()
	}
	if route.remote != nil {
		route.remote.Close()
	}

	// Signal all goroutines to stop
	close(route.quit)
	route.mu.Unlock()

	// Wait for all goroutines to finish
	route.wg.Wait()

	route.mu.Lock()
	route.status = STOPPED
	route.mu.Unlock()

	utils.Logger.Info("Stopped UDP route", "port", route.config.Port)
	return nil
}

func (route *UDPRoute) Start() error {
	route.mu.Lock()
	if route.status == RUNNING {
		route.mu.Unlock()
		route.Stop()
		route.mu.Lock()
	}

	route.status = STARTING
	route.quit = make(chan struct{})

	laddr := fmt.Sprintf(":%d", route.config.Port)

	var err error

	route.listener, err = net.ListenPacket("udp", laddr)
	if err != nil {
		route.status = STOPPED
		route.mu.Unlock()
		return err
	}

	tsIP, err := GetTailScaleServerIp(route.client)
	if err != nil {
		route.listener.Close()
		route.status = STOPPED
		route.mu.Unlock()
		log.Println("Failed to get Tailscale node address:", err)
		return err
	}
	remoteAddr := fmt.Sprintf("%s:%d", tsIP, route.config.Machine.Port)

	route.remote, err = route.client.ListenPacket("udp", remoteAddr)
	if err != nil {
		route.listener.Close()
		route.status = STOPPED
		route.mu.Unlock()
		utils.Logger.Error(err, "Failed to create Tailscale UDP socket:")
		return err
	}

	route.remoteAddr, err = net.ResolveUDPAddr("udp", route.backendAddr())
	if err != nil {
		route.listener.Close()
		route.remote.Close()
		route.status = STOPPED
		route.mu.Unlock()
		log.Fatal("Failed to resolve game server address:", err)
	}

	route.wg.Add(4)
	go route.reader()
	go route.serve()
	go route.cleanupStaleSessions()
	go route.runHeartbeat()

	route.status = RUNNING
	route.mu.Unlock()
	return nil
}

func (route *UDPRoute) serve() {
	defer route.wg.Done()
	buf := make([]byte, udpBufferSize)
	for {
		select {
		case <-route.quit:
			return
		default:
		}

		n, _, err := route.remote.ReadFrom(buf)
		if err != nil {
			select {
			case <-route.quit:
				return
			default:
				log.Println("Tailscale read error:", err)
				continue
			}
		}

		// Copy data for safe concurrent access
		data := make([]byte, n)
		copy(data, buf[:n])

		route.sessions.Range(func(_, v any) bool {
			s := v.(*udpSession)
			lastSeen := s.lastSeen.Load().(time.Time)

			if time.Since(lastSeen) > udpSessionTimeout {
				return true
			}

			_, err := route.listener.WriteTo(data, s.clientAddr)
			if err != nil {
				log.Printf("Public write error to %s: %v", s.clientAddr, err)
			} else {
				route.data.LogRecived(uint64(len(data)))
			}
			return true
		})
	}
}

func (route *UDPRoute) backendAddr() string {
	return fmt.Sprintf("%s:%d", route.config.Machine.Address, route.config.Machine.Port)
}

func (route *UDPRoute) cleanupStaleSessions() {
	defer route.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-route.quit:
			return
		case <-ticker.C:
			route.sessions.Range(func(key, value any) bool {
				s := value.(*udpSession)
				lastSeen := s.lastSeen.Load().(time.Time)
				if time.Since(lastSeen) > udpSessionTimeout {
					route.sessions.Delete(key)
					log.Printf("Session expired: %s", key)
				}
				return true
			})
		}
	}
}

func (route *UDPRoute) reader() {
	defer route.wg.Done()
	buf := make([]byte, udpBufferSize)
	for {
		select {
		case <-route.quit:
			return
		default:
		}

		n, clientAddr, err := route.listener.ReadFrom(buf)
		if err != nil {
			select {
			case <-route.quit:
				return
			default:
				log.Println("Public read error:", err)
				continue
			}
		}

		session := &udpSession{
			clientAddr: clientAddr,
		}
		session.lastSeen.Store(time.Now())

		// Load or store atomically - if exists, update lastSeen
		if existing, loaded := route.sessions.LoadOrStore(clientAddr.String(), session); loaded {
			existing.(*udpSession).lastSeen.Store(time.Now())
		}

		_, err = route.remote.WriteTo(buf[:n], route.remoteAddr)
		if err != nil {
			log.Println("Tailscale write error:", err)
		} else {
			route.data.LogSent(uint64(n))
		}
	}
}

func (route *UDPRoute) runHeartbeat() {
	defer route.wg.Done()
	ticker := time.NewTicker(udpHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-route.quit:
			return
		case <-ticker.C:
			route.measureLatency()
		}
	}
}

func (route *UDPRoute) measureLatency() {
	backendAddr := route.backendAddr()

	start := time.Now()
	conn, err := route.client.Dial(context.Background(), "udp", backendAddr)
	if err != nil {
		route.latencyMu.Lock()
		defer route.latencyMu.Unlock()
		route.latency = -1
		return
	}
	conn.Close()
	route.latency = time.Since(start)
}

func (route *UDPRoute) Ping() time.Duration {
	route.latencyMu.RLock()
	defer route.latencyMu.RUnlock()
	return route.latency
}
