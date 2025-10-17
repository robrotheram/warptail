package router

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	"warptail/pkg/utils"

	"tailscale.com/tsnet"
)

type HTTPRoute struct {
	config   utils.RouteConfig
	status   RouterStatus
	data     *utils.TimeSeries
	latency  time.Duration
	heatbeat *time.Ticker
	*http.Client
}

func NewHTTPRoute(config utils.RouteConfig, server *tsnet.Server) *HTTPRoute {
	client := server.HTTPClient()

	// Apply proxy settings timeout if configured
	if config.ProxySettings != nil && config.ProxySettings.Timeout > 0 {
		client.Timeout = time.Duration(config.ProxySettings.Timeout) * time.Second
	}

	return &HTTPRoute{
		config: config,
		data:   utils.NewTimeSeries(time.Second, 1000),
		status: STOPPED,
		Client: client,
	}
}

func (route *HTTPRoute) Update(config utils.RouteConfig) error {
	route.config = config

	// Update client timeout if proxy settings changed
	if config.ProxySettings != nil && config.ProxySettings.Timeout > 0 {
		route.Client.Timeout = time.Duration(config.ProxySettings.Timeout) * time.Second
	} else {
		// Reset to default timeout
		route.Client.Timeout = 30 * time.Second
	}

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

func (route *HTTPRoute) getUrl() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%d", route.config.Machine.Address, route.config.Machine.Port))
}

func (route *HTTPRoute) getTargetUrl(requestPath string) (*url.URL, string, bool) {
	// Check for path-based routing rules
	if route.config.ProxySettings != nil && len(route.config.ProxySettings.Rules) > 0 {
		for _, rule := range route.config.ProxySettings.Rules {
			if strings.HasPrefix(requestPath, rule.Path) {
				targetHost := rule.TargetHost
				targetPort := rule.TargetPort

				// Use default machine if not specified in rule
				if targetHost == "" {
					targetHost = route.config.Machine.Address
				}
				if targetPort == 0 {
					targetPort = int(route.config.Machine.Port)
				}

				targetUrl, err := url.Parse(fmt.Sprintf("http://%s:%d", targetHost, targetPort))
				if err != nil {
					continue
				}

				// Calculate rewritten path
				rewritePath := requestPath
				if rule.StripPath {
					rewritePath = strings.TrimPrefix(requestPath, rule.Path)
					if !strings.HasPrefix(rewritePath, "/") && rewritePath != "" {
						rewritePath = "/" + rewritePath
					}
				}
				if rule.Rewrite != "" {
					rewritePath = rule.Rewrite + strings.TrimPrefix(rewritePath, "/")
				}

				return targetUrl, rewritePath, true
			}
		}
	}

	// Default to original machine
	defaultUrl, _ := route.getUrl()
	return defaultUrl, requestPath, false
}

func (route *HTTPRoute) Handle(w http.ResponseWriter, r *http.Request) {
	if route.status != RUNNING {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	targetUrl, rewritePath, _ := route.getTargetUrl(r.URL.Path)
	if targetUrl == nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	if bodyBytes, err := io.ReadAll(r.Body); err == nil {
		route.data.LogSent(uint64(len(bodyBytes)))
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Transport = route.Transport

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Apply path rewriting
		req.URL.Path = rewritePath

		// Handle proxy settings
		if route.config.ProxySettings != nil {
			// Preserve or modify host header
			if route.config.ProxySettings.PreserveHost {
				req.Host = r.Host
			} else {
				req.Host = targetUrl.Host
			}

			// Apply custom headers
			if route.config.ProxySettings.CustomHeaders != nil {
				headers := route.config.ProxySettings.CustomHeaders

				// Remove headers
				for _, headerName := range headers.Remove {
					req.Header.Del(headerName)
				}

				// Add headers (don't overwrite existing)
				for key, value := range headers.Add {
					if req.Header.Get(key) == "" {
						req.Header.Set(key, value)
					}
				}

				// Set headers (overwrite existing)
				for key, value := range headers.Set {
					req.Header.Set(key, value)
				}
			}
		} else {
			// Default behavior - preserve original headers
			originalHost := r.Host
			req.Host = originalHost
		}

		// Ensure proper session and cookie handling
		if cookies := req.Header.Get("Cookie"); cookies != "" {
			req.Header.Set("Cookie", cookies)
		}

		// Handle WebSocket upgrades
		if strings.HasPrefix(req.Header.Get("Connection"), "Upgrade") {
			req.Header.Set("Connection", "Upgrade")
		}
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Apply response header modifications if configured
		if route.config.ProxySettings != nil && route.config.ProxySettings.CustomHeaders != nil {
			headers := route.config.ProxySettings.CustomHeaders

			// Remove response headers
			for _, headerName := range headers.Remove {
				resp.Header.Del(headerName)
			}

			// Add response headers
			for key, value := range headers.Add {
				if resp.Header.Get(key) == "" {
					resp.Header.Set(key, value)
				}
			}

			// Set response headers
			for key, value := range headers.Set {
				resp.Header.Set(key, value)
			}
		}

		// Preserve session cookies and headers
		for _, cookie := range resp.Cookies() {
			resp.Header.Add("Set-Cookie", cookie.String())
		}
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// Log to error log
		if utils.RequestLogger != nil {
			utils.RequestLogger.LogError(r, fmt.Errorf("proxy error to %s: %v", targetUrl.String(), err))
		}
		http.Error(w, "Bad Gateway: Unable to reach backend service", http.StatusBadGateway)
	}

	rr := NewResponseRecorder(w)
	proxy.ServeHTTP(rr, r)
	if utils.RequestLogger != nil {
		utils.RequestLogger.LogRequest(r, time.Now(), rr.statusCode, rr.responseSize)
	}
	route.data.LogRecived(uint64(rr.responseSize))
}

func (route *HTTPRoute) heartbeat(timeout time.Duration) {
	route.heatbeat = time.NewTicker(timeout) // pointer, not value
	go func() {
		for range route.heatbeat.C {
			if route.status != RUNNING {
				route.latency = -1
				route.heatbeat.Stop()
				route.heatbeat = nil
				return // exit goroutine after stopping ticker
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
				utils.Logger.Error(err, "Error pinging server", "url", url.String())
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
