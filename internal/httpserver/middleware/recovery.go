package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"sync"

	"os"
)

var logFileMu sync.Mutex
var logFile io.Writer

func init() {
	f, err := os.Create("debug.log")
	if err != nil {
		panic("failed to create debug.log file")
	}

	logFile = f
}

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
					"stack trace written to", "debug.log",
				)

				logFileMu.Lock()
				defer logFileMu.Unlock()

				buf := make([]byte, 10000)
				nBytes := runtime.Stack(buf, false)
				fmt.Fprintf(
					logFile,
					"\n\n--- Panic Recovered in %s  ---\n",
					r.Method+" "+r.URL.Path,
				)
				logFile.Write(buf[:nBytes])
				fmt.Fprint(
					logFile,
					"\n--- End of Stack Trace ---\n\n",
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
