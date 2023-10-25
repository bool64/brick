package brick

import (
	"time"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/bool64/brick/graceful"
	"github.com/bool64/brick/log"
	"github.com/bool64/brick/opencensus"
	ucase "github.com/bool64/brick/usecase"
	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/dev/version"
	"github.com/bool64/logz"
	"github.com/bool64/logz/ctxz"
	"github.com/bool64/prom-stats"
	"github.com/bool64/stats"
	"github.com/bool64/zapctxd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// NoOpLocator creates a dummy service locator, suitable to docs rendering.
func NoOpLocator() *BaseLocator {
	bl := &BaseLocator{}

	bl.OpenAPI = openapi.NewCollector(openapi3.NewReflector())

	ver := version.Info()
	spec := bl.OpenAPI.Reflector().SpecEns()

	if ver.Version == "dev" {
		spec.Info.Version = ver.String()
	} else {
		spec.Info.Version = ver.Version
	}

	if ver.Revision != "" {
		bl.OpenAPI.Reflector().SpecEns().Info.WithMapOfAnythingItem("x-rev", ver.Revision)
	}

	bl.HTTPServiceOptions = append(bl.HTTPServiceOptions, func(s *web.Service) {
		s.OpenAPICollector = bl.OpenAPI
	})

	bl.LoggerProvider = ctxd.NoOpLogger{}
	bl.TrackerProvider = stats.NoOp{}
	bl.cacheInvalidationIndex = cache.NewInvalidationIndex()

	return bl
}

// NewBaseLocator initializes basic service locator.
func NewBaseLocator(cfg BaseConfig) (*BaseLocator, error) {
	l := NoOpLocator()

	l.BaseConfig = cfg

	if !cfg.Initialized {
		return l, nil
	}

	l.Switch = graceful.NewSwitch(cfg.ShutdownTimeout)

	l.LoggerProvider = ctxz.NewObserver(zapctxd.New(cfg.Log).SkipCaller(), logz.Config{
		MaxCardinality: 100,
		MaxSamples:     50,
	})

	l.UseCaseMiddlewares = []usecase.Middleware{
		opencensus.UseCaseMiddleware{},
		ucase.StatsMiddleware(l.StatsTracker()),
		log.UsecaseErrors(l.CtxdLogger()),
	}

	if cfg.Debug.TraceSamplingProbability > 0 {
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.ProbabilitySampler(cfg.Debug.TraceSamplingProbability),
		})
	}

	l.HTTPRecoveryMiddleware = log.HTTPRecover{ // Panic recovery and request logging.
		Logger:      l.CtxdLogger(),
		FieldNames:  l.BaseConfig.Log.FieldNames,
		PrintPanic:  cfg.Log.DevMode,
		ExposePanic: cfg.Debug.ExposePanic,
	}.Middleware()

	l.HTTPServiceOptions = append(l.HTTPServiceOptions, func(s *web.Service) {
		s.PanicRecoveryMiddleware = l.HTTPRecoveryMiddleware
	})

	l.HTTPServerMiddlewares = append(l.HTTPServerMiddlewares,
		opencensus.Middleware, // Tracing.
		log.HTTPTraceTransaction(l.BaseConfig.Log.FieldNames), // Trace transaction.
		nethttp.UseCaseMiddlewares(l.UseCaseMiddlewares...),   // Use case middlewares.
	)

	l.cacheTransfer = &cache.HTTPTransfer{
		Logger: l.CtxdLogger(),
	}

	l.cacheInvalidationIndex = cache.NewInvalidationIndex()

	if err := setupPrometheus(l); err != nil {
		return l, err
	}

	return l, nil
}

func setupPrometheus(l *BaseLocator) error {
	promReg := prometheus.NewRegistry()

	if err := promReg.Register(collectors.NewGoCollector()); err != nil {
		return err
	}

	if err := promReg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return err
	}

	if err := view.Register(opencensus.Views()...); err != nil {
		return err
	}

	if err := view.Register(ocsql.DefaultViews...); err != nil {
		return err
	}

	// Initialize opencensus prometheus exporter.
	promExporter, err := ocprom.NewExporter(ocprom.Options{
		Registry: promReg,
	})
	if err != nil {
		return err
	}

	view.RegisterExporter(promExporter)

	l.OnShutdown("unregister_oc_prom", func() {
		view.Unregister(opencensus.Views()...)
		view.Unregister(ocsql.DefaultViews...)
		view.UnregisterExporter(promExporter)
	})

	view.SetReportingPeriod(time.Second)

	pt, err := prom.NewStatsTracker(promReg)
	if err != nil {
		return err
	}

	l.TrackerProvider = pt

	return nil
}
