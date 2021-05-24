package database

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/fs"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/jmoiron/sqlx"
	"github.com/vearutop/gooselite"
	"github.com/vearutop/gooselite/iofs"
)

// SetupStorage initializes database pool and prepares storage.
func SetupStorage(cfg Config, logger ctxd.Logger, driverName string, conn driver.Connector, migrations fs.FS) (*sqluct.Storage, error) {
	conn = WithTracing(conn)
	conn = WithQueriesLogging(conn, logger)

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

	if err := gooselite.SetDialect(driverName); err != nil {
		return nil, err
	}

	// Apply migrations.
	if err := iofs.Up(db, migrations, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to run up migrations: %w", err)
	}

	return st, nil
}
