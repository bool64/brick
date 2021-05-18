package brick

import (
	"net/http"
	"strings"

	"github.com/bool64/brick/debug"
	"github.com/bool64/logz/ctxz"
	"github.com/bool64/logz/logzpage"
	"github.com/bool64/prom-stats"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v3 "github.com/swaggest/swgui/v3"
)

// MountDevPortal mounts debug handlers to router.
func MountDevPortal(r chi.Router, l *BaseLocator, options ...func(options *debug.RouterConfig)) {
	cfg := l.BaseConfig

	if cfg.Debug.TraceURL != "" {
		options = append(options, func(options *debug.RouterConfig) {
			options.TraceToURL = func(traceID string) string {
				return strings.ReplaceAll(cfg.Debug.TraceURL, "{trace_id}", traceID)
			}
		})
	}

	options = append(options, cfg.Debug.RouterConfig...)

	prefix := cfg.Debug.URL

	r.Route(prefix, func(r chi.Router) {
		if cfg.Debug.DevPassword != "" {
			r.Use(middleware.BasicAuth("Developer Access", map[string]string{"dev": cfg.Debug.DevPassword}))
		}

		options = append([]func(options *debug.RouterConfig){debug.Prefix(prefix)}, options...)

		r.Mount("/", DebugHandler(l, options...))
	})
}

// DebugHandler provides developer tools via http.
func DebugHandler(l *BaseLocator, options ...func(options *debug.RouterConfig)) *chi.Mux {
	cfg := debug.RouterConfig{
		Links:  map[string]string{},
		Prefix: "/debug",
	}

	for _, o := range options {
		o(&cfg)
	}

	if l != nil {
		if pt, ok := l.StatsTracker().(*prom.Tracker); ok {
			cfg.Links["metrics"] = "Metrics"
			cfg.Routes = append(cfg.Routes, func(r *chi.Mux) {
				r.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(pt.Registry, promhttp.HandlerOpts{}))
			})
		}

		if lz, ok := l.CtxdLogger().(ctxz.Observer); ok {
			cfg.Links["logz"] = "Logs Overview"
			cfg.Routes = append(cfg.Routes, func(r *chi.Mux) {
				r.Mount("/logz", logzpage.Handler(lz.LevelObservers()...))
			})
		}

		cfg.Links["docs"] = "API Docs"
		cfg.Routes = append(cfg.Routes, func(r *chi.Mux) {
			r.Method(http.MethodGet, "/docs/openapi.json", l.OpenAPI)
			r.Mount("/docs", v3.NewHandler(l.OpenAPI.Reflector().SpecEns().Info.Title,
				cfg.Prefix+"/docs/openapi.json", cfg.Prefix+"/docs"))
		})
	}

	debugRouter := debug.Index(cfg)

	return debugRouter
}
