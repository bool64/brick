package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/bool64/ctxd"
	"github.com/bool64/dbwrap"
	"github.com/bool64/stats"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// driverNameWithTracing registers database driver name with OpenTelemetry instrumentation.
func driverNameWithTracing(driverName string) (string, error) {
	return otelsql.Register(driverName, tracingOptions()...)
}

func tracingOptions() []otelsql.Option {
	return []otelsql.Option{
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
		otelsql.WithMeterProvider(otel.GetMeterProvider()),
		otelsql.WithDisableSkipErrMeasurement(true),
	}
}

// withQueriesLogging instruments database connector with query logging.
func withQueriesLogging(cfg Config, dbConnector driver.Connector, logger ctxd.Logger, statsTracker stats.Tracker) driver.Connector {
	return dbwrap.WrapConnector(dbConnector, wrapOptions(logger, statsTracker, cfg.MethodSkipPackages)...)
}

// driverNameWithQueriesLogging registers database driver name with query logging.
func driverNameWithQueriesLogging(cfg Config, driverName string, logger ctxd.Logger, statsTracker stats.Tracker) (string, error) {
	return dbwrap.Register(driverName, wrapOptions(logger, statsTracker, cfg.MethodSkipPackages)...)
}

func wrapOptions(logger ctxd.Logger, statsTracker stats.Tracker, skipPackages []string) []dbwrap.Option {
	if logger == nil {
		logger = ctxd.NoOpLogger{}
	}

	if statsTracker == nil {
		statsTracker = stats.NoOp{}
	}

	skipPackages = append([]string{
		"github.com/Masterminds/squirrel",
		"github.com/bool64/sqluct",
		"github.com/jmoiron/sqlx",
	}, skipPackages...)

	return []dbwrap.Option{
		// This interceptor enables reverse debugging from DB side.
		dbwrap.WithInterceptor(func(ctx context.Context, _ dbwrap.Operation, statement string, args []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
			// Closest caller in the stack with package not equal to listed and to "database/sql".
			caller := dbwrap.Caller(skipPackages...)

			// Add caller name as statement comment.
			return ctx, statement + " -- " + caller, args
		}),

		// This option limits middleware applicability.
		dbwrap.WithOperations(dbwrap.Query, dbwrap.StmtQuery, dbwrap.Exec, dbwrap.StmtExec, dbwrap.RowsClose),

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
		caller := dbwrap.CallerCtx(ctx, skipPackages...)

		if operation == dbwrap.RowsClose {
			statsTracker.Add(ctx, "sql_storage_rows_close", 1, "method", caller)

			return nil, nil
		}

		ctx, span := otel.Tracer("github.com/bool64/brick/database").Start(ctx, caller+":"+string(operation))
		span.SetAttributes(
			attribute.String("db.statement", statement),
			attribute.String("db.args", fmt.Sprintf("%v", args)),
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
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)

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
