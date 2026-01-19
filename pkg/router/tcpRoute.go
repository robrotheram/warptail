package router

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
	"warptail/pkg/utils"

	tailscale "tailscale.com/tsnet"
)

const (
	tcpBufferSize        = 32 * 1024 // 32KB buffer for TCP
	tcpHeartbeatInterval = 5 * time.Second
)

// TCPRoute handles TCP traffic proxying through Tailscale.
type TCPRoute struct {
	config utils.RouteConfig
	client *tailscale.Server
	data   *utils.TimeSeries

	mu       sync.RWMutex
	status   RouterStatus
	listener net.Listener

	quit   chan struct{}
	cancel context.CancelFunc
	ctx    context.Context
	wg     sync.WaitGroup

	activeConns sync.Map // tracks active connections for graceful shutdown
	connCount   int64
	connCountMu sync.Mutex

	latency   time.Duration
	latencyMu sync.RWMutex
}

func NewTCPRoute(config utils.RouteConfig, client *tailscale.Server) *TCPRoute {
	return &TCPRoute{
		config: config,
		data:   utils.NewTimeSeries(time.Second, 1000),
		status: STOPPED,
		client: client,
	}
}

func (route *TCPRoute) Status() RouterStatus {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.status
}

func (route *TCPRoute) Config() utils.RouteConfig {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.config
}

func (route *TCPRoute) Stats() utils.TimeSeriesData {
	return route.data.Data
}

func (route *TCPRoute) Update(config utils.RouteConfig) error {
	route.Stop()
	route.mu.Lock()
	route.config = config
	route.mu.Unlock()
	return route.Start()
}

func (route *TCPRoute) Stop() error {
	route.mu.Lock()
	if route.status != RUNNING {
		route.mu.Unlock()
		return fmt.Errorf("route not running")
	}
	route.status = STOPPING

	// Cancel context to stop all dials
	if route.cancel != nil {
		route.cancel()
	}

	// Close listener to stop accepting new connections
	if route.listener != nil {
		route.listener.Close()
	}

	// Close all active connections
	route.activeConns.Range(func(key, value any) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		return true
	})

	// Signal all goroutines to stop
	close(route.quit)
	route.mu.Unlock()

	// Wait for all goroutines to finish
	route.wg.Wait()

	route.mu.Lock()
	route.status = STOPPED
	route.mu.Unlock()

	utils.Logger.Info("Stopped TCP route", "port", route.config.Port)
	return nil
}

func (route *TCPRoute) Start() error {
	route.mu.Lock()
	if route.status == RUNNING {
		route.mu.Unlock()
		route.Stop()
		route.mu.Lock()
	}

	route.status = STARTING
	route.quit = make(chan struct{})
	route.ctx, route.cancel = context.WithCancel(context.Background())

	laddr := fmt.Sprintf(":%d", route.config.Port)

	var err error
	route.listener, err = net.Listen("tcp", laddr)
	if err != nil {
		route.status = STOPPED
		route.mu.Unlock()
		return err
	}

	route.wg.Add(2)
	go route.acceptLoop()
	go route.runHeartbeat()

	route.status = RUNNING
	route.mu.Unlock()
	return nil
}

func (route *TCPRoute) acceptLoop() {
	defer route.wg.Done()

	for {
		select {
		case <-route.quit:
			return
		default:
		}

		conn, err := route.listener.Accept()
		if err != nil {
			select {
			case <-route.quit:
				return
			default:
				log.Println("TCP accept error:", err)
				continue
			}
		}

		// Track connection
		connID := fmt.Sprintf("%p", conn)
		route.activeConns.Store(connID, conn)

		route.connCountMu.Lock()
		route.connCount++
		route.connCountMu.Unlock()

		go route.handleConnection(conn, connID)
	}
}

func (route *TCPRoute) handleConnection(clientConn net.Conn, connID string) {
	defer func() {
		clientConn.Close()
		route.activeConns.Delete(connID)

		route.connCountMu.Lock()
		route.connCount--
		route.connCountMu.Unlock()
	}()

	// Connect to backend through Tailscale
	backendAddr := route.backendAddr()
	backendConn, err := route.client.Dial(route.ctx, "tcp", backendAddr)
	if err != nil {
		log.Printf("Failed to connect to backend %s: %v", backendAddr, err)
		return
	}
	defer backendConn.Close()

	// Bidirectional copy with stats tracking
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Backend
	go func() {
		defer wg.Done()
		sent := route.copyWithStats(backendConn, clientConn)
		route.data.LogSent(uint64(sent))
	}()

	// Backend -> Client
	go func() {
		defer wg.Done()
		received := route.copyWithStats(clientConn, backendConn)
		route.data.LogRecived(uint64(received))
	}()

	wg.Wait()
}

func (route *TCPRoute) copyWithStats(dst, src net.Conn) int64 {
	buf := make([]byte, tcpBufferSize)
	var totalBytes int64

	for {
		select {
		case <-route.quit:
			return totalBytes
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			if written > 0 {
				totalBytes += int64(written)
			}
			if writeErr != nil {
				return totalBytes
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				// Only log if it's not a normal close
				select {
				case <-route.quit:
				default:
					// Connection closed by peer is normal
				}
			}
			return totalBytes
		}
	}
}

func (route *TCPRoute) backendAddr() string {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return fmt.Sprintf("%s:%d", route.config.Machine.Address, route.config.Machine.Port)
}

func (route *TCPRoute) runHeartbeat() {
	defer route.wg.Done()
	ticker := time.NewTicker(tcpHeartbeatInterval)
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

func (route *TCPRoute) measureLatency() {
	backendAddr := route.backendAddr()

	start := time.Now()
	conn, err := net.DialTimeout("tcp", backendAddr, time.Second)

	route.latencyMu.Lock()
	defer route.latencyMu.Unlock()

	if err != nil {
		route.latency = -1
		return
	}
	conn.Close()
	route.latency = time.Since(start)
}

func (route *TCPRoute) Ping() time.Duration {
	route.latencyMu.RLock()
	defer route.latencyMu.RUnlock()
	return route.latency
}

// ActiveConnections returns the current number of active TCP connections
func (route *TCPRoute) ActiveConnections() int64 {
	route.connCountMu.Lock()
	defer route.connCountMu.Unlock()
	return route.connCount
}
