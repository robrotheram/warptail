package router

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"
	"warptail/pkg/utils"
)

type TCPRoute struct {
	*connectionRoute
	listener net.Listener
	conns    map[net.Conn]struct{}
	clients  int64
}

func NewTCPRoute(config utils.RouteConfig, backend Backend) *TCPRoute {
	return &TCPRoute{connectionRoute: newConnectionRoute(config, backend)}
}
func (r *TCPRoute) Start() error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	return r.start()
}
func (r *TCPRoute) start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == RUNNING {
		return nil
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.config.Port))
	if err != nil {
		return err
	}
	r.listener = listener
	r.conns = make(map[net.Conn]struct{})
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.status = RUNNING
	r.wg.Add(2)
	go r.acceptLoop(r.ctx, listener, machineAddress(r.config.Machine))
	go r.runHeartbeat(r.ctx, func(ctx context.Context) (time.Duration, error) {
		start := time.Now()
		conn, err := r.backend.Dial(ctx, "tcp", machineAddress(r.Config().Machine))
		if err != nil {
			return -1, err
		}
		conn.Close()
		return time.Since(start), nil
	})
	return nil
}
func (r *TCPRoute) Stop() error {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	return r.stop()
}
func (r *TCPRoute) stop() error {
	r.mu.Lock()
	if r.status == STOPPED {
		r.mu.Unlock()
		return nil
	}
	r.status = STOPPING
	r.cancel()
	r.listener.Close()
	for conn := range r.conns {
		conn.Close()
	}
	r.mu.Unlock()
	r.wg.Wait()
	r.mu.Lock()
	r.status = STOPPED
	r.latency = -1
	r.mu.Unlock()
	return nil
}
func (r *TCPRoute) Update(config utils.RouteConfig) error {
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
func (r *TCPRoute) track(ctx context.Context, conn net.Conn) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		conn.Close()
		return false
	}
	r.conns[conn] = struct{}{}
	return true
}
func (r *TCPRoute) release(conn net.Conn) {
	conn.Close()
	r.mu.Lock()
	delete(r.conns, conn)
	r.mu.Unlock()
}
func (r *TCPRoute) acceptLoop(ctx context.Context, listener net.Listener, address string) {
	defer r.wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() == nil {
				utils.Logger.Error(err, "TCP accept failed")
			}
			return
		}
		if !r.track(ctx, conn) {
			return
		}
		r.mu.Lock()
		r.clients++
		r.mu.Unlock()
		r.wg.Add(1)
		go r.handleConnection(ctx, conn, address)
	}
}
func (r *TCPRoute) handleConnection(ctx context.Context, client net.Conn, address string) {
	defer r.wg.Done()
	defer r.release(client)
	defer func() { r.mu.Lock(); r.clients--; r.mu.Unlock() }()
	dialCtx, cancel := context.WithTimeout(ctx, backendDialTimeout)
	backend, err := r.backend.Dial(dialCtx, "tcp", address)
	cancel()
	if err != nil {
		if ctx.Err() == nil {
			utils.Logger.Error(err, "TCP backend connection failed", "address", address)
		}
		return
	}
	if !r.track(ctx, backend) {
		return
	}
	defer r.release(backend)
	done := make(chan struct{}, 2)
	copyStream := func(dst, src net.Conn, logBytes func(uint64)) {
		_, err := io.Copy(&countingWriter{Writer: dst, logBytes: logBytes}, src)
		if err == nil {
			if half, ok := dst.(interface{ CloseWrite() error }); ok {
				err = half.CloseWrite()
			} else {
				err = dst.Close()
			}
		}
		if err != nil {
			client.Close()
			backend.Close()
		}
		done <- struct{}{}
	}
	go copyStream(backend, client, r.data.LogSent)
	go copyStream(client, backend, r.data.LogRecived)
	<-done
	<-done
}

type countingWriter struct {
	io.Writer
	logBytes func(uint64)
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.logBytes(uint64(n))
	return n, err
}
func (r *TCPRoute) ActiveConnections() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients
}
