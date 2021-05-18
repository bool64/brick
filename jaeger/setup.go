// Package jaeger configures OpenCensus Jaeger exporter.
package jaeger

import (
	"context"
	"fmt"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/bool64/ctxd"
	"go.opencensus.io/trace"
)

type deps interface {
	CtxdLogger() ctxd.Logger
	OnShutdown(name string, fn func())
}

// Setup configures Jaeger Exporter for OpenCensus traces.
//
// Add Jaeger to service configuration:
//    Jaeger jaeger.Options `split_words:"true"`
func Setup(cfg jaeger.Options, serviceName string, l deps) error {
	if cfg.AgentEndpoint == "" && cfg.CollectorEndpoint == "" {
		l.CtxdLogger().Info(context.Background(), "skipping jaeger setup")

		return nil
	}

	if serviceName != "" && cfg.Process.ServiceName == "" {
		cfg.Process.ServiceName = serviceName
	}

	l.CtxdLogger().Info(context.Background(), "setting up jaeger")

	cfg.OnError = func(err error) {
		l.CtxdLogger().Error(context.Background(), "jaeger exporter failed",
			"msg", err.Error(),
			"type", fmt.Sprintf("%T", err),
			"error", err)
	}

	jaegerExporter, err := jaeger.NewExporter(cfg)
	if err != nil {
		return err
	}

	trace.RegisterExporter(jaegerExporter)
	l.OnShutdown("unregister_oc_jaeger", func() {
		trace.UnregisterExporter(jaegerExporter)
	})

	return nil
}
