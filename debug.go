package brick

import (
	"net/http"
	"strings"

	"github.com/bool64/brick/debug"
	"github.com/bool64/brick/debug/zpages"
	"github.com/bool64/dev/version"
	"github.com/bool64/logz/ctxz"
	"github.com/bool64/logz/logzpage"
	"github.com/bool64/prom-stats"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v3 "github.com/swaggest/swgui/v3"
)

// MountDevPortal mounts debug handlers to router.
func MountDevPortal(r chi.Router, l *BaseLocator) {
	cfg := l.BaseConfig

	prefix := cfg.Debug.URL

	r.Route(prefix, func(r chi.Router) {
		if cfg.Debug.DevPassword != "" {
			r.Use(middleware.BasicAuth("Developer Access", map[string]string{"dev": cfg.Debug.DevPassword}))
		}

		l.SetupDebugRouter()
		r.Mount("/", l.DebugRouter)
	})
}

// SetupDebugRouter initializes a router with debug tools.
func (l *BaseLocator) SetupDebugRouter() {
	if l.DebugRouter != nil {
		return
	}

	cfg := l.BaseConfig

	prefix := cfg.Debug.URL
	dr := debug.NewMux(prefix)

	dr.AddLink("version", "Version")
	dr.Get("/version", version.Handler)

	dr.AddLink("zpages/tracez", "Trace Spans")

	if cfg.Debug.TraceURL != "" {
		dr.Mount("/zpages", zpages.Mux(prefix+"/zpages", func(traceID string) string {
			return strings.ReplaceAll(cfg.Debug.TraceURL, "{trace_id}", traceID)
		}))
	} else {
		dr.Mount("/zpages", zpages.Mux(prefix+"/zpages", nil))
	}

	if pt, ok := l.StatsTracker().(*prom.Tracker); ok {
		dr.AddLink("metrics", "Metrics")
		dr.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(pt.Registry, promhttp.HandlerOpts{}))
	}

	if lz, ok := l.CtxdLogger().(ctxz.Observer); ok {
		dr.AddLink("logz", "Logs Overview")
		dr.Mount("/logz", logzpage.Handler(lz.LevelObservers()...))
	}

	dr.AddLink("docs", "API Docs")
	dr.Method(http.MethodGet, "/docs/openapi.json", l.OpenAPI)
	dr.Mount("/docs", v3.NewHandler(l.OpenAPI.Reflector().SpecEns().Info.Title,
		prefix+"/docs/openapi.json", prefix+"/docs"))

	l.DebugRouter = dr
}
