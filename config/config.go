// Package config provides configuration loader based on env vars.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// WithEnvFiles populates env vars from provided files.
//
// It returns an error if file does not exist.
func WithEnvFiles(files ...string) func() error {
	return func() error { return godotenv.Load(files...) }
}

// WithOptionalEnvFiles populates env vars from provided files that exist.
//
// Non-existent files are ignored.
func WithOptionalEnvFiles(files ...string) func() error {
	var found []string

	for _, f := range files {
		if fileExists(f) {
			found = append(found, f)
		}
	}

	if len(found) == 0 {
		return func() error { return nil }
	}

	return WithEnvFiles(found...)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

// Load loads config from ENV vars, sources are called to populate ENV vars in advance.
//
// In no sources are provided then vars from .env.template, .env, .env.<ENVIRONMENT>
// files are loaded if available. Use nil or any other source to avoid that.
func Load(prefix string, spec interface{}, sources ...func() error) error {
	if len(sources) == 0 {
		sources = append(sources, WithOptionalEnvFiles(".env"))

		env := struct {
			// Environment is the name of environment where application runs.
			Environment string
		}{}
		envconfig.MustProcess(prefix, &env)

		if env.Environment != "" {
			sources = append(sources, WithOptionalEnvFiles(".env."+env.Environment))
		}

		sources = append(sources, WithOptionalEnvFiles(".env.template"))
	}

	for _, o := range sources {
		if o == nil {
			continue
		}

		if err := o(); err != nil {
			return fmt.Errorf("failed to apply config source: %w", err)
		}
	}

	return envconfig.Process(prefix, spec)
}
