package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func gatherMetrics(t *testing.T, handler http.Handler) string {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", rec.Code)
	}
	return rec.Body.String()
}

func TestMetrics_RecordsRequestCount(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics(prometheus.NewRegistry())
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := gatherMetrics(t, metrics.Handler())
	expected := metricRequestCount + `{method="POST",route="/auth/login",status="401"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected metrics to include %q, got %s", expected, body)
	}
}

func TestMetrics_NormalizesRoutes(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics(prometheus.NewRegistry())
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/holds/hold-123/confirm", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := gatherMetrics(t, metrics.Handler())
	expected := metricRequestCount + `{method="POST",route="/holds/{hold_id}/confirm",status="200"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected normalized route metric %q, got %s", expected, body)
	}
}
