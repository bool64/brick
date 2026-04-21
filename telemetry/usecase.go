package telemetry

import (
	"context"
	"fmt"

	"github.com/swaggest/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// UseCaseMiddleware is a tracing use case middleware.
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
		ctx, span := otel.Tracer("github.com/bool64/brick/telemetry").Start(ctx, spanName)
		if mw.WithInput {
			span.SetAttributes(attribute.String("input", fmt.Sprintf("%v", input)))
		}

		defer span.End()

		err := u.Interact(ctx, input, output)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		return err
	})
}
