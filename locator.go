package brick

import (
	"net/http"

	"github.com/acme-corp-tech/brick/debug"
	"github.com/acme-corp-tech/brick/graceful"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
)

// BaseLocator is a basic application agnostic service locator that manages common infrastructure.
type BaseLocator struct {
	BaseConfig BaseConfig

	ctxd.LoggerProvider
	stats.TrackerProvider
	*graceful.Switch
	DebugRouter *debug.Mux

	UseCaseMiddlewares []usecase.Middleware

	HTTPRequestDecoder    *request.DecoderFactory
	HTTPServerMiddlewares []func(h http.Handler) http.Handler
	OpenAPI               *openapi.Collector

	Storage *sqluct.Storage
}
