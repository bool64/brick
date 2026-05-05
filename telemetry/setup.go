package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/zpages"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ShutdownRegistrar registers shutdown callbacks by name.
type ShutdownRegistrar func(name string, fn func())

// SetupParams configure telemetry bootstrap.
type SetupParams struct {
	ServiceName         string
	Environment         string
	SamplingProbability float64
	Config              Config
	OnShutdown          ShutdownRegistrar
}

// SetupResult keeps initialized implementation details needed by callers.
type SetupResult struct {
	TracezProcessor *zpages.SpanProcessor
	LoggerOptions   []zap.Option
}

// Setup initializes global tracing and optional log export.
func Setup(p SetupParams) (SetupResult, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", p.ServiceName),
			attribute.String("deployment.environment", p.Environment),
		),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return SetupResult{}, err
	}

	tracezProcessor := zpages.NewSpanProcessor()
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(p.SamplingProbability))),
		sdktrace.WithSpanProcessor(tracezProcessor),
	}

	if exp, err := setupTraceExporter(p.Config); err != nil {
		return SetupResult{}, err
	} else if exp != nil {
		options = append(options, sdktrace.WithBatcher(exp))
	}

	tracerProvider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if p.OnShutdown != nil {
		p.OnShutdown("otel_tracer_provider", func() {
			logShutdownError("otel tracer provider", tracerProvider.Shutdown(context.Background()))
		})
	}

	logHandler, err := setupLogs(p.ServiceName, p.Config, res, p.OnShutdown)
	if err != nil {
		return SetupResult{}, err
	}

	return SetupResult{
		TracezProcessor: tracezProcessor,
		LoggerOptions:   loggerOptions(logHandler),
	}, nil
}

func setupTraceExporter(cfg Config) (sdktrace.SpanExporter, error) {
	if cfg.TraceEndpoint() == "" {
		return nil, nil
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.TraceEndpoint()),
	}

	if headers := cfg.Headers(); headers != nil {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp trace exporter: %w", err)
	}

	return exporter, nil
}

func setupLogs(serviceName string, cfg Config, res *resource.Resource, onShutdown ShutdownRegistrar) (slog.Handler, error) {
	if cfg.LogEndpoint() == "" {
		return nil, nil
	}

	opts := []otlploghttp.Option{
		otlploghttp.WithEndpointURL(cfg.LogEndpoint()),
	}

	if headers := cfg.Headers(); headers != nil {
		opts = append(opts, otlploghttp.WithHeaders(headers))
	}

	exporter, err := otlploghttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp log exporter: %w", err)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	logHandler := otelslog.NewHandler(serviceName,
		otelslog.WithLoggerProvider(loggerProvider),
	)

	if onShutdown != nil {
		onShutdown("otel_log_provider", func() {
			logShutdownError("otel log provider", loggerProvider.Shutdown(context.Background()))
		})
	}

	return logHandler, nil
}

func loggerOptions(logHandler slog.Handler) []zap.Option {
	if logHandler == nil {
		return nil
	}

	return []zap.Option{
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, NewZapCore(logHandler, zap.DebugLevel))
		}),
	}
}

func logShutdownError(name string, err error) {
	if err != nil {
		slog.Default().Error("telemetry shutdown failed", "component", name, "error", err)
	}
}
