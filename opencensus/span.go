package opencensus

import (
	"context"
	"errors"

	"github.com/bool64/brick/runtime"
	"github.com/swaggest/usecase/status"
	"go.opencensus.io/trace"
)

// AddSpan starts OpenCensus trace span and returns updated context with callback to finish span.
//
// Span is named by the parent function.
// Typically span should be finished with deferred statement.
//
//		var err error
//		ctx, finish := opencensus.AddSpan(ctx,
//			trace.StringAttribute("key", "value"),
//		)
//		defer func() { finish(err) }()
func AddSpan(ctx context.Context, attributes ...trace.Attribute) (context.Context, func(error)) {
	ctx, span := trace.StartSpan(ctx, runtime.CallerFunc(2))
	span.AddAttributes(attributes...)

	return ctx, func(err error) {
		if err != nil {
			st := status.Unknown

			var ws errorWithStatus
			if errors.As(err, &ws) {
				st = ws.Status()
			}

			span.SetStatus(trace.Status{
				Code:    int32(st),
				Message: err.Error(),
			})
		}

		span.End()
	}
}
