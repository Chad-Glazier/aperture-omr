package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Creates middleware that logs each request and response.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("received request", "endpoint", r.Method+" "+r.URL.Path)
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(
			"response sent",
			"endpoint", r.Method+" "+r.URL.Path,
			"milliseconds", time.Since(start).Milliseconds(),
		)
	})
}
