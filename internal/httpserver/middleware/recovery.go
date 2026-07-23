package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"os"
)

const logFilePath = "debug.log"
var logFileMu sync.Mutex
var logFile io.WriteCloser

func init() {
	f, err := os.Create(logFilePath)
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

				const maxStackTraceLen = 10000
				buf := make([]byte, maxStackTraceLen)
				nBytes := runtime.Stack(buf, false)

				trace := string(buf)
				trace = strings.Join(strings.Split(trace, "\n"), "\n│  ")
				trace = "│  " + trace
				buf = []byte(trace)

				fmt.Fprint(
					logFile,
					"\n┌── Panic Log ────────────────────────────────────────────────────────────────────────────────────────────────\n│\n",
				)

				fmt.Fprintf(
					logFile,
					"│   endpoint.... %s\n" +
					"│   time........ %s\n│\n",
					r.Method+" "+r.URL.Path,
					formatDate(time.Now()),
				)

				logFile.Write(buf[:nBytes])
				if len(buf) > maxStackTraceLen {
					fmt.Fprint(logFile, "\n│  [...]\n│")
				}

				fmt.Fprint(
					logFile,
					"\n│\n└─────────────────────────────────────────────────────────────────────────────────────────────────────────────\n",
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

func formatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}
