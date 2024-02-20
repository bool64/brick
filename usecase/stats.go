package usecase

import (
	"context"
	"errors"
	"strconv"

	"github.com/bool64/stats"
	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// StatsMiddleware counts use case interactions.
func StatsMiddleware(tracker stats.Tracker) usecase.Middleware {
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
			st := status.OK

			if err != nil {
				st = status.Unknown

				var withStatus rest.ErrWithCanonicalStatus
				if errors.As(err, &withStatus) {
					st = withStatus.Status()
				}
			}

			tracker.Add(ctx, "use_case_interactions_count", 1,
				"name", name,
				"status", st.String(),
			)
			tracker.Add(ctx, "use_case_interactions_total", 1,
				"name", name,
			)

			return err
		})
	})
}
