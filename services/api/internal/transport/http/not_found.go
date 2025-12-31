package http

import "net/http"

// NotFoundHandler returns a JSON 404 response for unknown routes.
func NotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "not found")
	})
}
