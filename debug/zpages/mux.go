// Package zpages provides OpenTelemetry tracez handlers.
package zpages

import (
	"net/http"

	otelzpages "go.opentelemetry.io/contrib/zpages"
)

// Mux creates a tracez mux to serve at prefixed path.
func Mux(prefix string, sp *otelzpages.SpanProcessor, _ func(traceID string) string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(prefix+"/tracez", otelzpages.NewTracezHandler(sp))
	mux.Handle(prefix+"/tracez/", otelzpages.NewTracezHandler(sp))

	return mux
}
