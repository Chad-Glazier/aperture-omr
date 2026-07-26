package middleware

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

var omittedEndpoints = []string{
	"GET /system/utilization",
}

// Creates middleware that logs each request and response.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if slices.Contains(omittedEndpoints, r.Method+" "+r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// We also skip any simple file endpoints (i.e., from the static pages
		// files). We check these by just looking for a file extension.
		if strings.Contains(r.URL.Path, ".") {
			next.ServeHTTP(w, r)
			return
		}


		slog.Info("incoming", "endpoint", r.Method+" "+r.URL.Path)
		start := time.Now()
		wrappedWriter := &loggedResponseWriter{
			resp:       w,
			statusCode: http.StatusOK,
		}
		next.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.plainTextContent != "" {
			slog.Info(
				"outgoing",
				"endpoint", r.Method+" "+r.URL.Path,
				"status", wrappedWriter.statusCode,
				"time_ms", time.Since(start).Milliseconds(),
				"text_content", wrappedWriter.plainTextContent,
			)
		} else {
			slog.Info(
				"outgoing",
				"endpoint", r.Method+" "+r.URL.Path,
				"status", wrappedWriter.statusCode,
				"time_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}

//
// Below, we implement a wrapper for the response writer. It delegates all
// method calls to the true response writer while also collecting information
// relevant for logging.
//

type loggedResponseWriter struct {
	resp             http.ResponseWriter
	statusCode       int
	plainTextContent string
}

func (l *loggedResponseWriter) Header() http.Header {
	return l.resp.Header()
}

func (l *loggedResponseWriter) WriteHeader(statusCode int) {
	l.statusCode = statusCode
	l.resp.WriteHeader(statusCode)
}

func (l *loggedResponseWriter) Write(data []byte) (int, error) {
	if strings.HasPrefix(l.resp.Header().Get("Content-Type"), "text/plain") {
		l.plainTextContent = truncateContent(data, 20)
	}
	return l.resp.Write(data)
}

func truncateContent(text []byte, length int) string {
	if len(text) <= length {
		return string(text)
	}
	return string(text[:length-3]) + "..."
}
