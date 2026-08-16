package server

import (
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/server/handler"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/server/res"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const (
	SoftMemoryLimit = 256 << 20
	DefaultAdminKey = "admin"
)

func setup(port string) *http.Server {

	//
	// Setup/Configuration
	//

	debug.SetMemoryLimit(SoftMemoryLimit)

	s, err := res.NewLocalResources("data")
	if err != nil {
		sys.Error("error getting server resources", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	j := mw.NewJobRegistrar(time.Hour * 24)
	defer j.Close()

	adminKey := os.Getenv("OMR_ADMIN_KEY")
	if adminKey == "" {
		s.SetAdminKey(DefaultAdminKey)
		sys.Warn(
			"OMR_ADMIN_KEY not set in the current environment; using default",
			"key", DefaultAdminKey,
		)
	} else {
		s.SetAdminKey(adminKey)
	}

	mux := http.NewServeMux()

	//
	// Core API
	//

	mux.HandleFunc("POST /template/mark", handler.PostMarkingTemplate(s))
	mux.HandleFunc("DELETE /template/mark", handler.DeleteMarkingTemplate(s))

	mux.HandleFunc("POST /template/preprocess", handler.PostPreprocessingTemplate(s))
	mux.HandleFunc("DELETE /template/preprocess", handler.DeletePreprocessingTemplate(s))

	mux.HandleFunc("POST /scan/pdf", j.SyncJob(handler.PostScanPdf(s)))
	mux.HandleFunc("POST /scan/pdf/async", j.AsyncJob(handler.PostScanPdf(s)))

	mux.HandleFunc("DELETE /scan", handler.DeleteScan(s))

	mux.HandleFunc("POST /mark", j.SyncJob(handler.RequestMarks(s)))
	mux.HandleFunc("POST /mark/async", j.AsyncJob(handler.RequestMarks(s)))

	mux.HandleFunc("GET /image/snippet", handler.GetSnippet(s))
	mux.HandleFunc("GET /image", handler.GetImage(s))

	mux.HandleFunc("GET /job", j.Handler())
	mux.HandleFunc("GET /jobs", j.ListHandler(s))
	mux.HandleFunc("GET /job/result", j.ResultHandler())

	//
	// System Administration Endpoints
	//

	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)
	mux.HandleFunc("GET /ping", handler.Ping)
	mux.HandleFunc("GET /system/utilization", handler.GetResourceUtilization(s))
	mux.HandleFunc("GET /system/logs", handler.GetLogs(s))
	mux.HandleFunc("GET /system/cpu", handler.GetCpuInfo(s))
	mux.HandleFunc("GET /system/memory", handler.GetMemoryInfo(s))

	mux.HandleFunc("GET /admin/authenticated", handler.CheckAdminKey(s))

	//
	// Static Pages
	//

	mux.Handle("GET /", http.FileServer(http.Dir("./pages")))

	//
	// Middleware
	//

	httpHandler := mw.Cors(mux)
	httpHandler = mw.Recovery(httpHandler)
	httpHandler = mw.Logger(httpHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: httpHandler,
	}

	return server
}

func Start(hostname, port string) {

	server := setup(port)

	baseUrl := "http://" + hostname + ":" + port
	sys.Log("failed to find TLS key files. consider running the generate-cert subcommand")
	sys.Log("starting server at on port " + port)
	sys.Info("view documentation at " + baseUrl)
	sys.Info("monitor system usage at " + baseUrl + "/dashboard.html")

	server.ListenAndServe()
}

func StartTls(hostname, port, certDir string) {

	server := setup(port)

	baseUrl := "https://" + hostname + ":" + port
	sys.Log("failed to find TLS key files. consider running the generate-cert subcommand")
	sys.Log("starting server at on port " + port)
	sys.Info("view documentation at " + baseUrl)
	sys.Info("monitor system usage at " + baseUrl + "/dashboard.html")

	server.ListenAndServeTLS(certDir+"cert.pem", certDir+"key.pem")
}
