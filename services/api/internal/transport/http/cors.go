package http

import (
	"net/http"
	"strings"
)

// CORS adds basic CORS headers for a configured allow-list.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowedOrigin := allowAll
		if !allowAll {
			_, allowedOrigin = allowed[origin]
		}
		if !allowedOrigin {
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				writeError(w, http.StatusForbidden, codeForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
