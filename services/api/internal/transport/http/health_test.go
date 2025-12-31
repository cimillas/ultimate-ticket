package http

import (
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_OK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	res := rec.Result()
	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	body := rec.Body.String()
	if body != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", body)
	}
}
