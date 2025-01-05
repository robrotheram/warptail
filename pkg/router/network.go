package router

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/client/tailscale"
)

type NetworkRoute struct {
	config   utils.RouteConfig
	status   RouterStatus
	client   *tailscale.LocalClient
	data     *utils.TimeSeries
	listener *net.TCPListener
	quit     chan bool
	exited   chan bool

	latency  time.Duration
	heatbeat time.Ticker
}

func NewNetworkRoute(config utils.RouteConfig, client *tailscale.LocalClient) *NetworkRoute {
	return &NetworkRoute{
		config: config,
		data:   utils.NewTimeSeries(time.Second, 1000),
		status: STOPPED,
		client: client,
	}
}

func (route *NetworkRoute) Status() RouterStatus {
	return route.status
}

func (route *NetworkRoute) Config() utils.RouteConfig {
	return route.config
}

func (route *NetworkRoute) Stats() utils.TimeSeriesData {
	return route.data.Data
}

func (route *NetworkRoute) Update(config utils.RouteConfig) error {
	route.Stop()
	route.config = config
	return route.Start()
}

func (route *NetworkRoute) Stop() error {
	if route.status != RUNNING {
		return fmt.Errorf("route not running")
	}
	route.status = STOPPING
	close(route.quit)
	<-route.exited
	utils.Logger.Info("Stopped successfully")
	route.status = STOPPED
	return nil
}

func (route *NetworkRoute) Start() error {
	if route.status == RUNNING {
		route.Stop()
	}
	go route.heartbeat(5 * time.Second)
	route.status = STARTING
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", route.config.Port))
	if err != nil {
		return err
	}
	route.quit = make(chan bool)
	route.exited = make(chan bool)

	route.listener, err = net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}

	go route.serve()
	route.status = RUNNING
	return nil
}

func (route *NetworkRoute) serve() {
	var handlers sync.WaitGroup
	for {
		select {
		case <-route.quit:
			utils.Logger.Info("Shutting down...")
			route.listener.Close()
			handlers.Wait()
			close(route.exited)
			return
		default:
			route.listener.SetDeadline(time.Now().Add(10 * time.Second))
			conn, err := route.listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				utils.Logger.Error(err, "failed to accept connection")
			}
			handlers.Add(1)
			go func() {
				route.handleConnection(conn)
				handlers.Done()
			}()
		}
	}
}

func (route *NetworkRoute) handleConnection(conn net.Conn) {
	proxy, err := route.client.UserDial(context.Background(), string(route.config.Type), route.config.Machine.Address, route.config.Machine.Port)
	if err != nil {
		utils.Logger.Error(err, "remote connection failed")
		return
	}

	sendWriter := &ConnMonitor{rw: proxy}
	reciveWriter := &ConnMonitor{rw: conn}

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go route.copy(reciveWriter, sendWriter, wg)
	go route.copy(sendWriter, reciveWriter, wg)
	go route.monitor(sendWriter, reciveWriter, wg)
	wg.Wait()
}
func (route *NetworkRoute) monitor(to, from *ConnMonitor, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		select {
		case <-route.quit:
			to.Close()
			from.Close()
			return
		default:
			route.data.LogRecived(uint64(to.bytesRead))
			route.data.LogSent(uint64(to.bytesWritten))
		}
	}
}

func (route *NetworkRoute) copy(from, to io.ReadWriter, wg *sync.WaitGroup) {
	defer wg.Done()
	select {
	case <-route.quit:
		return
	default:
		if _, err := io.Copy(to, from); err != nil {
			return
		}
	}
}

func (route *NetworkRoute) heartbeat(timeout time.Duration) {
	route.heatbeat = *time.NewTicker(timeout)
	go func() {
		for range route.heatbeat.C {
			if route.status != RUNNING {
				route.latency = -1
				route.heatbeat.Stop()
				continue
			}
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*1))
			conn, err := route.client.UserDial(ctx, string(route.config.Type), route.config.Machine.Address, route.config.Machine.Port)
			cancel()
			if err != nil {
				route.latency = -1
				continue
			}
			conn.Close()
			route.latency = time.Since(start) / time.Millisecond
		}
	}()
}

func (route *NetworkRoute) Ping() time.Duration {
	return route.latency
}
