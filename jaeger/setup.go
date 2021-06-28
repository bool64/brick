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

// Config defines Jaeger settings.
type Config struct {
	// CollectorEndpoint is the full url to the Jaeger HTTP Thrift collector.
	// For example, http://localhost:14268/api/traces
	CollectorEndpoint string `split_words:"true"`

	// AgentEndpoint instructs exporter to send spans to jaeger-agent at this address.
	// For example, localhost:6831.
	AgentEndpoint string `split_words:"true"`

	// OnError is the hook to be called when there is
	// an error occurred when uploading the stats data.
	// If no custom hook is set, errors are logged.
	// Optional.
	OnError func(err error)

	// Username to be used if basic auth is required.
	// Optional.
	Username string

	// Password to be used if basic auth is required.
	// Optional.
	Password string

	// ServiceName is the Jaeger service name.
	ServiceName string `split_words:"true"`

	// Tags are added to Jaeger Process exports.
	Tags []jaeger.Tag

	// BufferMaxCount defines the total number of traces that can be buffered in memory.
	BufferMaxCount int `split_words:"true"`
}

// Setup configures Jaeger Exporter for OpenCensus traces.
//
// Add Jaeger to service configuration:
//    Jaeger jaeger.Config `split_words:"true"`
func Setup(cfg Config, l deps) error {
	if cfg.AgentEndpoint == "" && cfg.CollectorEndpoint == "" {
		l.CtxdLogger().Info(context.Background(), "skipping jaeger setup")

		return nil
	}

	opt := jaeger.Options{}
	opt.CollectorEndpoint = cfg.CollectorEndpoint
	opt.AgentEndpoint = cfg.AgentEndpoint
	opt.Username = cfg.Username
	opt.Password = cfg.Password
	opt.Process.ServiceName = cfg.ServiceName
	opt.Process.Tags = cfg.Tags
	opt.BufferMaxCount = cfg.BufferMaxCount
	opt.OnError = cfg.OnError

	l.CtxdLogger().Info(context.Background(), "setting up jaeger")

	cfg.OnError = func(err error) {
		l.CtxdLogger().Error(context.Background(), "jaeger exporter failed",
			"msg", err.Error(),
			"type", fmt.Sprintf("%T", err),
			"error", err)
	}

	jaegerExporter, err := jaeger.NewExporter(opt)
	if err != nil {
		return err
	}

	trace.RegisterExporter(jaegerExporter)
	l.OnShutdown("unregister_oc_jaeger", func() {
		trace.UnregisterExporter(jaegerExporter)
	})

	return nil
}
