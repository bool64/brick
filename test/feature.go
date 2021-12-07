package test

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	brick "github.com/acme-corp-tech/brick"
	"github.com/acme-corp-tech/brick/config"
	"github.com/bool64/dbdog"
	"github.com/bool64/godogx"
	"github.com/bool64/godogx/allure"
	"github.com/bool64/httpdog"
	"github.com/bool64/shared"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Context is a test context for feature tests.
type Context struct {
	Vars                *shared.Vars
	Local               *httpdog.Local
	External            *httpdog.External
	Database            *dbdog.Manager
	ScenarioInitializer func(s *godog.ScenarioContext)
}

func newContext(t *testing.T) *Context {
	t.Helper()

	vars := &shared.Vars{}

	tc := &Context{}
	tc.Local = httpdog.NewLocal("")
	tc.Local.JSONComparer.Vars = vars
	tc.Local.Client.OnBodyMismatch = func(data []byte) {
		assert.NoError(t, ioutil.WriteFile("_last_mismatch.json", data, 0o600))
	}

	tc.External = &httpdog.External{}
	tc.External.Vars = vars

	tc.Database = dbdog.NewManager()
	tc.Database.Vars = vars

	return tc
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

	tc.Local.SetBaseURL(addr)

	dbi := tc.Database.Instances[dbdog.DefaultDatabase]
	dbi.Storage = l.Storage
	tc.Database.Instances[dbdog.DefaultDatabase] = dbi

	godogx.RegisterPrettyFailedFormatter()

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
		Options: &godog.Options{
			Format:        "pretty-failed",
			Strict:        true,
			Paths:         []string{"features"},
			Tags:          os.Getenv("GODOG_TAGS"),
			StopOnFailure: os.Getenv("GODOG_STOP_ON_FAILURE") == "1",
			TestingT:      t,
		},
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
