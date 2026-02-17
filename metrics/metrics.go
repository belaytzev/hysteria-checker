package metrics

import (
	"fmt"
	"net/http"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/middleware"
	"github.com/belaytzev/hysteria-checker/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	proxyStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hysteria_proxy_status",
			Help: "Proxy status: 1=up, 0=down",
		},
		[]string{"version", "address", "name", "id"},
	)

	proxyLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hysteria_proxy_latency_ms",
			Help: "Proxy latency in milliseconds, 0 if down",
		},
		[]string{"version", "address", "name", "id"},
	)
)

func init() {
	prometheus.MustRegister(proxyStatus)
	prometheus.MustRegister(proxyLatency)
}

// UpdateMetrics updates Prometheus gauges based on check results and proxy configs.
func UpdateMetrics(results map[string]checker.CheckResult, proxies []models.ProxyConfig) {
	// Reset to remove stale metrics for proxies that no longer exist
	proxyStatus.Reset()
	proxyLatency.Reset()

	for _, p := range proxies {
		labels := prometheus.Labels{
			"version": fmt.Sprintf("hy%d", p.Version),
			"address": p.Server,
			"name":    p.Name,
			"id":      p.StableID,
		}

		result, ok := results[p.StableID]
		if !ok {
			proxyStatus.With(labels).Set(0)
			proxyLatency.With(labels).Set(0)
			continue
		}

		if result.Alive {
			proxyStatus.With(labels).Set(1)
			proxyLatency.With(labels).Set(float64(result.Latency.Milliseconds()))
		} else {
			proxyStatus.With(labels).Set(0)
			proxyLatency.With(labels).Set(0)
		}
	}
}

// Handler returns an http.Handler that serves Prometheus metrics.
// If protected is true, it wraps the handler with Basic Auth.
func Handler(protected bool, username, password string) http.Handler {
	h := promhttp.Handler()
	if !protected {
		return h
	}
	return middleware.BasicAuth(h, username, password)
}
