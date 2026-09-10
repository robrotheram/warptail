package router

import (
	"context"
	"sync"
	"time"
	"warptail/pkg/utils"
)

const backendDialTimeout = 10 * time.Second

// lifecycleMu serializes Start, Stop and Update through worker shutdown.
type connectionRoute struct {
	lifecycleMu sync.Mutex
	mu          sync.RWMutex
	config      utils.RouteConfig
	backend     Backend
	data        *utils.TimeSeries
	status      RouterStatus
	latency     time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func newConnectionRoute(config utils.RouteConfig, backend Backend) *connectionRoute {
	return &connectionRoute{config: config, backend: backend,
		data: utils.NewTimeSeries(time.Second, 1000), status: STOPPED, latency: -1}
}
func (r *connectionRoute) Config() utils.RouteConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}
func (r *connectionRoute) Status() RouterStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}
func (r *connectionRoute) Stats() utils.TimeSeriesData { return r.data.Snapshot() }
func (r *connectionRoute) Ping() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.latency
}
func (r *connectionRoute) runHeartbeat(ctx context.Context, probe func(context.Context) (time.Duration, error)) {
	defer r.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			latency, err := probe(probeCtx)
			cancel()
			if err != nil {
				latency = -1
			}
			r.mu.Lock()
			r.latency = latency
			r.mu.Unlock()
		}
	}
}
