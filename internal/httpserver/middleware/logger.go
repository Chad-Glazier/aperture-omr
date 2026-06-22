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
		wrappedWriter := &loggedResponseWriter{ resp: w }
		next.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.plainTextContent != "" {
			slog.Info(
				"response sent",
				"endpoint", r.Method+" "+r.URL.Path,
				"milliseconds", time.Since(start).Milliseconds(),
				"status", wrappedWriter.statusCode,
				"text content", wrappedWriter.plainTextContent,
			)
		} else {
			slog.Info(
				"response sent",
				"endpoint", r.Method+" "+r.URL.Path,
				"milliseconds", time.Since(start).Milliseconds(),
				"status", wrappedWriter.statusCode,
			)			
		}
	})
}

//
// Below, we implement a wrapper for the response writer. It delegates all
// method calls to the true response writer, but also collects the information
// about the response relevant for logging.
//

type loggedResponseWriter struct {
	resp http.ResponseWriter
	statusCode int
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
	if l.resp.Header().Get("Content-Type") == "text/plain" {
		l.plainTextContent = string(data[:min(len(data), 20)])
	}
	return l.resp.Write(data)
}
