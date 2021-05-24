package database

import "time"

// Config describes database pool.
type Config struct {
	DSN             string        `envconfig:"DATABASE_DSN" required:"true"`
	MaxLifetime     time.Duration `split_words:"true" envconfig:"MAX_LIFETIME" default:"4h"`
	MaxIdle         int           `split_words:"true" envconfig:"MAX_IDLE" default:"5"`
	MaxOpen         int           `split_words:"true" envconfig:"MAX_OPEN" default:"5"`
	InitConn        bool          `split_words:"true" envconfig:"INIT_CONN"`
	ApplyMigrations bool          `split_words:"true" envconfig:"APPLY_MIGRATIONS"`
}
