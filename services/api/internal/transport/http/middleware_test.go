package http

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogger_LogsStatusAndPath(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/holds", nil)
	rec := httptest.NewRecorder()

	RequestLogger(handler, logger).ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "method=GET") {
		t.Fatalf("expected method in log, got %q", out)
	}
	if !strings.Contains(out, "path=/holds") {
		t.Fatalf("expected path in log, got %q", out)
	}
	if !strings.Contains(out, "status=201") {
		t.Fatalf("expected status in log, got %q", out)
	}
}

func TestRequestLogger_DefaultsTo200(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	RequestLogger(handler, logger).ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "status=200") {
		t.Fatalf("expected default status 200 in log, got %q", out)
	}
}
