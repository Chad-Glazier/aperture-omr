package mw

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

// We don't log the commonly-polled endpoints.
var omittedEndpoints = []string{
	"GET /system/utilization",
	"GET /system/logs",
	"GET /jobs",
	"GET /job",
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

		sys.Log("incoming", "endpoint", r.Method+" "+r.URL.Path)
		start := time.Now()
		wrappedWriter := &loggedResponseWriter{
			resp:       w,
			statusCode: http.StatusOK,
		}
		next.ServeHTTP(wrappedWriter, r)

		if wrappedWriter.statusCode >= 500 {
			sys.Error(
				"outgoing",
				"endpoint", r.Method+" "+r.URL.Path,
				"status", wrappedWriter.statusCode,
				"time ms", time.Since(start).Milliseconds(),
			)
		} else {
			sys.Log(
				"outgoing",
				"endpoint", r.Method+" "+r.URL.Path,
				"status", wrappedWriter.statusCode,
				"time ms", time.Since(start).Milliseconds(),
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
	resp       http.ResponseWriter
	statusCode int
}

var _ http.ResponseWriter = (*loggedResponseWriter)(nil)

func (l *loggedResponseWriter) Write(p []byte) (int, error) {
	return l.resp.Write(p)
}

func (l *loggedResponseWriter) Header() http.Header {
	return l.resp.Header()
}

func (l *loggedResponseWriter) WriteHeader(statusCode int) {
	l.statusCode = statusCode
	l.resp.WriteHeader(statusCode)
}
