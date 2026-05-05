package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	bridgeprom "go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MetricsParams configure metrics provider setup.
type MetricsParams struct {
	Config     Config
	Gatherer   prometheus.Gatherer
	OnShutdown ShutdownRegistrar
}

var (
	prometheusGathererMu sync.RWMutex
	prometheusGatherer   prometheus.Gatherer
)

// PrometheusGatherer returns a gatherer that includes native Prometheus metrics and OTel metrics.
func PrometheusGatherer(native prometheus.Gatherer) prometheus.Gatherer {
	prometheusGathererMu.RLock()
	otelGatherer := prometheusGatherer
	prometheusGathererMu.RUnlock()

	switch {
	case native == nil:
		return otelGatherer
	case otelGatherer == nil:
		return native
	default:
		return prometheus.Gatherers{native, otelGatherer}
	}
}

// SetupMetrics configures the global OTel meter provider with Prometheus and optional OTLP readers.
func SetupMetrics(p MetricsParams) error {
	otelRegistry := prometheus.NewRegistry()

	promExporter, err := otelprom.New(otelprom.WithRegisterer(otelRegistry))
	if err != nil {
		return err
	}

	prometheusGathererMu.Lock()
	prometheusGatherer = otelRegistry
	prometheusGathererMu.Unlock()

	readers := []metric.Option{metric.WithReader(promExporter)}

	var producers []metric.Producer

	if p.Gatherer != nil {
		producers = append(producers, bridgeprom.NewMetricProducer(bridgeprom.WithGatherer(p.Gatherer)))
	}

	if reader, err := NewMetricsReader(p.Config, producers...); err != nil {
		return err
	} else if reader != nil {
		readers = append(readers, metric.WithReader(reader))
	}

	meterProvider := metric.NewMeterProvider(readers...)
	otel.SetMeterProvider(meterProvider)

	if p.OnShutdown != nil {
		p.OnShutdown("otel_meter_provider", func() {
			prometheusGathererMu.Lock()
			prometheusGatherer = nil
			prometheusGathererMu.Unlock()

			logShutdownError("otel meter provider", meterProvider.Shutdown(context.Background()))
		})
	}

	return nil
}

// NewMetricsReader creates an OTLP metrics reader if metrics export is configured.
func NewMetricsReader(cfg Config, producers ...metric.Producer) (metric.Reader, error) {
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

	readerOpts := []metric.PeriodicReaderOption{metric.WithInterval(interval)}

	for _, producer := range producers {
		if producer != nil {
			readerOpts = append(readerOpts, metric.WithProducer(producer))
		}
	}

	return metric.NewPeriodicReader(exporter, readerOpts...), nil
}
