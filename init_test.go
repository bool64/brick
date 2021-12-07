package brick_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	brick "github.com/acme-corp-tech/brick"
	"github.com/acme-corp-tech/brick/config"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
	"go.uber.org/zap"
)

func TestNewBaseLocator(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/api/features" { // Laika init.
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("[]"))
			assert.NoError(t, err)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	cfg := brick.BaseConfig{}
	require.NoError(t, config.Load("TEST", &cfg))

	log := bytes.NewBuffer(nil)

	cfg.Environment = "live"
	cfg.Log.Level = zap.InfoLevel
	cfg.Log.Output = log
	cfg.Debug.TraceSamplingProbability = 1.0

	cfg.ServiceName = "test"

	u.User = url.UserPassword("foo", "")
	u.Path = "123123"

	l, err := brick.NewBaseLocator(cfg)
	require.NoError(t, err)
	require.NotNil(t, l)

	r := brick.NewBaseRouter(l)
	serviceURLPrefix := "/" + cfg.ServiceName

	uc := usecase.NewIOI(nil, nil, func(ctx context.Context, input, output interface{}) error {
		panic("oops")
	})

	r.Route(serviceURLPrefix, func(r chi.Router) {
		r.Method(http.MethodGet, "/something-public", nethttp.NewHandler(uc))
	})

	rw := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, serviceURLPrefix+"/something-public", nil)
	require.NoError(t, err)
	req.Header.Set("X-Foo", "bar")
	r.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
	assert.Equal(t, "application/json; charset=utf-8", rw.Header().Get("Content-Type"))
	assert.Equal(t, `{"error":"request panicked"}`+"\n", rw.Body.String())

	logs := "[" + strings.ReplaceAll(log.String(), "\n", ",\n") + "{}]"
	assertjson.Equal(t, []byte(`[{"level":"info","@timestamp":"<ignore-diff>","message":"http request started","client.ip":"","user_agent.original":"","url.original":"/test/something-public","http.request.method":"GET"},
        	            	{"level":"error","@timestamp":"<ignore-diff>","message":"request panicked","panic":"oops","stack":"<ignore-diff>","client.ip":"","user_agent.original":"","url.original":"/test/something-public","http.request.method":"GET"},
        	            	{}]`), []byte(logs), logs)

	l.Shutdown()

	assert.NoError(t, <-l.Wait())
}
