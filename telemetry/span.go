package telemetry

import (
	"context"

	"github.com/bool64/brick/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AddSpan starts OpenTelemetry span and returns updated context with callback to finish span.
//
// Span is named by the parent function.
// Typically, span should be finished with deferred statement.
//
//	var err error
//	ctx, finish := telemetry.AddSpan(ctx,
//		attribute.String("key", "value"),
//	)
//	defer finish(&err)
func AddSpan(ctx context.Context, attributes ...attribute.KeyValue) (context.Context, func(*error)) {
	ctx, span := otel.Tracer("github.com/bool64/brick/telemetry").Start(ctx, runtime.CallerFunc(2))
	span.SetAttributes(attributes...)

	return ctx, func(err *error) {
		if err != nil && *err != nil {
			e := *err
			span.SetStatus(codes.Error, e.Error())
			span.RecordError(e)
		}

		span.End()
	}
}
