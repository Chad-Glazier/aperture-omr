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

// Starts the HTTP server.
func Start() {

	err := database.CheckConnection()
	if err != nil {
		slog.Error("error connecting to database", "error", err)
		os.Exit(1)
	}

	//
	// Register the handlers.
	//

	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	//
	// Register global middleware.
	//

	handler := middleware.Cors(mux)
	handler = middleware.Logger(handler)

	//
	// Configure the server.
	//

	server := &http.Server{
		Addr:    ":" + config.PORT,
		Handler: handler,
	}

	slog.Info("starting server at http://" + config.HOST + ":" + config.PORT)

	err = server.ListenAndServe()
	if err != nil {
		slog.Error("server failed to start")
	}

}
