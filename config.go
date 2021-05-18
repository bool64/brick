package brick

import (
	"time"

	"github.com/bool64/brick/debug"
	"github.com/bool64/zapctxd"
)

// BaseConfig is a basic application agnostic service configuration that manages common infrastructure.
type BaseConfig struct {
	Log zapctxd.Config

	// Environment is the name of environment where application runs.
	Environment string `default:"dev"`

	// ServiceName is the name of the service to use in documentation and tracing.
	ServiceName string `split_words:"true"`

	// ShutdownTimeout limits time for graceful shutdown of an application.
	ShutdownTimeout time.Duration `default:"10s" split_words:"true"`

	// Debug controls dev tools.
	Debug debug.Config `split_words:"true"`
}
