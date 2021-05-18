package config_test

import (
	"os"
	"testing"

	"github.com/bool64/brick/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	require.NoError(t, os.Chdir("./__testdata"))

	cfg := struct {
		Foo string
		Bar int  `required:"true"`
		Baz bool `required:"true"`
	}{}

	require.NoError(t, os.Setenv("TEST_ENVIRONMENT", "test"))
	assert.EqualError(t, config.Load("TEST", &cfg), "required key TEST_BAZ missing value")
	require.NoError(t, os.Setenv("TEST_BAZ", "1"))
	assert.NoError(t, config.Load("TEST", &cfg))

	assert.Equal(t, 321, cfg.Bar)
	assert.Equal(t, "foo_template", cfg.Foo)
}
