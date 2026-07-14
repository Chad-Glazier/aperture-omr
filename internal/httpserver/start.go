package httpserver

import (
	"log/slog"
	"net/http"
	"os"

	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"
)

func Start(hostname, port string) {

	res, err := handler.NewLocalResources("data")
	if err != nil {
		slog.Error("error getting server resources", "err", err)
		os.Exit(1)
	}
	defer res.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)
	mux.HandleFunc("GET /health", handler.Health)

	mux.HandleFunc("POST /template/mark", handler.PostMarkingTemplate(res))
	mux.HandleFunc("POST /template/preprocess", handler.PostPreprocessingTemplate(res))
	mux.HandleFunc("POST /scan", handler.PostScan(res))
	mux.HandleFunc("POST /scan/pdf", handler.PostScanPdf(res))
	mux.HandleFunc("POST /mark", handler.PostMarkingJob(res))
	mux.HandleFunc("GET /snippet", handler.GetSnippet(res))

	httpHandler := middleware.Cors(mux)
	httpHandler = middleware.Recovery(httpHandler)
	httpHandler = middleware.Logger(httpHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: httpHandler,
	}

	slog.Info("starting server at http://" + hostname + ":" + port)
	server.ListenAndServe()
}
