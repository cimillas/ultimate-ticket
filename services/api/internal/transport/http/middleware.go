package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultServiceName = "api"

// RequestLogger logs basic request details and latency.
func RequestLogger(next http.Handler, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	jsonLogger := log.New(logger.Writer(), "", 0)
	serviceName := resolveServiceName()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		entry := requestLogEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
			RequestID:  RequestIDFromContext(r.Context()),
			Service:    serviceName,
			Method:     r.Method,
			Path:       r.URL.Path,
			Status:     rec.status,
			DurationMs: time.Since(start).Milliseconds(),
			Bytes:      rec.bytes,
			RemoteIP:   clientIP(r),
			UserAgent:  r.UserAgent(),
		}
		payload, err := json.Marshal(entry)
		if err != nil {
			logger.Printf("request log marshal error: %v", err)
			return
		}
		jsonLogger.Print(string(payload))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) RequestID() string {
	if getter, ok := r.ResponseWriter.(interface{ RequestID() string }); ok {
		return getter.RequestID()
	}
	return ""
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

type requestLogEntry struct {
	Timestamp  string `json:"ts"`
	RequestID  string `json:"request_id,omitempty"`
	Service    string `json:"service"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Bytes      int    `json:"bytes"`
	RemoteIP   string `json:"remote_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

func resolveServiceName() string {
	serviceName := strings.TrimSpace(os.Getenv("SERVICE_NAME"))
	if serviceName == "" {
		return defaultServiceName
	}
	return serviceName
}

type requestIDKey struct{}

type requestIDResponseWriter struct {
	http.ResponseWriter
	requestID string
}

func (w *requestIDResponseWriter) RequestID() string {
	return w.requestID
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-Id", reqID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		rw := &requestIDResponseWriter{ResponseWriter: w, requestID: reqID}
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func generateRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
