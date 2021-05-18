package opencensus

import (
	"context"
	"errors"
	"fmt"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"go.opencensus.io/trace"
)

type errorWithStatus interface {
	Status() status.Code
}

// UseCaseMiddleware is a tracing usecase middleware.
type UseCaseMiddleware struct {
	WithInput bool
}

// Wrap makes an instrumented use case interactor.
func (mw UseCaseMiddleware) Wrap(u usecase.Interactor) usecase.Interactor {
	var (
		withName  usecase.HasName
		withTitle usecase.HasTitle
		spanName  string
	)

	if usecase.As(u, &withName) && withName.Name() != "" {
		spanName = withName.Name()
	} else if usecase.As(u, &withTitle) && withTitle.Title() != "" {
		spanName = withTitle.Title()
	}

	if spanName == "" {
		spanName = "useCaseUnknown"
	}

	return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		ctx, span := trace.StartSpan(ctx, spanName)
		if mw.WithInput {
			span.AddAttributes(trace.StringAttribute("input", fmt.Sprintf("%v", input)))
		}

		defer span.End()

		err := u.Interact(ctx, input, output)
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

		return err
	})
}
