//go:build integration

package router

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/ipn/store/mem"
	"tailscale.com/net/netns"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
	"tailscale.com/tstest/integration"
	"tailscale.com/tstest/integration/testcontrol"
	"tailscale.com/types/logger"
)

// Uses actual tsnet nodes and encrypted connections with a local control/DERP
// server. No Tailscale account, auth keys, root access or production tailnet.
func TestTailscaleProxyIntegration(t *testing.T) {
	t.Setenv("TS_NO_LOGS_NO_SUPPORT", "true")
	netns.SetEnabled(false)
	t.Cleanup(func() { netns.SetEnabled(true) })
	control := &testcontrol.Server{
		DERPMap:        integration.RunDERPAndSTUN(t, logger.Discard, "127.0.0.1"),
		DNSConfig:      &tailcfg.DNSConfig{Proxied: true},
		MagicDNSDomain: "test.ts.net",
		Logf:           t.Logf,
	}
	control.HTTPTestServer = httptest.NewServer(control)
	t.Cleanup(control.HTTPTestServer.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	startNode := func(name string) (*tsnet.Server, []netip.Addr) {
		t.Helper()
		server := &tsnet.Server{Dir: t.TempDir(), Store: new(mem.Store), Hostname: name,
			ControlURL: control.HTTPTestServer.URL, Ephemeral: true, Logf: t.Logf, UserLogf: t.Logf}
		t.Cleanup(func() { server.Close() })
		status, err := server.Up(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return server, status.TailscaleIPs
	}
	target, ips := startNode("target")
	source, _ := startNode("proxy")
	backend := &tailscaleBackend{ts: source}
	// Establish connectivity before sending UDP, which cannot retry a lost handshake packet.
	if _, err := backend.Ping(ctx, ips[0].String()); err != nil {
		t.Fatal(err)
	}
	targets := []string{ips[0].String(), "target.test.ts.net"}
	for _, ip := range ips {
		if ip.Is6() {
			targets = append(targets, ip.String())
			break
		}
	}
	for _, host := range targets {
		t.Run(host, func(t *testing.T) {
			tcp, err := target.Listen("tcp", ":8080")
			if err != nil {
				t.Fatal(err)
			}
			defer tcp.Close()
			serverDone := make(chan error, 1)
			go func() {
				conn, err := tcp.Accept()
				if err != nil {
					serverDone <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				payload, err := io.ReadAll(conn)
				if err == nil {
					_, err = conn.Write(append([]byte("reply:"), payload...))
				}
				serverDone <- err
			}()
			route := NewTCPRoute(utils.RouteConfig{Type: utils.TCP, Machine: utils.Machine{Address: host, Port: 8080}}, backend)
			if err := route.Start(); err != nil {
				t.Fatal(err)
			}
			defer route.Stop()
			client, err := net.Dial("tcp", publicAddress(t, route.listener.Addr()))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			client.SetDeadline(time.Now().Add(5 * time.Second))
			io.WriteString(client, "over tailscale")
			client.(*net.TCPConn).CloseWrite()
			response, err := io.ReadAll(client)
			if err != nil {
				t.Fatal(err)
			}
			if string(response) != "reply:over tailscale" {
				t.Fatalf("TCP response: %q", response)
			}
			if err := <-serverDone; err != nil {
				t.Fatal(err)
			}
			stopWithin(t, route)

			listenIP := ips[0].String()
			if ip, err := netip.ParseAddr(host); err == nil {
				listenIP = ip.String()
			}
			udp, err := target.ListenPacket("udp", net.JoinHostPort(listenIP, "8081"))
			if err != nil {
				t.Fatal(err)
			}
			defer udp.Close()
			udpRoute := NewUDPRoute(utils.RouteConfig{Type: utils.UDP, Machine: utils.Machine{Address: host, Port: 8081}}, backend)
			if err := udpRoute.Start(); err != nil {
				t.Fatal(err)
			}
			defer udpRoute.Stop()
			clients := make([]net.Conn, 2)
			peers := make([]net.Addr, 2)
			for i := range clients {
				clients[i], err = net.Dial("udp", publicAddress(t, udpRoute.listener.LocalAddr()))
				if err != nil {
					t.Fatal(err)
				}
				defer clients[i].Close()
				message := fmt.Sprintf("client-%d", i)
				io.WriteString(clients[i], message)
				udp.SetReadDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, 64)
				n, addr, err := udp.ReadFrom(buf)
				if err != nil {
					t.Fatal(err)
				}
				if string(buf[:n]) != message {
					t.Fatalf("UDP payload: %q", buf[:n])
				}
				peers[i] = addr
			}
			if peers[0].String() == peers[1].String() {
				t.Fatal("Tailscale UDP clients share a source port")
			}
			for i := len(clients) - 1; i >= 0; i-- {
				message := fmt.Sprintf("reply-%d", i)
				if _, err := udp.WriteTo([]byte(message), peers[i]); err != nil {
					t.Fatal(err)
				}
				clients[i].SetReadDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, 64)
				n, err := clients[i].Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				if string(buf[:n]) != message {
					t.Fatalf("UDP reply: %q", buf[:n])
				}
			}
			// Verify datagram boundaries and empty payloads through tsnet itself.
			for _, payload := range [][]byte{{}, bytes.Repeat([]byte{0xa5}, 1200)} {
				if _, err := clients[0].Write(payload); err != nil {
					t.Fatal(err)
				}
				udp.SetReadDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, udpBufferSize)
				n, from, err := udp.ReadFrom(buf)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(buf[:n], payload) || from.String() != peers[0].String() {
					t.Fatal("Tailscale datagram/session changed")
				}
				if _, err := udp.WriteTo(buf[:n], from); err != nil {
					t.Fatal(err)
				}
				clients[0].SetReadDeadline(time.Now().Add(5 * time.Second))
				n, err = clients[0].Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(buf[:n], payload) {
					t.Fatal("Tailscale reply datagram changed")
				}
			}
			stopWithin(t, udpRoute)
		})
	}
}
