package httpserver

import (
	"log/slog"
	"net/http"
	"os"

	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"
	"ubco-team15/omr/internal/pdf"
)

func Start(hostname, port string) {

	res, err := handler.NewLocalResources("data")
	if err != nil {
		slog.Error("error getting server resources", "err", err)
		os.Exit(1)
	}
	defer res.Close()

	pdf.Init(res.DB)

	mux := http.NewServeMux()

	//
	// Development
	//

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /", handler.DocsPage)

	//
	// Core API
	//

	mux.HandleFunc("GET /health", handler.Health)

	mux.HandleFunc("POST /template/mark", handler.PostMarkingTemplate(res))
	mux.HandleFunc("DELETE /template/mark", handler.DeleteMarkingTemplate(res))
	mux.HandleFunc("POST /template/preprocess", handler.PostPreprocessingTemplate(res))
	mux.HandleFunc("DELETE /template/preprocess", handler.DeletePreprocessingTemplate(res))

	mux.HandleFunc("POST /scan/images", handler.PostScan(res))
	mux.HandleFunc("POST /scan/pdf", handler.PostScanPdf(res))
	mux.HandleFunc("DELETE /scan", handler.DeleteScans(res))

	mux.HandleFunc("POST /mark", handler.PostMarkingJob(res))

	mux.HandleFunc("GET /image/snippet", handler.GetSnippet(res))
	mux.HandleFunc("GET /image", handler.GetImage(res))

	//
	// Deprecated endpoints
	//

	mux.HandleFunc("POST /scan", handler.PostScan(res))
	mux.HandleFunc("GET /snippet", handler.GetSnippet(res))

	//
	// Middleware
	//

	httpHandler := middleware.Cors(mux)
	httpHandler = middleware.Recovery(httpHandler)
	httpHandler = middleware.Logger(httpHandler)

	//
	// Serve!
	//

	server := &http.Server{
		Addr:    ":" + port,
		Handler: httpHandler,
	}

	slog.Info("starting server at http://" + hostname + ":" + port)
	server.ListenAndServe()
}
