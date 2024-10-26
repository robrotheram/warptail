package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

type HTTPRoute struct {
	config   utils.RouteConfig
	status   RouterStatus
	data     *utils.TimeSeries
	latency  time.Duration
	heatbeat time.Ticker
	*http.Client
}

func NewHTTPRoute(config utils.RouteConfig, server *tsnet.Server) *HTTPRoute {
	return &HTTPRoute{
		config: config,
		data:   utils.NewTimeSeries(time.Second, 1000),
		status: STOPPED,
		Client: server.HTTPClient(),
	}
}

func (route *HTTPRoute) Update(config utils.RouteConfig) error {
	route.config = config
	return nil
}
func (route *HTTPRoute) Start() error {
	route.status = RUNNING
	go route.heartbeat(5 * time.Second)
	return nil
}
func (route *HTTPRoute) Stop() error {
	route.status = STOPPED
	return nil
}

func (route *HTTPRoute) Status() RouterStatus {
	return route.status
}

func (route *HTTPRoute) Config() utils.RouteConfig {
	return route.config
}

func (route *HTTPRoute) Stats() utils.TimeSeriesData {
	return route.data.Data
}

func parseRequestSize(header http.Header) (int64, error) {
	contentLength := header.Get("Content-Length")
	if contentLength == "" {
		return 0, nil
	}
	return strconv.ParseInt(contentLength, 10, 64)
}
func (route *HTTPRoute) getUrl() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%d", route.config.Machine.Address, route.config.Machine.Port))
}
func (route *HTTPRoute) Handle(w http.ResponseWriter, r *http.Request) {
	if route.status != RUNNING {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	url, err := route.getUrl()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = route.Transport

	if size, err := parseRequestSize(r.Header); err == nil {
		route.data.LogRecived(uint64(size))
	}

	proxy.ServeHTTP(w, r)
	if size, err := parseRequestSize(w.Header()); err == nil {
		route.data.LogSent(uint64(size))
	}
}

func (route *HTTPRoute) heartbeat(timeout time.Duration) {
	route.heatbeat = *time.NewTicker(timeout)
	go func() {
		for range route.heatbeat.C {
			if route.status != RUNNING {
				route.latency = -1
				route.heatbeat.Stop()
			}
			route.Client.Timeout = 5 * time.Second
			start := time.Now()
			url, err := route.getUrl()
			if err != nil {
				route.latency = -1 // Unable to reach the server
				continue
			}
			resp, err := route.Get(url.String())
			if err != nil {
				route.latency = -1 // Unable to reach the server
				continue
			}
			resp.Body.Close()
			route.latency = time.Since(start) / time.Millisecond
		}
	}()
}

func (route *HTTPRoute) Ping() time.Duration {
	return route.latency
}
