package http

import (
	stdhttp "net/http"
)

// HealthHandler reports basic liveness for the service.
func HealthHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
