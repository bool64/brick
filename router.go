package brick

import (
	"net/http"

	"github.com/bool64/prom-stats"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	v3 "github.com/swaggest/swgui/v3"
)

// NewBaseRouter initializes default http router.
func NewBaseRouter(l *BaseLocator) chi.Router {
	l.HTTPRequestDecoder.ApplyDefaults = true
	l.HTTPRequestDecoder.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	// Create router.
	r := chirouter.NewWrapper(chi.NewRouter())

	// Setup middlewares.
	r.Use(l.HTTPServerMiddlewares...)

	if pt, ok := l.StatsTracker().(*prom.Tracker); ok {
		r.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(pt.Registry, promhttp.HandlerOpts{}))
	}

	if l.BaseConfig.Debug.DevTools {
		MountDevPortal(r, l)
	}

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", l.OpenAPI)
	r.Mount("/docs", v3.NewHandler(l.OpenAPI.Reflector().SpecEns().Info.Title,
		"/docs/openapi.json", "/docs"))

	return r
}
