package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotFoundHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)
	mux.Handle("/", NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("X-Request-Id", "req-123")
	rec := httptest.NewRecorder()

	RequestID(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] != codeNotFound {
		t.Fatalf("expected code %s, got %v", codeNotFound, resp["code"])
	}
	if resp["request_id"] != "req-123" {
		t.Fatalf("expected request_id req-123, got %v", resp["request_id"])
	}
}
