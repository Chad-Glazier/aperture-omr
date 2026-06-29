package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"ubco-team15/omr/config"
	"ubco-team15/omr/internal/database"
	"ubco-team15/omr/internal/database/sqlc"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"

	"github.com/google/uuid"
)

// Starts the HTTP server and shuts it down cleanly on SIGINT or SIGTERM.
func Start() {

	res, err := NewServerResources()
	if err != nil {
		slog.Error("error getting server resources", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	// mux.HandleFunc("POST /upload", handler.PostUpload)
	// mux.HandleFunc("GET /upload", handler.GetUpload)
	// mux.HandleFunc("DELETE /upload", handler.DeleteUpload)

	mux.HandleFunc("POST /template/mark", handler.PostMarkingTemplate(res))

	httpHandler := middleware.Cors(mux)
	httpHandler = middleware.Logger(httpHandler)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: httpHandler,
	}

	slog.Info("starting server at http://" + config.Host + ":" + config.Port)
	server.ListenAndServe()
}

//
// Below, we implement the ServerResources interface. This is how the database
// and file storage is hooked up to the endpoints.
//

type ServerResources struct {
	db database.Querier
}

var _ handler.ServerResources = &ServerResources{}

func NewServerResources() (*ServerResources, error) {
	db, err := database.Connect()
	if err != nil {
		return nil, err
	}

	res := &ServerResources{
		db: db,
	}
	return res, nil
}

func (s *ServerResources) SaveMarkingTemplate(
	tmpl *dto.MarkingTemplate,
) (string, error) {
	id := uuid.New()
	err := s.db.CreateMarkingTemplate(
		context.Background(),
		sqlc.CreateMarkingTemplateParams{
			ID: id.String(),
		},
	)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
