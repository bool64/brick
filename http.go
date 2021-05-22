package brick

import (
	"context"
	"errors"
	"fmt"
	"net"
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

// StartHTTPServer starts HTTP server with provided handler
// in a goroutine and returns listening addr or error.
//
// Server will listen at BaseConfig.HTTPListenAddr, if the value
// is empty free random port will be used.
//
// Server will be gracefully stopped on service locator shutdown.
func (l *BaseLocator) StartHTTPServer(handler http.Handler) (string, error) {
	cfg := l.BaseConfig

	if cfg.HTTPListenAddr == "" {
		cfg.HTTPListenAddr = ":0"
	}

	listener, err := net.Listen("tcp", cfg.HTTPListenAddr)
	if err != nil {
		return "", fmt.Errorf("failed to start http server: %w", err)
	}

	// Initialize HTTP server.
	srv := http.Server{Handler: handler}

	// Start HTTP server.
	l.CtxdLogger().Important(context.Background(), fmt.Sprintf("starting server, Swagger UI at http://%s/docs",
		listener.Addr().String()))

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.CtxdLogger().Error(context.Background(), err.Error())
			l.Shutdown()
		}
	}()

	// Wait for termination signal and HTTP shutdown finished.
	l.OnShutdown("http", func() {
		err := srv.Shutdown(context.Background())
		if err != nil {
			l.CtxdLogger().Error(context.Background(), "failed to shutdown http", "error", err.Error())
		}
	})

	return listener.Addr().String(), nil
}
