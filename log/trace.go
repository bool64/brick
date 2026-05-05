package log

import (
	"context"

	"github.com/bool64/ctxd"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type tracerWithLog struct {
	embedded.Tracer
	field string
	next  trace.Tracer
}

func (t *tracerWithLog) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := t.next.Start(ctx, spanName, opts...)
	sc := span.SpanContext()

	if !sc.IsValid() {
		return ctx, span
	}

	return ctxd.SetFields(ctx, t.field, sc.SpanID().String()), span
}

// SpanIDFieldToContexts instruments trace.Tracer to add span ID to context fields.
func SpanIDFieldToContexts(fieldName string, tracer trace.Tracer) trace.Tracer {
	if _, ok := tracer.(*tracerWithLog); ok {
		return tracer
	}

	return &tracerWithLog{next: tracer, field: fieldName}
}
