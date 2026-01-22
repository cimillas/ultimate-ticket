package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricRequestCount    = "http_requests_total"
	metricRequestDuration = "http_request_duration_seconds"
	unknownRoute          = "unknown"
)

// Metrics captures request counters and latency distributions.
type Metrics struct {
	registry        *prometheus.Registry
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewMetrics creates a metrics registry and instruments request traffic.
func NewMetrics(registry *prometheus.Registry) *Metrics {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	requestCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricRequestCount,
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    metricRequestDuration,
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	registry.MustRegister(requestCount, requestDuration)

	return &Metrics{
		registry:        registry,
		requestCount:    requestCount,
		requestDuration: requestDuration,
	}
}

// Handler exposes metrics in Prometheus format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware records request metrics after each request.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route := normalizeMetricRoute(r.URL.Path)
		status := strconv.Itoa(rec.status)
		m.requestCount.WithLabelValues(r.Method, route, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

func normalizeMetricRoute(path string) string {
	if path == "" {
		return unknownRoute
	}
	clean := strings.TrimRight(path, "/")
	if clean == "" {
		clean = "/"
	}

	switch clean {
	case "/health", "/events", "/orders", "/holds", "/me", "/metrics":
		return clean
	case "/auth/register", "/auth/login", "/auth/logout", "/auth/password":
		return clean
	case "/admin/events":
		return clean
	}

	if _, ok := parseEventZonesPath(clean); ok {
		return "/events/{event_id}/zones"
	}
	if _, ok := parseConfirmHoldPath(clean); ok {
		return "/holds/{hold_id}/confirm"
	}
	if _, ok := parseAdminEventCancelPath(clean); ok {
		return "/admin/events/{event_id}/cancel"
	}
	if _, ok := parseAdminEventZonesPath(clean); ok {
		return "/admin/events/{event_id}/zones"
	}
	if _, _, resource, ok := parseAdminEventZoneResourcePath(clean); ok {
		switch resource {
		case "holds", "orders":
			return "/admin/events/{event_id}/zones/{zone_id}/" + resource
		}
	}

	return unknownRoute
}
