package log

import (
	"context"
	"strconv"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
)

// UsecaseErrors logs use case errors.
func UsecaseErrors(logger ctxd.Logger) usecase.Middleware {
	unknownIndex := 0

	return usecase.MiddlewareFunc(func(u usecase.Interactor) usecase.Interactor {
		var (
			withName  usecase.HasName
			withTitle usecase.HasTitle

			name string
		)

		if usecase.As(u, &withName) {
			name = withName.Name()
		}

		if name == "" && usecase.As(u, &withTitle) {
			name = withTitle.Title()
		}

		if name == "" {
			unknownIndex++
			name = "unnamed" + strconv.Itoa(unknownIndex)
		}

		return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			err := u.Interact(ctx, input, output)
			if err != nil {
				logger.Error(ctx, "usecase failed", "error", err, "name", name)
			}

			return err
		})
	})
}
