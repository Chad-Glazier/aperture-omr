package server

import (
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/server/handler"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const (
	SoftMemoryLimit = 256 << 20
	DefaultAdminKey = "admin"
)

func setup(port string, globalKey string) *http.Server {

	//
	// Setup/Configuration
	//

	debug.SetMemoryLimit(SoftMemoryLimit)

	s, err := resources.NewLocal("data")
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

	if globalKey == "" {
		globalKey = os.Getenv("OMR_GLOBAL_KEY")
	}
	if globalKey != "" {
		s.SetGlobalKey(globalKey)
	}

	//
	// Core API
	//

	api := group{
		middleware: []middleware{
			mw.Cors,
			mw.GlobalKey(s),
			mw.Logger,
			mw.Recovery,
		},
		routes: []route{
			{endpoint: "POST /template/mark",
				handler: handler.PostMarkingTemplate(s),
			},
			{endpoint: "DELETE /template/mark",
				handler: handler.DeleteMarkingTemplate(s),
			},
			{endpoint: "POST /template/preprocess",
				handler: handler.PostPreprocessingTemplate(s),
			},
			{endpoint: "DELETE /template/preprocess",
				handler: handler.DeletePreprocessingTemplate(s),
			},
			{endpoint: "POST /scan/pdf",
				handler: j.SyncJob(handler.PostScanPdf(s)),
			},
			{endpoint: "POST /scan/pdf/async",
				handler: j.AsyncJob(handler.PostScanPdf(s)),
			},
			{endpoint: "DELETE /scan",
				handler: handler.DeleteScan(s),
			},
			{endpoint: "POST /mark",
				handler: j.SyncJob(handler.RequestMarks(s)),
			},
			{endpoint: "POST /mark/async",
				handler: j.AsyncJob(handler.RequestMarks(s)),
			},
			{endpoint: "GET /image/snippet",
				handler: handler.GetSnippet(s),
			},
			{endpoint: "GET /image",
				handler: handler.GetImage(s),
			},
			{endpoint: "GET /job",
				handler: j.Handler(),
			},
			{endpoint: "GET /jobs",
				handler: j.ListHandler(s),
			},
			{endpoint: "GET /job/result",
				handler: j.ResultHandler(),
			},
			{endpoint: "GET /openapi.yaml",
				handler: handler.OpenAPISpec,
			},
			{endpoint: "GET /ping",
				handler: handler.Ping,
			},
			{endpoint: "GET /system/utilization",
				handler: handler.GetResourceUtilization(s),
			},
			{endpoint: "GET /system/logs",
				handler: handler.GetLogs(s),
			},
			{endpoint: "GET /system/cpu",
				handler: handler.GetCpuInfo(s),
			},
			{endpoint: "GET /system/memory",
				handler: handler.GetMemoryInfo(s),
			},
			{endpoint: "GET /admin/authenticated",
				handler: handler.CheckAdminKey(s),
			},
			{endpoint: "DELETE /admin/scans",
				handler: handler.DeleteScansOlderThan(s),
			},
		},
	}

	//
	// Static Pages
	//

	mux := http.NewServeMux()

	api.register(mux)

	mux.Handle("GET /", http.FileServer(http.Dir("./pages")))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server
}

func Start(hostname, port, globalKey string) {

	server := setup(port, globalKey)

	baseUrl := "http://" + hostname + ":" + port
	sys.Log("starting server on port " + port)
	sys.Info("view documentation at " + baseUrl)
	sys.Info("monitor system usage at " + baseUrl + "/dashboard.html")

	server.ListenAndServe()
}

func StartTls(hostname, port, globalKey, certDir string) {

	server := setup(port, globalKey)

	baseUrl := "https://" + hostname + ":" + port
	sys.Log("starting server on port " + port)
	sys.Info("view documentation at " + baseUrl)
	sys.Info("monitor system usage at " + baseUrl + "/dashboard.html")

	err := server.ListenAndServeTLS(certDir+"cert.pem", certDir+"key.pem")
	if err != nil && err != http.ErrServerClosed {
		sys.Error("failed to find valid TLS key files. consider running the generate-cert subcommand")
	}
}

type group struct {
	middleware []middleware
	routes     []route
}

type middleware func(http.Handler) http.Handler

type route struct {
	middleware []middleware
	endpoint   string
	handler    http.Handler
}

func (g *group) register(s *http.ServeMux) {
	for _, r := range g.routes {
		handler := http.Handler(r.handler)
		for _, middleware := range r.middleware {
			handler = middleware(handler)
		}
		for _, middleware := range g.middleware {
			handler = middleware(handler)
		}
		s.Handle(r.endpoint, handler)
	}
}
