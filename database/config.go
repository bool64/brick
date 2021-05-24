package database

import "time"

// Config describes database pool.
type Config struct {
	DSN         string        `envconfig:"DATABASE_DSN" required:"true"`
	MaxLifetime time.Duration `envconfig:"MAX_LIFETIME" default:"4h"`
	MaxIdle     int           `envconfig:"MAX_IDLE" default:"5"`
	MaxOpen     int           `envconfig:"MAX_OPEN" default:"5"`
	InitConn    bool          `envconfig:"INIT_CONN"`
}
