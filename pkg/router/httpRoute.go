package router

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"warptail/pkg/utils"
)

type HTTPRoute struct {
	mu          sync.RWMutex
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	config      utils.RouteConfig
	status      RouterStatus
	data        *utils.TimeSeries
	latency     time.Duration
	*http.Client
	heartbeatClient *http.Client
}

func NewHTTPRoute(config utils.RouteConfig, backend Backend) *HTTPRoute {
	client := backendHTTPClient(backend)

	// Configure optimized transport for connection pooling and keep-alive
	if transport, ok := client.Transport.(*http.Transport); ok {
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 20
		transport.IdleConnTimeout = 90 * time.Second
		transport.DisableKeepAlives = false
		transport.ForceAttemptHTTP2 = true
		transport.WriteBufferSize = 64 * 1024
		transport.ReadBufferSize = 64 * 1024
	}

	// Apply proxy settings timeout if configured
	if config.ProxySettings != nil && config.ProxySettings.Timeout > 0 {
		client.Timeout = time.Duration(config.ProxySettings.Timeout) * time.Second
	}

	// Create separate client for heartbeat to avoid affecting main traffic
	heartbeatClient := backendHTTPClient(backend)
	heartbeatClient.Timeout = 5 * time.Second

	return &HTTPRoute{
		config:          config,
		data:            utils.NewTimeSeries(time.Second, 1000),
		status:          STOPPED,
		Client:          client,
		heartbeatClient: heartbeatClient,
	}
}

func backendHTTPClient(backend Backend) *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext:           backend.Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		IdleConnTimeout:       90 * time.Second,
	}}
}

func (route *HTTPRoute) Update(config utils.RouteConfig) error {
	route.mu.Lock()
	defer route.mu.Unlock()
	route.config = config
	return nil
}
func (route *HTTPRoute) Start() error {
	route.lifecycleMu.Lock()
	defer route.lifecycleMu.Unlock()
	route.mu.Lock()
	defer route.mu.Unlock()
	if route.status == RUNNING {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	route.cancel = cancel
	route.status = RUNNING
	route.wg.Add(1)
	go route.heartbeat(ctx)
	return nil
}
func (route *HTTPRoute) Stop() error {
	route.lifecycleMu.Lock()
	defer route.lifecycleMu.Unlock()
	route.mu.Lock()
	route.status = STOPPED
	if route.cancel != nil {
		route.cancel()
	}
	route.mu.Unlock()
	route.wg.Wait()
	route.CloseIdleConnections()
	route.heartbeatClient.CloseIdleConnections()
	return nil
}
func (route *HTTPRoute) Status() RouterStatus {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.status
}
func (route *HTTPRoute) Config() utils.RouteConfig {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.config
}
func (route *HTTPRoute) Stats() utils.TimeSeriesData { return route.data.Snapshot() }

func (route *HTTPRoute) getUrl() (*url.URL, error) {
	return url.Parse("http://" + machineAddress(route.Config().Machine))
}

func getTargetUrl(config utils.RouteConfig, requestPath string) (*url.URL, string, bool) {
	// Check for path-based routing rules
	if config.ProxySettings != nil && len(config.ProxySettings.Rules) > 0 {
		for _, rule := range config.ProxySettings.Rules {
			if strings.HasPrefix(requestPath, rule.Path) {
				targetHost := rule.TargetHost
				targetPort := rule.TargetPort

				// Use default machine if not specified in rule
				if targetHost == "" {
					targetHost = config.Machine.Address
				}
				if targetPort == 0 {
					targetPort = int(config.Machine.Port)
				}

				targetUrl, err := url.Parse("http://" + net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
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
	defaultUrl, _ := url.Parse("http://" + machineAddress(config.Machine))
	return defaultUrl, requestPath, false
}

func (route *HTTPRoute) Handle(w http.ResponseWriter, r *http.Request) {
	if route.Status() != RUNNING {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	config := route.Config()
	if config.ProxySettings != nil && config.ProxySettings.Timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.ProxySettings.Timeout)*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
	}
	targetUrl, rewritePath, _ := getTargetUrl(config, r.URL.Path)
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
		if config.ProxySettings != nil {
			// Preserve or modify host header
			if config.ProxySettings.PreserveHost {
				req.Host = r.Host
			} else {
				req.Host = targetUrl.Host
			}

			// Apply custom headers
			if config.ProxySettings.CustomHeaders != nil {
				headers := config.ProxySettings.CustomHeaders

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
		if config.ProxySettings != nil && config.ProxySettings.CustomHeaders != nil {
			headers := config.ProxySettings.CustomHeaders

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

func (route *HTTPRoute) heartbeat(ctx context.Context) {
	defer route.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			target, err := route.getUrl()
			latency := time.Duration(-1)
			if err == nil {
				req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
				if reqErr == nil {
					resp, reqErr := route.heartbeatClient.Do(req)
					if reqErr == nil {
						resp.Body.Close()
						latency = time.Since(start)
					}
				}
			}
			route.mu.Lock()
			route.latency = latency
			route.mu.Unlock()
		}
	}
}
func (route *HTTPRoute) Ping() time.Duration {
	route.mu.RLock()
	defer route.mu.RUnlock()
	return route.latency
}
