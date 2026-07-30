package httpserver

import (
	"net/http"
	"os"

	"ubco-team15/omr/internal/httpserver/handler"
	"ubco-team15/omr/internal/httpserver/middleware"
	"ubco-team15/omr/internal/sys"
)

func Start(hostname, port string) {

	sys.ClearScreen()

	res, err := handler.NewLocalResources("data")
	if err != nil {
		sys.Error("error getting server resources", "err", err)
		os.Exit(1)
	}
	defer res.Close()

	adminKey := os.Getenv("OMR_ADMIN_KEY")
	if adminKey == "" {
		res.SetAdminKey("admin")
		sys.Warn(
			"OMR_ADMIN_KEY not set in the current environment; using default",
			"key", "admin",
		)
	} else {
		res.SetAdminKey(adminKey)
	}

	mux := http.NewServeMux()

	//
	// Developer Endpoints
	//

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)

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

	mux.HandleFunc("GET /system/utilization", handler.GetResourceUtilization(res))
	mux.HandleFunc("GET /system/logs", handler.GetLogs(res))
	mux.HandleFunc("GET /system/cpu", handler.GetCpuInfo(res))
	mux.HandleFunc("GET /system/memory", handler.GetMemoryInfo(res))
	mux.HandleFunc("GET /admin/authenticated", handler.CheckAdminKey(res))
	mux.HandleFunc("PUT /admin/resource-limits", handler.UpdateResourceLimits(res))

	//
	// Deprecated endpoints
	//

	mux.HandleFunc("POST /scan", handler.PostScan(res))
	mux.HandleFunc("GET /snippet", handler.GetSnippet(res))

	//
	// Static Pages
	//

	mux.Handle("GET /", http.FileServer(http.Dir("./pages")))

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

	sys.Info("starting server at http://" + hostname + ":" + port)
	sys.Info(
		"monitor system usage at " +
			"http://" + hostname + ":" + port + "/dashboard.html",
	)
	server.ListenAndServe()
}
