package telemetry

import (
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

// Middleware instruments router with OpenTelemetry metrics and traces.
func Middleware(handler http.Handler) http.Handler {
	var withRoute rest.HandlerWithRoute

	if nethttp.HandlerAs(handler, &withRoute) {
		method := withRoute.RouteMethod()
		pattern := withRoute.RoutePattern()
		next := handler

		handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if labeler, ok := otelhttp.LabelerFromContext(req.Context()); ok {
				labeler.Add(attribute.String("http.route", pattern))
			}

			next.ServeHTTP(rw, req)
		})

		return otelhttp.NewHandler(handler, method+" "+pattern,
			otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string {
				return method + " " + pattern
			}),
		)
	}

	return otelhttp.NewHandler(handler, "http.server")
}
