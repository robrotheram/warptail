package router

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"warptail/pkg/utils"

	tailscale "tailscale.com/client/local"
)

const (
	udpBufferSize  = 65535
	sessionTimeout = 2 * time.Minute
)

// udpSession represents a client session for UDP NAT traversal
type udpSession struct {
	clientAddr  *net.UDPAddr
	proxy       net.Conn
	lastActive  time.Time
	mu          sync.Mutex
	ready       chan struct{} // signals that first packet has been sent
	readyOnce   sync.Once
	done        chan struct{} // signals the session is done
	closed      bool
	packetsSent int
	packetsRecv int
}

func (s *udpSession) updateLastActive() {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()
}

func (s *udpSession) isExpired(timeout time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.lastActive) > timeout
}

type UDPRoute struct {
	config   utils.RouteConfig
	status   RouterStatus
	client   *tailscale.Client
	data     *utils.TimeSeries
	listener *net.UDPConn
	quit     chan bool
	exited   chan bool

	sessions   map[string]*udpSession
	sessionsMu sync.RWMutex

	latency  time.Duration
	heatbeat *time.Ticker
}

func NewUDPRoute(config utils.RouteConfig, client *tailscale.Client) *UDPRoute {
	return &UDPRoute{
		config:   config,
		data:     utils.NewTimeSeries(time.Second, 1000),
		status:   STOPPED,
		client:   client,
		sessions: make(map[string]*udpSession),
	}
}

func (route *UDPRoute) Status() RouterStatus {
	return route.status
}

func (route *UDPRoute) Config() utils.RouteConfig {
	return route.config
}

func (route *UDPRoute) Stats() utils.TimeSeriesData {
	return route.data.Data
}

func (route *UDPRoute) Update(config utils.RouteConfig) error {
	route.Stop()
	route.config = config
	return route.Start()
}

func (route *UDPRoute) Stop() error {
	if route.status != RUNNING {
		return fmt.Errorf("route not running")
	}
	route.status = STOPPING
	close(route.quit)
	<-route.exited
	utils.Logger.Info("Stopped UDP route successfully")
	route.status = STOPPED
	return nil
}

func (route *UDPRoute) Start() error {
	if route.status == RUNNING {
		route.Stop()
	}
	// Disable heartbeat for UDP - it creates extra connections that could interfere
	// go route.heartbeat(5 * time.Second)
	route.status = STARTING

	laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", route.config.Port))
	if err != nil {
		return err
	}

	route.quit = make(chan bool)
	route.exited = make(chan bool)
	route.sessions = make(map[string]*udpSession)

	route.listener, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	// Increase socket buffer sizes for better throughput
	route.listener.SetReadBuffer(4 * 1024 * 1024)  // 4MB
	route.listener.SetWriteBuffer(4 * 1024 * 1024) // 4MB

	go route.serve()
	go route.cleanupSessions()
	route.status = RUNNING
	return nil
}

func (route *UDPRoute) serve() {
	buffer := make([]byte, udpBufferSize)
	utils.Logger.Info("UDP route serving", "port", route.config.Port, "backend", fmt.Sprintf("%s:%d", route.config.Machine.Address, route.config.Machine.Port))

	for {
		select {
		case <-route.quit:
			utils.Logger.Info("Shutting down UDP route...")
			route.listener.Close()
			route.closeAllSessions()
			close(route.exited)
			return
		default:
			route.listener.SetDeadline(time.Now().Add(1 * time.Second))
			n, clientAddr, err := route.listener.ReadFromUDP(buffer)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				utils.Logger.Error(err, "failed to read UDP packet")
				continue
			}

			// Get or create session for this client
			session, err := route.getOrCreateSession(clientAddr)
			if err != nil {
				utils.Logger.Error(err, "failed to create session for client")
				continue
			}

			// Forward packet to backend synchronously to preserve order
			session.updateLastActive()
			data := make([]byte, n)
			copy(data, buffer[:n])

			route.forwardToBackend(session, data)
		}
	}
}

func (route *UDPRoute) getOrCreateSession(clientAddr *net.UDPAddr) (*udpSession, error) {
	key := clientAddr.String()

	route.sessionsMu.RLock()
	session, exists := route.sessions[key]
	route.sessionsMu.RUnlock()

	if exists {
		return session, nil
	}

	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	// Double-check after acquiring write lock
	if session, exists = route.sessions[key]; exists {
		return session, nil
	}

	// Create direct UDP connection to backend (Tailscale IP)
	// This maintains a stable source port unlike UserDial
	backendAddr := fmt.Sprintf("%s:%d", route.config.Machine.Address, route.config.Machine.Port)
	utils.Logger.Info("Creating new UDP session", "client", key, "backend", backendAddr)

	var proxy net.Conn
	var err error

	// Try direct UDP connection first (works if both on same Tailnet)
	proxy, err = net.Dial("udp", backendAddr)
	if err != nil {
		// Fall back to Tailscale UserDial
		utils.Logger.Info("Direct UDP failed, trying Tailscale UserDial", "error", err)
		proxy, err = route.client.UserDial(context.Background(), "udp", route.config.Machine.Address, route.config.Machine.Port)
		if err != nil {
			return nil, fmt.Errorf("failed to dial backend: %w", err)
		}
	}

	// Try to increase buffer sizes on the proxy connection
	if udpConn, ok := proxy.(*net.UDPConn); ok {
		udpConn.SetReadBuffer(4 * 1024 * 1024)
		udpConn.SetWriteBuffer(4 * 1024 * 1024)
	}

	session = &udpSession{
		clientAddr: clientAddr,
		proxy:      proxy,
		lastActive: time.Now(),
		ready:      make(chan struct{}),
		done:       make(chan struct{}),
	}
	route.sessions[key] = session

	// Start goroutine to forward responses back to client
	go route.forwardToClient(session)

	return session, nil
}

func (route *UDPRoute) forwardToBackend(session *udpSession, data []byte) {
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return
	}
	n, err := session.proxy.Write(data)
	session.packetsSent++
	session.mu.Unlock()

	// Signal that first packet has been sent
	session.readyOnce.Do(func() {
		close(session.ready)
	})

	if err != nil {
		utils.Logger.Error(err, "failed to forward packet to backend", "client", session.clientAddr.String())
		return
	}
	route.data.LogSent(uint64(n))
}

func (route *UDPRoute) forwardToClient(session *udpSession) {
	buffer := make([]byte, udpBufferSize)
	clientKey := session.clientAddr.String()

	// Wait for the first packet to be sent before we start reading
	select {
	case <-session.ready:
		utils.Logger.Info("Started response forwarder for client", "client", clientKey)
	case <-route.quit:
		return
	case <-time.After(sessionTimeout):
		utils.Logger.Info("Session timed out waiting for first packet", "client", clientKey)
		route.removeSession(clientKey)
		return
	}

	consecutiveEOFs := 0
	maxConsecutiveEOFs := 10

	for {
		// Check quit channel non-blocking
		select {
		case <-route.quit:
			utils.Logger.Info("Stopping response forwarder due to quit signal", "client", clientKey)
			return
		default:
		}

		// Set short read deadline for fast polling
		session.proxy.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		n, err := session.proxy.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Check if session expired
				if session.isExpired(sessionTimeout) {
					utils.Logger.Info("Session expired, removing", "client", clientKey)
					route.removeSession(clientKey)
					return
				}
				consecutiveEOFs = 0 // reset on timeout (normal)
				continue
			}

			// For EOF, don't reconnect - just wait and retry
			// Tailscale UDP connections may return EOF between packets
			if err == io.EOF {
				consecutiveEOFs++
				if consecutiveEOFs >= maxConsecutiveEOFs {
					// Too many EOFs, check if session is still active
					if session.isExpired(sessionTimeout) {
						utils.Logger.Info("Session expired after multiple EOFs", "client", clientKey)
						route.removeSession(clientKey)
						return
					}
					consecutiveEOFs = 0
				}
				// Small sleep to avoid busy loop, then continue reading
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Other connection error
			utils.Logger.Error(err, "Error reading from backend, closing session", "client", clientKey)
			route.removeSession(clientKey)
			return
		}

		consecutiveEOFs = 0
		session.mu.Lock()
		session.packetsRecv++
		session.mu.Unlock()
		session.updateLastActive()

		// Send response back to client
		_, err = route.listener.WriteToUDP(buffer[:n], session.clientAddr)
		if err != nil {
			continue
		}
		route.data.LogRecived(uint64(n))
	}
}

func (route *UDPRoute) removeSession(key string) {
	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	if session, exists := route.sessions[key]; exists {
		session.mu.Lock()
		if !session.closed {
			session.closed = true
			close(session.done)
		}
		session.mu.Unlock()
		session.proxy.Close()
		delete(route.sessions, key)
	}
}

func (route *UDPRoute) closeAllSessions() {
	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	for key, session := range route.sessions {
		session.mu.Lock()
		if !session.closed {
			session.closed = true
			close(session.done)
		}
		session.mu.Unlock()
		session.proxy.Close()
		delete(route.sessions, key)
	}
}

func (route *UDPRoute) cleanupSessions() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-route.quit:
			return
		case <-ticker.C:
			route.sessionsMu.Lock()
			for key, session := range route.sessions {
				if session.isExpired(sessionTimeout) {
					session.proxy.Close()
					delete(route.sessions, key)
					utils.Logger.Info("Cleaned up expired UDP session", "client", key)
				}
			}
			route.sessionsMu.Unlock()
		}
	}
}

func (route *UDPRoute) heartbeat(timeout time.Duration) {
	route.heatbeat = time.NewTicker(timeout)
	go func() {
		for range route.heatbeat.C {
			if route.status != RUNNING {
				route.latency = time.Duration(-1)
				route.heatbeat.Stop()
				continue
			}
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			conn, err := route.client.UserDial(ctx, "udp", route.config.Machine.Address, route.config.Machine.Port)
			cancel()
			if err != nil {
				route.latency = time.Duration(-1)
				continue
			}
			conn.Close()
			route.latency = time.Since(start)
		}
	}()
}

func (route *UDPRoute) Ping() time.Duration {
	return route.latency
}
