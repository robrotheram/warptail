package router

import (
	"context"
	"net"
	"strconv"
	"time"
	"warptail/pkg/utils"
)

// Backend opens outbound connections. UDP connections must preserve datagram
// boundaries and use a distinct source port per Dial. Implementations must honor
// dial cancellation and connection deadlines. TCP should implement CloseWrite.
type Backend interface {
	Dial(context.Context, string, string) (net.Conn, error)
}

// BackendPinger is optional: opening a UDP socket does not prove reachability.
type BackendPinger interface {
	Ping(context.Context, string) (time.Duration, error)
}

// SystemBackend uses host routing, including externally managed WireGuard or
// Nebula interfaces. Tunnel setup and DNS remain the host's responsibility.
type SystemBackend struct{ Dialer net.Dialer }

func (b *SystemBackend) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return b.Dialer.DialContext(ctx, network, address)
}

func machineAddress(machine utils.Machine) string {
	return net.JoinHostPort(machine.Address, strconv.Itoa(int(machine.Port)))
}
