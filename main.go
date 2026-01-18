package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

const (
	publicListenAddr = ":5520"               // Public UDP port
	gameServerAddr   = "100.118.227.76:5520" // Tailnet game server address
	sessionTimeout   = 30 * time.Second
)

type session struct {
	clientAddr net.Addr
	lastSeen   time.Time
}

func main() {
	// Tailscale userspace node
	ts := &tsnet.Server{
		Hostname: "udp-forwarder",
		// Uncomment and set if using auth key instead of interactive login:
		// AuthKey: os.Getenv("TS_AUTHKEY"),
	}
	defer ts.Close()

	// Start Tailscale and wait for it to be ready
	log.Println("Starting Tailscale...")
	status, err := ts.Up(context.Background())
	if err != nil {
		log.Fatal("Failed to start Tailscale:", err)
	}
	if len(status.TailscaleIPs) == 0 {
		log.Fatal("No Tailscale IPs assigned")
	}
	tsIP := status.TailscaleIPs[0] // Use first IP (IPv4)
	log.Printf("Tailscale up! IP: %v", tsIP)

	// Public UDP socket
	pubConn, err := net.ListenPacket("udp", publicListenAddr)
	if err != nil {
		log.Fatal("Failed to listen on public port:", err)
	}
	defer pubConn.Close()

	// Tailnet UDP socket - bind to our Tailscale IP
	tsConn, err := ts.ListenPacket("udp", net.JoinHostPort(tsIP.String(), "0"))
	if err != nil {
		log.Fatal("Failed to create Tailscale UDP socket:", err)
	}
	defer tsConn.Close()

	gameAddr, err := net.ResolveUDPAddr("udp", gameServerAddr)
	if err != nil {
		log.Fatal("Failed to resolve game server address:", err)
	}

	log.Printf("UDP forwarder running: %s -> tailscale -> %s", publicListenAddr, gameServerAddr)

	var sessions sync.Map

	// Cleanup stale sessions periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sessions.Range(func(key, value any) bool {
				s := value.(*session)
				if time.Since(s.lastSeen) > sessionTimeout {
					sessions.Delete(key)
					log.Printf("Session expired: %s", key)
				}
				return true
			})
		}
	}()

	// Read from public → tailnet
	go func() {
		buf := make([]byte, 65535) // Separate buffer, max UDP size
		for {
			n, clientAddr, err := pubConn.ReadFrom(buf)
			if err != nil {
				log.Println("Public read error:", err)
				continue
			}

			sessions.Store(clientAddr.String(), &session{
				clientAddr: clientAddr,
				lastSeen:   time.Now(),
			})

			_, err = tsConn.WriteTo(buf[:n], gameAddr)
			if err != nil {
				log.Println("Tailscale write error:", err)
			}
		}
	}()

	// Read from tailnet → public
	buf := make([]byte, 65535) // Separate buffer for this direction
	for {
		n, _, err := tsConn.ReadFrom(buf)
		if err != nil {
			log.Println("Tailscale read error:", err)
			continue
		}

		// Copy data for safe concurrent access
		data := make([]byte, n)
		copy(data, buf[:n])

		sessions.Range(func(_, v any) bool {
			s := v.(*session)

			if time.Since(s.lastSeen) > sessionTimeout {
				return true
			}

			_, err := pubConn.WriteTo(data, s.clientAddr)
			if err != nil {
				log.Printf("Public write error to %s: %v", s.clientAddr, err)
			}
			return true
		})
	}
}
