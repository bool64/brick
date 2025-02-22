package database

import "time"

// Config describes database pool.
type Config struct {
	DriverName      string        `split_words:"true"`
	DSN             string        `required:"true"`
	MaxLifetime     time.Duration `split_words:"true" default:"4h"`
	MaxIdle         int           `split_words:"true" default:"5"`
	MaxOpen         int           `split_words:"true" default:"5"`
	InitConn        bool          `split_words:"true"`
	ApplyMigrations bool          `split_words:"true"`

	// MethodSkipPackages provides helper package paths to skip when identifying method name for observability.
	// Item example: "github.com/jmoiron/sqlx".
	MethodSkipPackages []string
}
