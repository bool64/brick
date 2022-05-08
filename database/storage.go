package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/jmoiron/sqlx"
	"github.com/vearutop/gooselite"
	"github.com/vearutop/gooselite/iofs"
)

// SetupStorage initializes database pool and prepares storage.
func SetupStorage(cfg Config, logger ctxd.Logger, statsTracker stats.Tracker, driverName string, conn driver.Connector, migrations fs.FS) (*sqluct.Storage, error) {
	conn = WithTracing(conn)
	conn = WithQueriesLogging(conn, logger, statsTracker)

	db := sql.OpenDB(conn)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	st := sqluct.NewStorage(sqlx.NewDb(sql.OpenDB(conn), driverName))

	switch driverName {
	case "mysql", "sqlite", "sqlite3":
		st.Format = squirrel.Question
		st.IdentifierQuoter = sqluct.QuoteBackticks
	case "postgres":
		st.Format = squirrel.Dollar
		st.IdentifierQuoter = sqluct.QuoteANSI
	}

	if cfg.InitConn {
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
	}

	if migrations == nil || !cfg.ApplyMigrations {
		return st, nil
	}

	gooselite.SetLogger(gooseLogger{c: context.Background(), l: logger})

	if err := gooselite.SetDialect(driverName); err != nil {
		return nil, err
	}

	// Apply migrations.
	if err := iofs.Up(db, migrations, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to run up migrations: %w", err)
	}

	return st, nil
}

// GooseLogger adapts contextualized logger for goose.
type gooseLogger struct {
	c context.Context // nolint:containedctx // Implemented interface is not contextualized, so ctx is contained.
	l ctxd.Logger
}

func (l gooseLogger) Fatal(v ...interface{}) { l.l.Error(l.c, fmt.Sprint(v...)); os.Exit(1) }
func (l gooseLogger) Fatalf(f string, v ...interface{}) {
	l.l.Error(l.c, fmt.Sprintf(f, v...))
	os.Exit(1)
}

func (l gooseLogger) Print(v ...interface{}) {
	l.l.Info(l.c, strings.TrimRight(fmt.Sprint(v...), "\n"))
}
func (l gooseLogger) Println(v ...interface{}) { l.l.Info(l.c, fmt.Sprint(v...)) }
func (l gooseLogger) Printf(f string, v ...interface{}) {
	l.l.Info(l.c, strings.TrimRight(fmt.Sprintf(f, v...), "\n"))
}
