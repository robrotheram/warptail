package router

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
	"warptail/pkg/utils"
)

type dialFunc func(context.Context, string, string) (net.Conn, error)

func (f dialFunc) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func targetMachine(t *testing.T, address string) utils.Machine {
	t.Helper()
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return utils.Machine{Address: host, Port: uint16(number)}
}
func publicAddress(t *testing.T, addr net.Addr) string {
	t.Helper()
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		t.Fatal(err)
	}
	return net.JoinHostPort("127.0.0.1", port)
}
func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition did not become true")
		}
		time.Sleep(time.Millisecond)
	}
}
func stopWithin(t *testing.T, route Route) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- route.Stop() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("route shutdown hung")
	}
	if route.Status() != STOPPED {
		t.Fatalf("route status = %s", route.Status())
	}
}

func TestTCPHalfCloseAndConcurrentClients(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				// Deliberately wait for EOF before responding: requires a forwarded FIN.
				request, err := io.ReadAll(conn)
				if err != nil {
					return
				}
				conn.Write(append([]byte("reply:"), request...))
			}()
		}
	}()
	defer func() { listener.Close(); workers.Wait() }()
	route := NewTCPRoute(utils.RouteConfig{Type: utils.TCP, Machine: targetMachine(t, listener.Addr().String())}, &SystemBackend{})
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	defer route.Stop()
	address := publicAddress(t, route.listener.Addr())
	var clients sync.WaitGroup
	for i := range 8 {
		clients.Add(1)
		go func() {
			defer clients.Done()
			conn, err := net.Dial("tcp", address)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			payload := bytes.Repeat([]byte(fmt.Sprintf("client-%d/", i)), 16384)
			if _, err := conn.Write(payload); err != nil {
				t.Error(err)
				return
			}
			if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
				t.Error(err)
				return
			}
			response, err := io.ReadAll(conn)
			if err != nil {
				t.Error(err)
				return
			}
			if !bytes.Equal(response, append([]byte("reply:"), payload...)) {
				t.Errorf("client %d response differs", i)
			}
		}()
	}
	// Exercise statistics snapshots during concurrent forwarding.
	snapshotsDone := make(chan struct{})
	go func() {
		defer close(snapshotsDone)
		for range 1000 {
			_ = route.Stats()
			_ = route.Ping()
			_ = route.ActiveConnections()
		}
	}()
	clients.Wait()
	<-snapshotsDone
	waitUntil(t, func() bool { return route.ActiveConnections() == 0 })
	totals := route.Stats().Total
	if totals.Sent == 0 || totals.Received != totals.Sent+8*6 {
		t.Fatalf("unexpected stats: %+v", totals)
	}
	stopWithin(t, route)
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	stopWithin(t, route)
}

func TestTCPStopClosesBothSides(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	route := NewTCPRoute(utils.RouteConfig{Type: utils.TCP, Machine: targetMachine(t, listener.Addr().String())}, &SystemBackend{})
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	defer route.Stop()
	client, err := net.Dial("tcp", publicAddress(t, route.listener.Addr()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var backend net.Conn
	select {
	case backend = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("backend was not connected")
	}
	defer backend.Close()
	stopWithin(t, route)
	for _, conn := range []net.Conn{client, backend} {
		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, err := conn.Read(make([]byte, 1))
		if err == nil {
			t.Fatal("connection remained open")
		}
		if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			t.Fatal("connection was not closed by Stop")
		}
	}
	if route.ActiveConnections() != 0 {
		t.Fatal("active connections remain")
	}
}

func TestUDPClientIsolationDatagramsAndSessionReuse(t *testing.T) {
	backend, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	route := NewUDPRoute(utils.RouteConfig{Type: utils.UDP, Machine: targetMachine(t, backend.LocalAddr().String())}, &SystemBackend{})
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	defer route.Stop()
	clients := make([]net.Conn, 2)
	for i := range clients {
		clients[i], err = net.Dial("udp", publicAddress(t, route.listener.LocalAddr()))
		if err != nil {
			t.Fatal(err)
		}
		defer clients[i].Close()
	}
	readBackend := func(expected []byte) net.Addr {
		t.Helper()
		backend.SetReadDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, udpBufferSize)
		n, addr, err := backend.ReadFrom(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf[:n], expected) {
			t.Fatalf("backend datagram size/content differs: got %d, want %d", n, len(expected))
		}
		return addr
	}
	readClient := func(client net.Conn, expected []byte) {
		t.Helper()
		client.SetReadDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, udpBufferSize)
		n, err := client.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf[:n], expected) {
			t.Fatalf("reply delivered incorrectly: got %q, want %q", buf[:n], expected)
		}
	}
	peers := make([]net.Addr, 2)
	for i, client := range clients {
		packet := []byte(fmt.Sprintf("request-%d", i))
		if _, err := client.Write(packet); err != nil {
			t.Fatal(err)
		}
		peers[i] = readBackend(packet)
	}
	if peers[0].String() == peers[1].String() {
		t.Fatal("clients share a backend source port")
	}
	// Reverse reply order so broadcasting or a last-client-wins implementation fails.
	for i := len(clients) - 1; i >= 0; i-- {
		reply := []byte(fmt.Sprintf("response-%d", i))
		if _, err := backend.WriteTo(reply, peers[i]); err != nil {
			t.Fatal(err)
		}
		readClient(clients[i], reply)
	}
	for _, payload := range [][]byte{{}, bytes.Repeat([]byte{0xa5}, 60000), []byte("another packet")} {
		if _, err := clients[0].Write(payload); err != nil {
			t.Fatal(err)
		}
		peer := readBackend(payload)
		if peer.String() != peers[0].String() {
			t.Fatal("session source port changed")
		}
		if _, err := backend.WriteTo(payload, peer); err != nil {
			t.Fatal(err)
		}
		readClient(clients[0], payload)
	}
	clients[1].SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if _, err := clients[1].Read(make([]byte, udpBufferSize)); err == nil {
		t.Fatal("client received another client's reply")
	}
	stopWithin(t, route)
	route.mu.RLock()
	remaining := len(route.sessions)
	route.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("%d sessions remain after Stop", remaining)
	}
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	stopWithin(t, route)
}

func TestUDPIdleExpiryAndBackendActivity(t *testing.T) {
	backend, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	route := NewUDPRoute(utils.RouteConfig{Type: utils.UDP, Machine: targetMachine(t, backend.LocalAddr().String())}, &SystemBackend{})
	route.sessionTimeout = 200 * time.Millisecond
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	defer route.Stop()
	client, err := net.Dial("udp", publicAddress(t, route.listener.LocalAddr()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	client.Write([]byte("subscribe"))
	backend.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 64)
	_, peer, err := backend.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	// Server-only traffic must keep a subscription alive beyond the idle timeout.
	for range 6 {
		time.Sleep(50 * time.Millisecond)
		if _, err := backend.WriteTo([]byte("event"), peer); err != nil {
			t.Fatal(err)
		}
		client.SetReadDeadline(time.Now().Add(time.Second))
		if _, err := client.Read(buf); err != nil {
			t.Fatal(err)
		}
	}
	waitUntil(t, func() bool { route.mu.RLock(); defer route.mu.RUnlock(); return len(route.sessions) == 0 })
	client.Write([]byte("reconnect"))
	backend.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := backend.ReadFrom(buf); err != nil {
		t.Fatalf("expired session did not reconnect: %v", err)
	}
}

func TestStopCancelsPendingDials(t *testing.T) {
	for _, protocol := range []utils.RouteType{utils.TCP, utils.UDP} {
		t.Run(string(protocol), func(t *testing.T) {
			dialStarted := make(chan struct{}, 1)
			backend := dialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
				if network != string(protocol) || address != "backend.example:8080" {
					t.Errorf("unexpected dial %s %s", network, address)
				}
				dialStarted <- struct{}{}
				<-ctx.Done()
				return nil, ctx.Err()
			})
			route, err := NewRoute(utils.RouteConfig{Type: protocol, Machine: utils.Machine{Address: "backend.example", Port: 8080}}, backend)
			if err != nil {
				t.Fatal(err)
			}
			if err := route.Start(); err != nil {
				t.Fatal(err)
			}
			defer route.Stop()
			var addr net.Addr
			switch r := route.(type) {
			case *TCPRoute:
				addr = r.listener.Addr()
			case *UDPRoute:
				addr = r.listener.LocalAddr()
			}
			conn, err := net.Dial(string(protocol), publicAddress(t, addr))
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.Write([]byte("hello"))
			select {
			case <-dialStarted:
			case <-time.After(3 * time.Second):
				t.Fatal("dial did not start")
			}
			stopWithin(t, route)
		})
	}
}

func TestRoutesLifecycleAndDisabledUpdates(t *testing.T) {
	for _, protocol := range []utils.RouteType{utils.TCP, utils.UDP, utils.HTTP} {
		t.Run(string(protocol), func(t *testing.T) {
			config := utils.RouteConfig{Type: protocol, Machine: utils.Machine{Address: "127.0.0.1", Port: 9}}
			route, err := NewRoute(config, &SystemBackend{})
			if err != nil {
				t.Fatal(err)
			}
			if err := route.Update(config); err != nil {
				t.Fatal(err)
			}
			if route.Status() != STOPPED {
				t.Fatal("updating a disabled route started it")
			}
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range 4 {
						if err := route.Start(); err != nil {
							t.Error(err)
						}
						if err := route.Update(config); err != nil {
							t.Error(err)
						}
						if err := route.Stop(); err != nil {
							t.Error(err)
						}
						_ = route.Stats()
						_ = route.Ping()
					}
				}()
			}
			wg.Wait()
			stopWithin(t, route)
		})
	}
}

func TestServiceStartErrorsAndEnabledState(t *testing.T) {
	router := NewRouterWithBackend(&SystemBackend{})
	blocked, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer blocked.Close()
	cfg := utils.ServiceConfig{Name: "blocked", Enabled: true, Routes: []utils.RouteConfig{{Type: utils.TCP, Port: blocked.Addr().(*net.TCPAddr).Port}}}
	if _, err := router.Create(cfg); err == nil {
		t.Fatal("listener bind error was discarded")
	}
	if router.DoesExists("blocked") {
		t.Fatal("failed service was retained")
	}
	cfg.Name = "disabled"
	cfg.Enabled = false
	cfg.Routes[0].Port = 0
	svc, routeErr := router.Create(cfg)
	if routeErr != nil {
		t.Fatal(routeErr)
	}
	defer router.StopAll()
	if _, err := router.Update(svc.Id, cfg); err != nil {
		t.Fatal(err)
	}
	router.StartAll()
	if svc.Enabled || svc.Routes[0].Status() != STOPPED {
		t.Fatal("disabled service was started")
	}
	cfg.Enabled = true
	if _, err := router.Update(svc.Id, cfg); err != nil {
		t.Fatal(err)
	}
	router.StopAll()
	if !svc.Enabled {
		t.Fatal("network shutdown changed saved enabled state")
	}
	router.StartAll()
	if svc.Routes[0].Status() != RUNNING {
		t.Fatal("enabled service was not restarted")
	}
	cfg.Enabled = false
	if _, err := router.Update(svc.Id, cfg); err != nil {
		t.Fatal(err)
	}
	if svc.Routes[0].Status() != STOPPED {
		t.Fatal("service update reactivated disabled route")
	}
}

func TestHTTPUsesBackendAndIPv6Address(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "via backend") }))
	defer upstream.Close()
	machine := targetMachine(t, upstream.Listener.Addr().String())
	backend := dialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" || address != "[fd7a:115c:a1e0::1]:8080" {
			t.Errorf("unexpected dial %s %s", network, address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, machineAddress(machine))
	})
	route := NewHTTPRoute(utils.RouteConfig{Type: utils.HTTP, Machine: utils.Machine{Address: "fd7a:115c:a1e0::1", Port: 8080}}, backend)
	route.Start()
	defer route.Stop()
	request := httptest.NewRequest(http.MethodGet, "http://proxy.example/test", nil)
	response := httptest.NewRecorder()
	route.Handle(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "via backend" {
		t.Fatalf("response: %d %q", response.Code, response.Body.String())
	}
}

func TestTailscaleStatusBeforeInitialization(t *testing.T) {
	router := NewRouter()
	_ = router.GetTailScaleStatus()
	_ = router.GetTailScaleConfig()
	if _, err := router.GetPeers(); err == nil {
		t.Fatal("expected uninitialized status error")
	}
}

func TestServiceFailedUpdateRestoresPreviousRoutes(t *testing.T) {
	router := NewRouterWithBackend(&SystemBackend{})
	cfg := utils.ServiceConfig{Name: "original", Enabled: true, Routes: []utils.RouteConfig{{Type: utils.TCP}}}
	svc, err := router.Create(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer router.StopAll()
	blocked, listenErr := net.Listen("tcp", ":0")
	if listenErr != nil {
		t.Fatal(listenErr)
	}
	defer blocked.Close()
	changed := utils.ServiceConfig{Name: "renamed", Enabled: true, Routes: []utils.RouteConfig{{Type: utils.TCP, Port: blocked.Addr().(*net.TCPAddr).Port}}}
	if _, err := router.Update(svc.Id, changed); err == nil {
		t.Fatal("expected bind error")
	}
	if svc.Id != "original" || svc.Name != "original" || !svc.Enabled || svc.Routes[0].Status() != RUNNING {
		t.Fatalf("original service was not restored: %+v", svc)
	}
	if !router.DoesExists("original") || router.DoesExists("renamed") {
		t.Fatal("service identity changed after failed update")
	}
}

func TestUnchangedTCPUpdateKeepsConnections(t *testing.T) {
	connected := make(chan net.Conn, 1)
	backend := dialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
		a, b := net.Pipe()
		connected <- b
		return a, nil
	})
	cfg := utils.RouteConfig{Type: utils.TCP}
	route := NewTCPRoute(cfg, backend)
	if err := route.Start(); err != nil {
		t.Fatal(err)
	}
	defer route.Stop()
	client, err := net.Dial("tcp", publicAddress(t, route.listener.Addr()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var peer net.Conn
	select {
	case peer = <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("dial did not happen")
	}
	defer peer.Close()
	if err := route.Update(cfg); err != nil {
		t.Fatal(err)
	}
	written := make(chan error, 1)
	go func() { _, err := io.WriteString(peer, "still connected"); written <- err }()
	client.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, len("still connected"))
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatal(err)
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
}

func TestRouterSnapshotsDuringRestarts(t *testing.T) {
	router := NewRouterWithBackend(&SystemBackend{})
	_, err := router.Create(utils.ServiceConfig{Name: "service", Enabled: true, Routes: []utils.RouteConfig{{Type: utils.HTTP}}})
	if err != nil {
		t.Fatal(err)
	}
	defer router.StopAll()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			router.StopAll()
			router.StartAll()
		}
	}()
	for range 1000 {
		for _, svc := range router.All() {
			_ = svc.Status(true)
		}
	}
	<-done
}
