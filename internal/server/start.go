package server

import (
	"net/http"
	"os"
	"path/filepath"
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

	debug.SetMemoryLimit(SoftMemoryLimit)

	s, err := resources.NewLocal("data")
	if err != nil {
		sys.Error("error getting server resources", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	j := mw.NewJobRegistrar(time.Hour * 24)
	defer j.Close()

	//
	// Keys
	//

	k := mw.NewKeyholder()

	adminKey := os.Getenv("OMR_ADMIN_KEY")
	if adminKey == "" {
		k.SetAdminKey(DefaultAdminKey)
		sys.Warn(
			"OMR_ADMIN_KEY not set in the current environment; using default",
			"key", DefaultAdminKey,
		)
	} else {
		k.SetAdminKey(adminKey)
	}

	if globalKey == "" {
		globalKey = os.Getenv("OMR_GLOBAL_KEY")
	}
	if globalKey != "" {
		k.SetGlobalKey(globalKey)
	}

	//
	// Routes
	//

	core := newRouteGroup(
		mw.Cors,
		mw.GlobalKey(k),
		mw.Logger,
		mw.Recovery,
	)

	core.routes["POST /template/mark"] = handler.PostMarkingTemplate(s)
	core.routes["DELETE /template/mark"] = handler.DeleteMarkingTemplate(s)
	core.routes["POST /template/preprocess"] = handler.PostPreprocessingTemplate(s)
	core.routes["DELETE /template/preprocess"] = handler.DeletePreprocessingTemplate(s)
	core.routes["POST /scan/pdf"] = j.SyncJob(handler.PostScanPdf(s))
	core.routes["POST /scan/pdf/async"] = j.AsyncJob(handler.PostScanPdf(s))
	core.routes["DELETE /scan"] = handler.DeleteScan(s)
	core.routes["POST /mark"] = j.SyncJob(handler.RequestMarks(s))
	core.routes["POST /mark/async"] = j.AsyncJob(handler.RequestMarks(s))
	core.routes["GET /image/snippet"] = handler.GetSnippet(s)
	core.routes["GET /image"] = handler.GetImage(s)
	core.routes["GET /job"] = j.Handler()
	core.routes["GET /job/result"] = j.ResultHandler()
	core.routes["GET /openapi.yaml"] = handler.OpenAPISpec
	core.routes["GET /ping"] = handler.Ping

	noLog := newRouteGroup(
		mw.Cors,
		mw.GlobalKey(k),
		mw.Recovery,
	)

	noLog.routes["GET /system/utilization"] = handler.GetResourceUtilization(s)
	noLog.routes["GET /system/logs"] = handler.GetLogs(s)
	noLog.routes["GET /system/cpu"] = handler.GetCpuInfo(s)
	noLog.routes["GET /system/memory"] = handler.GetMemoryInfo(s)	

	admin := newRouteGroup(
		mw.Cors,
		mw.GlobalKey(k),
		mw.AdminKey(k),
		mw.Logger,
		mw.Recovery,
	)

	admin.routes["GET /jobs"] = j.ListHandler()
	admin.routes["GET /admin/authenticated"] = handler.CheckAdminKey(s)
	admin.routes["DELETE /admin/scans"] = handler.DeleteScansOlderThan(s)

	mux := http.NewServeMux()

	core.register(mux)
	admin.register(mux)
	noLog.register(mux)

	mux.Handle("GET /", http.FileServer(http.Dir("./pages")))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server
}

//
// Startup functions
//

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

	err := server.ListenAndServeTLS(
		filepath.Join(certDir, "cert.pem"), 
		filepath.Join(certDir, "key.pem"),
	)
	if err != nil && err != http.ErrServerClosed {
		sys.Error("failed to find valid TLS key files. consider running the generate-cert subcommand")
	}
}

//
// Helper type to make scoping middleware a little simpler.
//

type routeGroup struct {
	middleware []middleware
	routes     route
}

func newRouteGroup(middleware ...middleware) routeGroup {
	return routeGroup{
		middleware: middleware,
		routes: make(route),
	}
}

type middleware func(http.Handler) http.Handler
type route map[string]http.Handler

func (g *routeGroup) register(s *http.ServeMux) {
	for endpoint, handler := range g.routes {
		handler := http.Handler(handler)
		for _, middleware := range g.middleware {
			handler = middleware(handler)
		}
		s.Handle(endpoint, handler)
	}
}
