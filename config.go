package brick

import (
	"time"

	"github.com/bool64/brick/debug"
	"github.com/bool64/zapctxd"
)

// BaseConfig is a basic application agnostic service configuration that manages common infrastructure.
type BaseConfig struct {
	// Initialized indicates zero/uninitialized value of the configuration.
	Initialized bool `default:"true"`

	Log zapctxd.Config `split_words:"true"`

	// Environment is the name of environment where application runs.
	Environment string `default:"dev"`

	// ServiceName is the name of the service to use in documentation and tracing.
	ServiceName string `split_words:"true"`

	// HTTPListenAddr is the address of HTTP server listener.
	HTTPListenAddr string `split_words:"true" default:":80"`

	// ShutdownTimeout limits time for graceful shutdown of an application.
	ShutdownTimeout time.Duration `split_words:"true" default:"10s"`

	// Debug controls dev tools.
	Debug debug.Config `split_words:"true"`
}

// WithBaseConfig is an embedded config accessor.
type WithBaseConfig interface {
	Base() BaseConfig
}

// Base exposes base config.
func (c *BaseConfig) Base() BaseConfig {
	return *c
}
