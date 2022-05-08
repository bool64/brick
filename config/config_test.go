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

	defer func() {
		require.NoError(t, os.Chdir(".."))
	}()

	cfg := struct {
		Foo string
		Bar int `minimum:"500" required:"true"`
		Baz bool
	}{}

	require.NoError(t, os.Setenv("TEST_ENVIRONMENT", "test"))
	assert.EqualError(t, config.Load("TEST", &cfg), "validate: I[#/Bar] S[#/properties/Bar/minimum] must be >= 500/1 but found 321")
	require.NoError(t, os.Setenv("TEST_BAR", "600"))
	assert.NoError(t, config.Load("TEST", &cfg))

	assert.Equal(t, 600, cfg.Bar)
	assert.Equal(t, "foo_template", cfg.Foo)
}
