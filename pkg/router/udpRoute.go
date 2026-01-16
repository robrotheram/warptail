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
	udpBufferSize        = 65535
	udpSessionTimeout    = 2 * time.Minute
	udpSocketBufferSize  = 4 * 1024 * 1024 // 4MB
	udpReadTimeout       = 10 * time.Millisecond
	udpHeartbeatInterval = 5 * time.Second
)

// udpSession represents a client session for UDP NAT traversal.
// Each client gets its own session with a dedicated backend connection
// to maintain consistent source ports for protocols like QUIC.
type udpSession struct {
	clientAddr *net.UDPAddr
	proxy      net.Conn
	lastActive time.Time
	mu         sync.Mutex
	ready      chan struct{}
	readyOnce  sync.Once
	closed     bool
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

func (s *udpSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.proxy.Close()
	}
}

func (s *udpSession) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// UDPRoute handles UDP traffic proxying through Tailscale.
// It maintains per-client sessions to preserve connection identity
// for stateful UDP protocols like QUIC.
type UDPRoute struct {
	config   utils.RouteConfig
	status   RouterStatus
	client   *tailscale.Client
	data     *utils.TimeSeries
	listener *net.UDPConn

	quit   chan struct{}
	exited chan struct{}

	sessions   map[string]*udpSession
	sessionsMu sync.RWMutex

	latency   time.Duration
	latencyMu sync.RWMutex
	heartbeat *time.Ticker
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

	if route.heartbeat != nil {
		route.heartbeat.Stop()
	}

	close(route.quit)
	<-route.exited

	utils.Logger.Info("Stopped UDP route", "port", route.config.Port)
	route.status = STOPPED
	return nil
}

func (route *UDPRoute) Start() error {
	if route.status == RUNNING {
		route.Stop()
	}
	route.status = STARTING

	laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", route.config.Port))
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	route.listener, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Increase socket buffer sizes for better throughput
	route.listener.SetReadBuffer(udpSocketBufferSize)
	route.listener.SetWriteBuffer(udpSocketBufferSize)

	route.quit = make(chan struct{})
	route.exited = make(chan struct{})
	route.sessions = make(map[string]*udpSession)

	go route.serve()
	go route.cleanupSessions()
	go route.runHeartbeat()

	route.status = RUNNING
	utils.Logger.Info("UDP route started", "port", route.config.Port, "backend", route.backendAddr())
	return nil
}

func (route *UDPRoute) backendAddr() string {
	return fmt.Sprintf("%s:%d", route.config.Machine.Address, route.config.Machine.Port)
}

func (route *UDPRoute) serve() {
	buffer := make([]byte, udpBufferSize)

	for {
		select {
		case <-route.quit:
			route.listener.Close()
			route.closeAllSessions()
			close(route.exited)
			return
		default:
		}

		route.listener.SetDeadline(time.Now().Add(1 * time.Second))
		n, clientAddr, err := route.listener.ReadFromUDP(buffer)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			utils.Logger.Error(err, "failed to read UDP packet")
			continue
		}

		session, err := route.getOrCreateSession(clientAddr)
		if err != nil {
			utils.Logger.Error(err, "failed to create session", "client", clientAddr)
			continue
		}

		// Forward packet synchronously to preserve ordering
		session.updateLastActive()
		route.forwardToBackend(session, buffer[:n])
	}
}

func (route *UDPRoute) getOrCreateSession(clientAddr *net.UDPAddr) (*udpSession, error) {
	key := clientAddr.String()

	// Fast path: check if session exists
	route.sessionsMu.RLock()
	session, exists := route.sessions[key]
	route.sessionsMu.RUnlock()
	if exists && !session.isClosed() {
		return session, nil
	}

	// Slow path: create new session
	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	// Double-check after acquiring write lock
	if session, exists = route.sessions[key]; exists && !session.isClosed() {
		return session, nil
	}

	// Create connection to backend
	proxy, err := route.dialBackend()
	if err != nil {
		return nil, err
	}

	session = &udpSession{
		clientAddr: clientAddr,
		proxy:      proxy,
		lastActive: time.Now(),
		ready:      make(chan struct{}),
	}
	route.sessions[key] = session

	go route.forwardToClient(session)

	utils.Logger.Info("Created UDP session", "client", key, "backend", route.backendAddr())
	return session, nil
}

func (route *UDPRoute) dialBackend() (net.Conn, error) {
	backendAddr := route.backendAddr()

	// Check if this is a Tailscale IP (100.64.0.0/10 CGNAT range used by Tailscale)
	// If so, we must use UserDial to route through Tailscale
	backendIP := net.ParseIP(route.config.Machine.Address)
	isTailscaleIP := backendIP != nil && backendIP.To4() != nil &&
		backendIP.To4()[0] == 100 && (backendIP.To4()[1]&0xC0) == 64

	if isTailscaleIP {
		// Use Tailscale UserDial for Tailscale IPs
		conn, err := route.client.UserDial(context.Background(), "udp", route.config.Machine.Address, route.config.Machine.Port)
		if err != nil {
			return nil, fmt.Errorf("failed to dial backend via Tailscale: %w", err)
		}
		return conn, nil
	}

	// Try direct UDP connection for non-Tailscale addresses
	conn, err := net.Dial("udp", backendAddr)
	if err == nil {
		if udpConn, ok := conn.(*net.UDPConn); ok {
			udpConn.SetReadBuffer(udpSocketBufferSize)
			udpConn.SetWriteBuffer(udpSocketBufferSize)
		}
		return conn, nil
	}

	// Fall back to Tailscale UserDial
	conn, err = route.client.UserDial(context.Background(), "udp", route.config.Machine.Address, route.config.Machine.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to dial backend: %w", err)
	}
	return conn, nil
}

func (route *UDPRoute) forwardToBackend(session *udpSession, data []byte) {
	if session.isClosed() {
		return
	}

	session.mu.Lock()
	n, err := session.proxy.Write(data)
	session.mu.Unlock()

	// Signal ready after first packet sent
	session.readyOnce.Do(func() {
		close(session.ready)
	})

	if err != nil {
		utils.Logger.Error(err, "failed to forward to backend", "client", session.clientAddr)
		return
	}
	route.data.LogSent(uint64(n))
}

func (route *UDPRoute) forwardToClient(session *udpSession) {
	buffer := make([]byte, udpBufferSize)
	clientKey := session.clientAddr.String()

	// Wait for first packet before reading responses
	select {
	case <-session.ready:
	case <-route.quit:
		return
	case <-time.After(udpSessionTimeout):
		route.removeSession(clientKey)
		return
	}

	consecutiveEOFs := 0

	for {
		select {
		case <-route.quit:
			return
		default:
		}

		session.proxy.SetReadDeadline(time.Now().Add(udpReadTimeout))
		n, err := session.proxy.Read(buffer)

		if err != nil {
			if isTimeout(err) {
				if session.isExpired(udpSessionTimeout) {
					route.removeSession(clientKey)
					return
				}
				consecutiveEOFs = 0
				continue
			}

			if err == io.EOF {
				consecutiveEOFs++
				if consecutiveEOFs >= 10 && session.isExpired(udpSessionTimeout) {
					route.removeSession(clientKey)
					return
				}
				time.Sleep(udpReadTimeout)
				continue
			}

			route.removeSession(clientKey)
			return
		}

		consecutiveEOFs = 0
		session.updateLastActive()

		if _, err = route.listener.WriteToUDP(buffer[:n], session.clientAddr); err == nil {
			route.data.LogRecived(uint64(n))
		}
	}
}

func (route *UDPRoute) removeSession(key string) {
	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	if session, exists := route.sessions[key]; exists {
		session.close()
		delete(route.sessions, key)
	}
}

func (route *UDPRoute) closeAllSessions() {
	route.sessionsMu.Lock()
	defer route.sessionsMu.Unlock()

	for key, session := range route.sessions {
		session.close()
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
				if session.isExpired(udpSessionTimeout) {
					session.close()
					delete(route.sessions, key)
					utils.Logger.Info("Cleaned up expired UDP session", "client", key)
				}
			}
			route.sessionsMu.Unlock()
		}
	}
}

func (route *UDPRoute) runHeartbeat() {
	route.heartbeat = time.NewTicker(udpHeartbeatInterval)

	for {
		select {
		case <-route.quit:
			return
		case <-route.heartbeat.C:
			route.measureLatency()
		}
	}
}

func (route *UDPRoute) measureLatency() {
	start := time.Now()
	conn, err := net.DialTimeout("udp", route.backendAddr(), time.Second)

	route.latencyMu.Lock()
	defer route.latencyMu.Unlock()

	if err != nil {
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

// isTimeout checks if an error is a network timeout
func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
