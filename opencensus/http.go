package opencensus

import (
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"go.opencensus.io/plugin/ochttp"
)

// Middleware instruments router with OpenCensus metrics.
func Middleware(handler http.Handler) http.Handler {
	var withRoute rest.HandlerWithRoute

	if nethttp.HandlerAs(handler, &withRoute) {
		method := withRoute.RouteMethod()
		pattern := withRoute.RoutePattern()

		return ochttp.WithRouteTag(&ochttp.Handler{
			FormatSpanName: func(_ *http.Request) string {
				return method + " " + pattern
			},
			Handler: handler,
		}, method+pattern)
	}

	return &ochttp.Handler{
		Handler: handler,
	}
}
