package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MetricsParams configure metrics provider setup.
type MetricsParams struct {
	Config     Config
	Registerer prometheus.Registerer
	OnShutdown ShutdownRegistrar
}

// SetupMetrics configures the global OTel meter provider with Prometheus and optional OTLP readers.
func SetupMetrics(p MetricsParams) error {
	promExporter, err := otelprom.New(otelprom.WithRegisterer(p.Registerer))
	if err != nil {
		return err
	}

	readers := []metric.Option{metric.WithReader(promExporter)}
	if reader, err := NewMetricsReader(p.Config); err != nil {
		return err
	} else if reader != nil {
		readers = append(readers, metric.WithReader(reader))
	}

	meterProvider := metric.NewMeterProvider(readers...)
	otel.SetMeterProvider(meterProvider)

	if p.OnShutdown != nil {
		p.OnShutdown("otel_meter_provider", func() {
			_ = meterProvider.Shutdown(context.Background())
		})
	}

	return nil
}

// NewMetricsReader creates an OTLP metrics reader if metrics export is configured.
func NewMetricsReader(cfg Config) (metric.Reader, error) {
	if cfg.MetricEndpoint() == "" {
		return nil, nil
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(cfg.MetricEndpoint()),
	}

	if headers := cfg.Headers(); headers != nil {
		opts = append(opts, otlpmetrichttp.WithHeaders(headers))
	}

	exporter, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp metrics exporter: %w", err)
	}

	interval := cfg.MetricsInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}

	return metric.NewPeriodicReader(exporter, metric.WithInterval(interval)), nil
}
