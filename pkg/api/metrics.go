package api

import (
	"fmt"
	"strconv"
	"time"
	"warptail/pkg/router"
	"warptail/pkg/utils"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics struct for Prometheus
type ServiceMetrics struct {
	ServiceEnabled *prometheus.GaugeVec
	ServiceLatency *prometheus.GaugeVec
	TotalSent      *prometheus.GaugeVec
	TotalReceived  *prometheus.GaugeVec

	RouteStatus  *prometheus.GaugeVec
	RouteLatency *prometheus.GaugeVec
}

// CreateMetrics initializes and registers Prometheus metrics for the service
func CreateMetrics() *ServiceMetrics {
	return &ServiceMetrics{
		ServiceEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_service_enabled",
				Help: "Indicates if the warptail service is enabled",
			},
			[]string{"service_name"},
		),
		ServiceLatency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_service_latency",
				Help: "Latency of the warptail service",
			},
			[]string{"service_name"},
		),
		TotalSent: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_service_total_sent",
				Help: "Total data sent by the warptail service",
			},
			[]string{"service_name"},
		),
		TotalReceived: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_service_total_received",
				Help: "Total data received by the warptail service",
			},
			[]string{"service_name"},
		),
		RouteLatency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_service_route_latency",
				Help: "Latency of warptail route",
			},
			[]string{"service_name", "route_type", "route_entrypoint", "tailscale_address"},
		),
		RouteStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "warptail_route_status",
				Help: "Status of each route in the warptail service",
			},
			[]string{"service_name", "route_type", "route_entrypoint", "tailscale_address"},
		),
	}
}

func (metrics *ServiceMetrics) Register() {
	prometheus.MustRegister(metrics.ServiceEnabled)
	prometheus.MustRegister(metrics.ServiceLatency)
	prometheus.MustRegister(metrics.RouteLatency)
	prometheus.MustRegister(metrics.RouteStatus)
	prometheus.MustRegister(metrics.TotalSent)
	prometheus.MustRegister(metrics.TotalReceived)
}

// UpdateMetrics updates the Prometheus metrics with data from the Service struct
func (metrics *ServiceMetrics) Update(servics []router.Service) {
	for _, svc := range servics {
		service := svc.Status(true)
		enabled := 0.0
		if service.Enabled {
			enabled = 1.0
		}

		metrics.ServiceEnabled.WithLabelValues(service.Name).Set(enabled)
		metrics.ServiceLatency.WithLabelValues(service.Name).Set(float64(service.Latency))
		metrics.TotalSent.WithLabelValues(service.Name).Set(float64(service.Stats.Total.Sent))
		metrics.TotalReceived.WithLabelValues(service.Name).Set(float64(service.Stats.Total.Received))

		for _, route := range service.Routes {
			statusValue := 0.0
			if route.Status == "Running" {
				statusValue = 1.0
			}
			var label = []string{}
			switch route.Type {
			case utils.HTTP, utils.HTTPS:
				label = []string{
					service.Name,
					string(route.Type),
					route.Domain,
					fmt.Sprintf("%s:%d", route.Machine.Address, route.Machine.Port),
				}
			case utils.TCP, utils.UDP:
				label = []string{
					service.Name,
					string(route.Type),
					strconv.Itoa(route.Port),
					fmt.Sprintf("%s:%d", route.Machine.Address, route.Machine.Port),
				}
			}
			metrics.RouteStatus.WithLabelValues(label...).Set(statusValue)
			metrics.RouteLatency.WithLabelValues(label...).Set(float64(route.Latency))
		}
	}
}

func (api *api) startMetrics() {
	metrics := CreateMetrics()
	metrics.Register()
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		metrics.Update(api.All())
	}
}
