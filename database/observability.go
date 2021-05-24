package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/bool64/ctxd"
	"github.com/bool64/dbwrap"
	"go.opencensus.io/trace"
)

// WithTracing instruments database connector with OpenCensus tracing.
func WithTracing(dbConnector driver.Connector) driver.Connector {
	return ocsql.WrapConnector(dbConnector,
		ocsql.WithQuery(true),
		ocsql.WithRowsClose(true),
		ocsql.WithRowsAffected(true),
		ocsql.WithAllowRoot(true),
		ocsql.WithDisableErrSkip(true),
	)
}

// WithQueriesLogging instruments database connector with query logging.
func WithQueriesLogging(dbConnector driver.Connector, logger ctxd.Logger) driver.Connector {
	if logger == nil {
		logger = ctxd.NoOpLogger{}
	}

	skipPackages := []string{
		"github.com/Masterminds/squirrel",
		"github.com/bool64/sqluct",
		"github.com/jmoiron/sqlx",
	}

	return dbwrap.WrapConnector(dbConnector,
		// This interceptor enables reverse debugging from DB side.
		dbwrap.WithInterceptor(func(ctx context.Context, operation dbwrap.Operation, statement string, args []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
			// Closest caller in the stack with package not equal to listed and to "database/sql".
			caller := dbwrap.Caller(skipPackages...)

			// Add caller name as statement comment.
			return ctx, statement + " -- " + caller, args
		}),

		// This option limits middleware applicability.
		dbwrap.WithOperations(dbwrap.Query, dbwrap.StmtQuery, dbwrap.Exec, dbwrap.StmtExec),

		// This middleware logs statements with arguments at DEBUG level.
		dbwrap.WithMiddleware(log(logger, skipPackages)),
	)
}

func log(logger ctxd.Logger, skipPackages []string) dbwrap.Middleware {
	return func(
		ctx context.Context,
		operation dbwrap.Operation,
		statement string,
		args []driver.NamedValue,
	) (nCtx context.Context, onFinish func(error)) {
		// Exec and Query with args is upgraded to prepared statement.
		if len(args) != 0 && (operation == dbwrap.Exec || operation == dbwrap.Query) {
			return ctx, nil
		}

		// Closest caller in the stack with package not equal to listed and to "database/sql".
		caller := dbwrap.Caller(skipPackages...)

		ctx, span := trace.StartSpan(ctx, caller+":"+string(operation))
		span.AddAttributes(
			trace.StringAttribute("stmt", statement),
			trace.StringAttribute("args", fmt.Sprintf("%v", args)),
		)

		started := time.Now()

		return ctx, func(err error) {
			defer span.End()

			// ErrSkip happens in Exec or Query that is upgraded to prepared statement.
			if errors.Is(err, driver.ErrSkip) {
				return
			}

			res := " complete"

			if err != nil {
				span.SetStatus(trace.Status{Message: err.Error()})

				res = " failed"
			}

			logger.Debug(ctx, caller+" "+string(operation)+res,
				"stmt", statement,
				"args", args,
				"elapsed", time.Since(started).String(),
				"err", err,
			)
		}
	}
}
