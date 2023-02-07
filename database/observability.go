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
	"github.com/bool64/stats"
	"go.opencensus.io/trace"
)

// WithTracing instruments database connector with OpenCensus tracing.
func WithTracing(dbConnector driver.Connector) driver.Connector {
	return ocsql.WrapConnector(dbConnector, tracingOptions()...)
}

// DriverNameWithTracing registers database driver name with OpenCensus tracing.
func DriverNameWithTracing(driverName string) (string, error) {
	return ocsql.Register(driverName, tracingOptions()...)
}

func tracingOptions() []ocsql.TraceOption {
	return []ocsql.TraceOption{
		ocsql.WithQuery(true),
		ocsql.WithRowsClose(true),
		ocsql.WithRowsAffected(true),
		ocsql.WithAllowRoot(true),
		ocsql.WithDisableErrSkip(true),
	}
}

// WithQueriesLogging instruments database connector with query logging.
func WithQueriesLogging(dbConnector driver.Connector, logger ctxd.Logger, statsTracker stats.Tracker) driver.Connector {
	return dbwrap.WrapConnector(dbConnector, wrapOptions(logger, statsTracker)...)
}

// DriverNameWithQueriesLogging registers database driver name with query logging.
func DriverNameWithQueriesLogging(driverName string, logger ctxd.Logger, statsTracker stats.Tracker) (string, error) {
	return dbwrap.Register(driverName, wrapOptions(logger, statsTracker)...)
}

func wrapOptions(logger ctxd.Logger, statsTracker stats.Tracker) []dbwrap.Option {
	if logger == nil {
		logger = ctxd.NoOpLogger{}
	}

	if statsTracker == nil {
		statsTracker = stats.NoOp{}
	}

	skipPackages := []string{
		"github.com/Masterminds/squirrel",
		"github.com/bool64/sqluct",
		"github.com/jmoiron/sqlx",
	}

	return []dbwrap.Option{
		// This interceptor enables reverse debugging from DB side.
		dbwrap.WithInterceptor(func(ctx context.Context, operation dbwrap.Operation, statement string, args []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
			// Closest caller in the stack with package not equal to listed and to "database/sql".
			caller := dbwrap.Caller(skipPackages...)

			// Add caller name as statement comment.
			return ctx, statement + " -- " + caller, args
		}),

		// This option limits middleware applicability.
		dbwrap.WithOperations(dbwrap.Query, dbwrap.StmtQuery, dbwrap.Exec, dbwrap.StmtExec),

		// This middleware logs statements with arguments at DEBUG level and counts stats.
		dbwrap.WithMiddleware(observe(logger, statsTracker, skipPackages)),
	}
}

func observe(logger ctxd.Logger, statsTracker stats.Tracker, skipPackages []string) dbwrap.Middleware {
	return func(
		ctx context.Context,
		operation dbwrap.Operation,
		statement string,
		args []driver.NamedValue,
	) (nCtx context.Context, onFinish func(error)) {
		// Closest caller in the stack with package not equal to listed and to "database/sql".
		caller := dbwrap.Caller(skipPackages...)

		ctx, span := trace.StartSpan(ctx, caller+":"+string(operation))
		span.AddAttributes(
			trace.StringAttribute("stmt", statement),
			trace.StringAttribute("args", fmt.Sprintf("%v", args)),
		)

		statsTracker.Add(ctx, "sql_storage_queries_total", 1, "method", caller)

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

			statsTracker.Add(ctx, "sql_storage_queries_seconds", time.Since(started).Seconds(),
				"method", caller)

			logger.Debug(ctx, caller+" "+string(operation)+res,
				"stmt", statement,
				"args", args,
				"elapsed", time.Since(started).String(),
				"err", err,
			)
		}
	}
}
