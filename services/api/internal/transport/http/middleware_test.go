package http

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeLogEntry(t *testing.T, buf *bytes.Buffer) requestLogEntry {
	t.Helper()

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatalf("expected log output")
	}
	var entry requestLogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("decode log json: %v", err)
	}
	return entry
}

func TestRequestLogger_LogsStatusAndPath(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/holds", nil)
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(handler, logger)).ServeHTTP(rec, req)

	entry := decodeLogEntry(t, buf)
	if entry.Method != http.MethodGet {
		t.Fatalf("expected method GET, got %q", entry.Method)
	}
	if entry.Path != "/holds" {
		t.Fatalf("expected path /holds, got %q", entry.Path)
	}
	if entry.Status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", entry.Status)
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

	RequestID(RequestLogger(handler, logger)).ServeHTTP(rec, req)

	entry := decodeLogEntry(t, buf)
	if entry.Status != http.StatusOK {
		t.Fatalf("expected default status 200 in log, got %d", entry.Status)
	}
}

func TestRequestID_GeneratesAndPropagates(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Fatalf("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	RequestID(handler).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got == "" {
		t.Fatalf("expected X-Request-Id header to be set")
	}
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id != "req-123" {
			t.Fatalf("expected request id req-123, got %q", id)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-Id", "req-123")
	rec := httptest.NewRecorder()

	RequestID(handler).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "req-123" {
		t.Fatalf("expected X-Request-Id header to be req-123, got %q", got)
	}
}

func TestRequestLogger_LogsRequestIDAndBytes(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-Id", "req-456")
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(handler, logger)).ServeHTTP(rec, req)

	entry := decodeLogEntry(t, buf)
	if entry.RequestID != "req-456" {
		t.Fatalf("expected request_id req-456, got %q", entry.RequestID)
	}
	if entry.Bytes != 5 {
		t.Fatalf("expected bytes 5, got %d", entry.Bytes)
	}
}
