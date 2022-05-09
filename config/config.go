// Package config provides configuration loader based on env vars.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	jsval "github.com/santhosh-tekuri/jsonschema/v3"
	"github.com/swaggest/jsonschema-go"
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

// DefaultLoaders populate ENV vars from .env files.
func DefaultLoaders(prefix string) []func() error {
	loaders := []func() error{WithOptionalEnvFiles(".env")}

	env := struct {
		// Environment is the name of environment where application runs.
		Environment string
	}{}
	envconfig.MustProcess(prefix, &env)

	if env.Environment != "" {
		loaders = append(loaders, WithOptionalEnvFiles(".env."+env.Environment))
	}

	loaders = append(loaders, WithOptionalEnvFiles(".env.template"))

	return loaders
}

// Load loads config from ENV vars, loaders are called to populate ENV vars in advance.
//
// In no loaders are provided then vars from .env.template, .env, .env.<ENVIRONMENT>
// files are loaded if available. Use nil or any other source to avoid that.
func Load(prefix string, spec interface{}, loaders ...func() error) error {
	if len(loaders) == 0 {
		loaders = DefaultLoaders(prefix)
	}

	for _, o := range loaders {
		if o == nil {
			continue
		}

		if err := o(); err != nil {
			return fmt.Errorf("failed to apply config source: %w", err)
		}
	}

	err := envconfig.Process(prefix, spec)
	if err != nil {
		return err
	}

	return validate(spec)
}

func validate(spec interface{}) error {
	specj, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	schema, err := (&jsonschema.Reflector{}).Reflect(spec, jsonschema.ProcessWithoutTags, func(rc *jsonschema.ReflectContext) {
		rc.SkipNonConstraints = true
	})
	if err != nil {
		return fmt.Errorf("reflect schema: %w", err)
	}

	js, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	compiler := jsval.NewCompiler()

	err = compiler.AddResource("schema.json", bytes.NewReader(js))
	if err != nil {
		return fmt.Errorf("validator add schema: %w", err)
	}

	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("validator compile schema: %w", err)
	}

	err = sch.Validate(bytes.NewReader(specj))
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	return nil
}
