package main

//
// This entrypoint starts an HTTP server that exposes the RestAPI documented in
// `/api/openapi.yaml`.
//

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"ubc/team15/config"
	"ubc/team15/internal/db"
	"ubc/team15/internal/http/handler"
	"ubc/team15/internal/http/middleware"
)

func main() {

	//
	// First, we ensure that we can connect to the database.
	//

	_, err := db.Connect()
	if err != nil {
		slog.Error("error connecting to database", "error", err)
		os.Exit(1)
	}

	//
	// Register the handler functions to a multiplexer.
	//

	mux := http.NewServeMux()

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	//
	// Register global middleware.
	//

	handler := middleware.Cors(mux)

	//
	// Configure the server.
	//

	server := &http.Server{
		Addr:    ":" + config.PORT,
		Handler: handler,
	}

	//
	// Start the server.
	//

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("server started at http://" + config.HOST + ":" + config.PORT)

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("server failed to start")
	}

}
