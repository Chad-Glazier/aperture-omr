package httpserver

import (
	"log/slog"
	"net/http"
	"os"

	"ubco-team15/omr/config"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"
)

// Starts the HTTP server and shuts it down cleanly on SIGINT or SIGTERM.
func Start() {
	if err := database.CheckConnection(); err != nil {
		slog.Error("error connecting to database", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	httpHandler := middleware.Cors(mux)
	httpHandler = middleware.Logger(httpHandler)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: httpHandler,
	}

	slog.Info("starting server at http://" + config.Host + ":" + config.Port)
	server.ListenAndServe()
}
