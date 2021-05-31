package database

import "time"

// Config describes database pool.
type Config struct {
	DSN             string        `required:"true"`
	MaxLifetime     time.Duration `split_words:"true" default:"4h"`
	MaxIdle         int           `split_words:"true" default:"5"`
	MaxOpen         int           `split_words:"true" default:"5"`
	InitConn        bool          `split_words:"true"`
	ApplyMigrations bool          `split_words:"true"`
}
