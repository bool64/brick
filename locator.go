package brick

import (
	"net/http"

	"github.com/bool64/brick/debug"
	"github.com/bool64/brick/graceful"
	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/swgui"
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

	// HTTPServiceOptions can be used to configure low-level middlewares like middleware.StripSlashes on an
	// initialized web.Service.
	HTTPServiceOptions     []func(s *web.Service, initialized bool)
	HTTPRecoveryMiddleware func(h http.Handler) http.Handler
	HTTPServerMiddlewares  []func(h http.Handler) http.Handler
	OpenAPI                *openapi.Collector
	SwaggerUIOptions       []func(cfg *swgui.Config)

	Storage       *sqluct.Storage
	CacheTransfer *cache.HTTPTransfer
}
