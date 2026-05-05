package brick

import (
	"net/http"
	"strings"

	"github.com/bool64/brick/debug"
	"github.com/bool64/brick/debug/zpages"
	"github.com/bool64/brick/telemetry"
	"github.com/bool64/dev/version"
	"github.com/bool64/logz/ctxz"
	"github.com/bool64/logz/logzpage"
	"github.com/bool64/prom-stats"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swg "github.com/swaggest/swgui/v5cdn"
	otelzpages "go.opentelemetry.io/contrib/zpages"
)

// MountDevPortal mounts debug handlers to router.
func MountDevPortal(r chi.Router, l *BaseLocator) {
	cfg := l.BaseConfig

	prefix := cfg.Debug.URL

	r.Route(prefix, func(r chi.Router) {
		if cfg.Debug.DevPassword != "" {
			r.Use(middleware.BasicAuth("Developer Access", map[string]string{"dev": cfg.Debug.DevPassword}))
		}

		r.Use(cfg.Debug.Middlewares...)

		l.SetupDebugRouter()
		r.Mount("/", l.DebugRouter)
	})
}

// SetupDebugRouter initializes a router with debug tools.
func (l *BaseLocator) SetupDebugRouter() {
	if l.DebugRouter != nil {
		return
	}

	l.DebugRouter = newDebugRouter(l, nil)
}

func newDebugRouter(l *BaseLocator, tracezProcessor *otelzpages.SpanProcessor) *debug.Mux {
	cfg := l.BaseConfig

	prefix := cfg.Debug.URL
	dr := debug.NewMux(prefix)

	dr.AddLink("version", "Version")
	dr.Get("/version", version.Handler)

	dr.AddLink("zpages/tracez", "Trace Spans")

	if tracezProcessor != nil {
		if cfg.Debug.TraceURL != "" {
			dr.Mount("/zpages", zpages.Mux(prefix+"/zpages", tracezProcessor, func(traceID string) string {
				return strings.ReplaceAll(cfg.Debug.TraceURL, "{trace_id}", traceID)
			}))
		} else {
			dr.Mount("/zpages", zpages.Mux(prefix+"/zpages", tracezProcessor, nil))
		}
	}

	if pt, ok := l.StatsTracker().(*prom.Tracker); ok {
		dr.AddLink("metrics", "Metrics")
		dr.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(telemetry.PrometheusGatherer(pt.PrometheusRegistry()), promhttp.HandlerOpts{}))
	}

	if lz, ok := l.CtxdLogger().(ctxz.Observer); ok {
		dr.AddLink("logz", "Logs Overview")
		dr.Mount("/logz", logzpage.Handler(lz.LevelObservers()...))
	}

	if l.cacheTransfer != nil && l.cacheTransfer.CachesCount() > 0 {
		dr.AddLink("export-cache", "Export Cache As JSONL")
		dr.AddLink("transfer-cache", "Transfer Cache")
		dr.Method(http.MethodGet, "/export-cache", l.cacheTransfer.ExportJSONL())
		dr.Method(http.MethodGet, "/transfer-cache", l.cacheTransfer.Export())
	}

	dr.AddLink("docs", "API Docs")
	dr.Method(http.MethodGet, "/docs/openapi.json", l.OpenAPI)
	dr.Mount("/docs", swg.NewHandler(l.OpenAPI.Reflector().SpecEns().Info.Title,
		prefix+"/docs/openapi.json", prefix+"/docs"))

	return dr
}
