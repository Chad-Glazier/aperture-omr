package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		Addr:              ":" + config.PORT,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverError := make(chan error, 1)
	go func() {
		slog.Info("starting server at http://" + config.HOST + ":" + config.PORT)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
		}
		return
	case <-shutdownSignal.Done():
		slog.Info("shutting down server")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error("server shutdown failed", "error", err)
		return
	}

	slog.Info("server stopped")
}
