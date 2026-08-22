package mw

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

// Makes it so that handlers will gracefully recover from panics and send back
// an error message.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer RecoverAndRespond(w, r)
		next.ServeHTTP(w, r)
	})
}

// Invokes [recover] and, if there was a panic recovered, sends back a 500
// response and logs the panic.
func RecoverAndRespond(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		sys.Error(
			"panic recovered",
			"endpoint", r.Method+" "+r.URL.Path,
			"err", err,
		)

		out := strings.Builder{}

		fmt.Fprint(&out,
			"\n┌── Panic Log ────────────────────────────────────────────────────────────────────────────────────────────────\n│\n",
		)

		fmt.Fprintf(&out,
			"│   endpoint..... %s\n"+
				"│   time......... %s\n"+
				"│   recovered.... %v\n│\n",
			r.Method+" "+r.URL.Path,
			time.Now().Format("2006-01-02 15:04:05.000"),
			err,
		)

		buf := make([]byte, 4<<10)
		n := runtime.Stack(buf, false)
		trace := string(buf[:n])
		trace = strings.Join(strings.Split(trace, "\n"), "\n│  ")
		trace = "│  " + trace

		out.WriteString(trace)

		if n == len(buf) {
			fmt.Fprint(&out, "\n│  [...]\n│")
		}

		fmt.Fprint(&out,
			"\n└─────────────────────────────────────────────────────────────────────────────────────────────────────────────\n\n",
		)

		fmt.Fprint(os.Stderr, out.String())

		// Only write a response if one hasn't already been written.
		if w.Header().Get("Content-Type") == "" {
			http.Error(w,
				"Unknown server error. Check the server logs.",
				http.StatusInternalServerError,
			)
		}
	}
}
