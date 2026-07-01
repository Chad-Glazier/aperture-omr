package middleware

import (
	"net/http"
	"log/slog"
)

// Makes it so that handlers will gracefully recover from panics and send back
// an error message.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error(
					"panic recovered",
					"endpoint", r.Method+" "+r.URL.Path,
					"err", err,
				)

				// Only write a response if one hasn't already been written.
				if w.Header().Get("Content-Type") == "" {
					http.Error(
						w,
						"Unknown server error. Check the server logs.",
						http.StatusInternalServerError,
					)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
