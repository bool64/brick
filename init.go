package brick

import (
	"time"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/bool64/brick/graceful"
	"github.com/bool64/brick/log"
	"github.com/bool64/brick/opencensus"
	ucase "github.com/bool64/brick/usecase"
	"github.com/bool64/ctxd"
	"github.com/bool64/logz"
	"github.com/bool64/logz/ctxz"
	"github.com/bool64/prom-stats"
	"github.com/bool64/stats"
	"github.com/bool64/zapctxd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// NoOpLocator creates a dummy service locator, suitable to docs rendering.
func NoOpLocator() *BaseLocator {
	bl := &BaseLocator{}

	bl.OpenAPI = &openapi.Collector{}
	bl.HTTPRequestDecoder = request.NewDecoderFactory()
	bl.LoggerProvider = ctxd.NoOpLogger{}
	bl.TrackerProvider = stats.NoOp{}

	bl.HTTPServerMiddlewares = append(bl.HTTPServerMiddlewares,
		nethttp.OpenAPIMiddleware(bl.OpenAPI), // Documentation collector.
	)

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

	l.HTTPRequestDecoder = request.NewDecoderFactory()

	l.UseCaseMiddlewares = []usecase.Middleware{
		opencensus.UseCaseMiddleware{},
		ucase.StatsMiddleware(l.StatsTracker()),
	}

	if cfg.Debug.TraceSamplingProbability > 0 {
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.ProbabilitySampler(cfg.Debug.TraceSamplingProbability),
		})
	}

	// Init request decoder and validator.
	validatorFactory := jsonschema.NewFactory(l.OpenAPI, l.OpenAPI)

	l.HTTPServerMiddlewares = append(l.HTTPServerMiddlewares,
		log.HTTPRecover{
			Logger:      l.CtxdLogger(),
			FieldNames:  l.BaseConfig.Log.FieldNames,
			PrintPanic:  cfg.Log.DevMode,
			ExposePanic: cfg.Debug.ExposePanic,
		}.Middleware(), // Panic recovery and request logging.
		opencensus.Middleware, // Tracing.
		log.HTTPTraceTransaction(l.BaseConfig.Log.FieldNames), // Trace transaction.
		nethttp.UseCaseMiddlewares(l.UseCaseMiddlewares...),   // Use case middlewares.
		request.DecoderMiddleware(l.HTTPRequestDecoder),       // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory),         // Request validator setup.
		response.EncoderMiddleware,                            // Response encoder setup.
	)

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
