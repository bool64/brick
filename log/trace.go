package log

import (
	"context"

	"github.com/bool64/ctxd"
	"go.opencensus.io/trace"
)

type tracerWithLog struct {
	field string
	trace.Tracer
}

func (t *tracerWithLog) NewContext(parent context.Context, s *trace.Span) context.Context {
	ctx := t.Tracer.NewContext(parent, s)
	sc := s.SpanContext()

	return ctxd.SetFields(ctx, t.field, sc.SpanID)
}

// SpanIDFieldToContexts instruments trace.Tracer to add SpanID to context fields.
func SpanIDFieldToContexts(fieldName string, tracer trace.Tracer) trace.Tracer {
	if _, ok := tracer.(*tracerWithLog); ok {
		return tracer
	}

	return &tracerWithLog{Tracer: tracer, field: fieldName}
}
