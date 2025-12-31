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
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != codeNotFound {
		t.Fatalf("expected code %s, got %s", codeNotFound, resp.Code)
	}
}
