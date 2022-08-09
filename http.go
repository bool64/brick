package brick

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bool64/prom-stats"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v4emb"
)

// NewBaseWebService initializes default http router.
func NewBaseWebService(l *BaseLocator) *web.Service {
	// Create router.
	r := web.DefaultService(l.HTTPServiceOptions...)

	// Setup middlewares.
	r.Wrap(l.HTTPServerMiddlewares...)

	if pt, ok := l.StatsTracker().(*prom.Tracker); ok {
		r.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(pt.PrometheusRegistry(), promhttp.HandlerOpts{}))
	}

	if l.BaseConfig.Debug.DevTools {
		MountDevPortal(r.Wrapper, l)
	}

	// Swagger UI endpoint at /docs.
	r.Docs("/docs", swgui.New)

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

	if cfg.HTTPListenAddr == "" || cfg.HTTPListenAddr == ":0" {
		cfg.HTTPListenAddr = "127.0.0.1:0"
	}

	listener, err := net.Listen("tcp", cfg.HTTPListenAddr)
	if err != nil {
		return "", fmt.Errorf("failed to start http server: %w", err)
	}

	// Initialize HTTP server.
	srv := http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

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
