package brick

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/bool64/brick/config"
	"github.com/bool64/dbdog"
	"github.com/bool64/httpdog"
	"github.com/bool64/shared"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext is a test context for feature tests.
type TestContext struct {
	Vars                *shared.Vars
	Local               *httpdog.Local
	External            *httpdog.External
	Database            *dbdog.Manager
	ScenarioInitializer func(s *godog.ScenarioContext)
}

func newTestContext(t *testing.T) *TestContext {
	t.Helper()

	vars := &shared.Vars{}

	tc := &TestContext{}
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

// RunTests runs feature tests.
func RunTests(t *testing.T, envPrefix string, cfg WithBaseConfig, init func(tc *TestContext) (*BaseLocator, http.Handler)) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require.NoError(t, config.Load(envPrefix, cfg, config.WithOptionalEnvFiles(".env.integration-test")))

	tc := newTestContext(t)
	l, router := init(tc)

	addr, err := l.StartHTTPServer(router)
	require.NoError(t, err)

	tc.Local.SetBaseURL(addr)

	dbi := tc.Database.Instances[dbdog.DefaultDatabase]
	dbi.Storage = l.Storage
	tc.Database.Instances[dbdog.DefaultDatabase] = dbi

	output := bytes.NewBuffer(nil)
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			output.Reset()
			s.AfterScenario(func(sc *godog.Scenario, err error) {
				if err != nil {
					t.Helper()
					t.Run(sc.GetName(), func(t *testing.T) {
						t.Fatal(output.String())
					})
				}
			})

			tc.Local.RegisterSteps(s)
			tc.External.RegisterSteps(s)
			tc.Database.RegisterSteps(s)

			if tc.ScenarioInitializer != nil {
				tc.ScenarioInitializer(s)
			}
		},
		Options: &godog.Options{
			Format:        "pretty",
			Output:        output,
			Strict:        true,
			Concurrency:   1,
			Paths:         []string{"features"},
			Tags:          os.Getenv("GODOG_TAGS"),
			StopOnFailure: os.Getenv("GODOG_STOP_ON_FAILURE") == "1",
		},
	}

	suite.Run()

	l.Shutdown()
	<-l.Wait()
}
