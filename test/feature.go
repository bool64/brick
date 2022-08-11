package test

import (
	"flag"
	"net/http"
	"os"
	"testing"

	"github.com/bool64/brick"
	"github.com/bool64/brick/config"
	"github.com/bool64/godogx"
	"github.com/bool64/httpmock"
	"github.com/bool64/shared"
	"github.com/cucumber/godog"
	"github.com/godogx/allure"
	"github.com/godogx/dbsteps"
	"github.com/godogx/httpsteps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Context is a test context for feature tests.
type Context struct {
	Vars                *shared.Vars
	Local               *httpsteps.LocalClient
	External            *httpsteps.ExternalServer
	Database            *dbsteps.Manager
	ScenarioInitializer func(s *godog.ScenarioContext)
	OptionsInitializer  func(options *godog.Options)
	Concurrency         int
}

func newContext(t *testing.T) *Context {
	t.Helper()

	vars := &shared.Vars{}

	tc := &Context{}
	tc.Local = httpsteps.NewLocalClient("", func(client *httpmock.Client) {
		client.OnBodyMismatch = func(data []byte) {
			assert.NoError(t, os.WriteFile("_last_mismatch.json", data, 0o600))
		}
	})
	tc.Local.Vars = vars

	tc.External = httpsteps.NewExternalServer()
	tc.External.Vars = vars

	tc.Database = dbsteps.NewManager()
	tc.Database.Vars = vars

	return tc
}

var (
	feature      = flag.String("feature", "features", "Feature file to test.")
	godogOptions = godog.Options{
		Format:        "pretty-failed",
		Strict:        true,
		Tags:          os.Getenv("GODOG_TAGS"),
		StopOnFailure: os.Getenv("GODOG_STOP_ON_FAILURE") == "1",
	}
)

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &godogOptions)
}

// RunFeatures runs feature tests.
func RunFeatures(t *testing.T, envPrefix string, cfg brick.WithBaseConfig, init func(tc *Context) (*brick.BaseLocator, http.Handler)) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require.NoError(t, config.Load(envPrefix, cfg, config.WithOptionalEnvFiles(".env.integration-test")))

	tc := newContext(t)
	l, router := init(tc)

	addr, err := l.StartHTTPServer(router)
	require.NoError(t, err)

	require.NoError(t, tc.Local.SetBaseURL(addr, httpsteps.Default))

	dbi := tc.Database.Instances[dbsteps.Default]
	dbi.Storage = l.Storage
	tc.Database.Instances[dbsteps.Default] = dbi

	godogx.RegisterPrettyFailedFormatter()

	options := godogOptions

	options.TestingT = t
	if options.Concurrency == 0 {
		options.Concurrency = tc.Concurrency
	}

	if len(options.Paths) == 0 {
		options.Paths = []string{*feature}
	}

	if tc.OptionsInitializer != nil {
		tc.OptionsInitializer(&options)
	}

	suite := godog.TestSuite{
		Name: cfg.Base().ServiceName + "-integration-test",
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			tc.Local.RegisterSteps(s)
			tc.External.RegisterSteps(s)
			tc.Database.RegisterSteps(s)

			if tc.ScenarioInitializer != nil {
				tc.ScenarioInitializer(s)
			}
		},
		Options: &options,
	}

	if os.Getenv("GODOG_ALLURE") != "" {
		allure.RegisterFormatter()

		suite.Options.Format += ",allure"
	}

	assert.Equal(t, 0, suite.Run(), "non-zero status returned, failed to run feature tests")

	// An instance can keep on running if developer would like to use or debug it after tests have finished.
	if os.Getenv("GODOG_KEEP_INSTANCE") == "1" {
		println("tests passed, keeping instance, kill it manually at will")
	} else {
		l.Shutdown()
	}

	<-l.Wait()
}
