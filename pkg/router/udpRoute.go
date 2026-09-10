package router

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync/atomic"
	"time"
	"warptail/pkg/utils"
)

const (
	udpBufferSize     = 65535
	udpSessionTimeout = 30 * time.Second
	udpMaxSessions    = 1024
	udpQueueSize      = 64
)

// Each client owns a backend socket and a bounded send queue. Replies only
// return to that client; its backend source port persists until idle expiry.
type udpSession struct {
	clientAddr net.Addr
	conn       net.Conn // guarded by route.mu
	packets    chan []byte
	lastSeen   atomic.Int64
}
type UDPRoute struct {
	*connectionRoute
	listener       net.PacketConn
	sessions       map[string]*udpSession
	sessionTimeout time.Duration
}

func NewUDPRoute(config utils.RouteConfig, backend Backend) *UDPRoute {
	return &UDPRoute{connectionRoute: newConnectionRoute(config, backend), sessionTimeout: udpSessionTimeout}
}
func (r *UDPRoute) Start() error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	return r.start()
}
func (r *UDPRoute) start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == RUNNING {
		return nil
	}
	listener, err := net.ListenPacket("udp", fmt.Sprintf(":%d", r.config.Port))
	if err != nil {
		return err
	}
	r.listener = listener
	r.sessions = make(map[string]*udpSession)
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.status = RUNNING
	r.wg.Add(1)
	go r.readClients(r.ctx, listener, machineAddress(r.config.Machine))
	if pinger, ok := r.backend.(BackendPinger); ok {
		address := r.config.Machine.Address
		r.wg.Add(1)
		go r.runHeartbeat(r.ctx, func(ctx context.Context) (time.Duration, error) { return pinger.Ping(ctx, address) })
	}
	return nil
}
func (r *UDPRoute) Stop() error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	return r.stop()
}
func (r *UDPRoute) stop() error {
	r.mu.Lock()
	if r.status == STOPPED {
		r.mu.Unlock()
		return nil
	}
	r.status = STOPPING
	r.cancel()
	r.listener.Close()
	for _, session := range r.sessions {
		if session.conn != nil {
			session.conn.Close()
		}
	}
	r.mu.Unlock()
	r.wg.Wait()
	r.mu.Lock()
	r.status = STOPPED
	r.latency = -1
	r.mu.Unlock()
	return nil
}
func (r *UDPRoute) Update(config utils.RouteConfig) error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if reflect.DeepEqual(r.Config(), config) {
		return nil
	}
	running := r.Status() == RUNNING
	r.stop()
	r.mu.Lock()
	r.config = config
	r.mu.Unlock()
	if running {
		return r.start()
	}
	return nil
}
func (r *UDPRoute) readClients(ctx context.Context, listener net.PacketConn, address string) {
	defer r.wg.Done()
	buf := make([]byte, udpBufferSize)
	for {
		n, clientAddr, err := listener.ReadFrom(buf)
		if err != nil {
			if ctx.Err() == nil {
				utils.Logger.Error(err, "UDP listener failed")
			}
			return
		}
		key := clientAddr.String()
		r.mu.Lock()
		if ctx.Err() != nil {
			r.mu.Unlock()
			return
		}
		session := r.sessions[key]
		if session == nil {
			if len(r.sessions) >= udpMaxSessions {
				r.mu.Unlock()
				continue
			}
			session = &udpSession{clientAddr: clientAddr, packets: make(chan []byte, udpQueueSize)}
			session.lastSeen.Store(time.Now().UnixNano())
			r.sessions[key] = session
			r.wg.Add(1)
			go r.serveSession(ctx, listener, address, key, session)
		}
		session.lastSeen.Store(time.Now().UnixNano())
		packet := append([]byte(nil), buf[:n]...)
		select {
		case session.packets <- packet:
		default: // Drop excess UDP packets without blocking other clients.
		}
		r.mu.Unlock()
	}
}
func (r *UDPRoute) serveSession(ctx context.Context, listener net.PacketConn, address, key string, session *udpSession) {
	defer r.wg.Done()
	defer func() { r.mu.Lock(); delete(r.sessions, key); r.mu.Unlock() }()
	dialCtx, cancel := context.WithTimeout(ctx, backendDialTimeout)
	conn, err := r.backend.Dial(dialCtx, "udp", address)
	cancel()
	if err != nil {
		if ctx.Err() == nil {
			utils.Logger.Error(err, "UDP backend connection failed", "address", address)
		}
		return
	}
	defer conn.Close()
	r.mu.Lock()
	if ctx.Err() != nil {
		r.mu.Unlock()
		return
	}
	session.conn = conn
	r.mu.Unlock()
	readDone := make(chan struct{})
	go func() { defer close(readDone); r.readReplies(listener, session, conn) }()
	defer func() { conn.Close(); <-readDone }()
	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case packet := <-session.packets:
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Write(packet)
			if err != nil {
				return
			}
			r.data.LogSent(uint64(n))
		}
	}
}
func (r *UDPRoute) readReplies(listener net.PacketConn, session *udpSession, conn net.Conn) {
	buf := make([]byte, udpBufferSize)
	for {
		conn.SetReadDeadline(time.Unix(0, session.lastSeen.Load()).Add(r.sessionTimeout))
		n, err := conn.Read(buf)
		if err != nil {
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() && time.Since(time.Unix(0, session.lastSeen.Load())) < r.sessionTimeout {
				continue // Client traffic extended the session while Read was blocked.
			}
			return
		}
		session.lastSeen.Store(time.Now().UnixNano())
		written, err := listener.WriteTo(buf[:n], session.clientAddr)
		if err != nil {
			return
		}
		r.data.LogRecived(uint64(written))
	}
}
