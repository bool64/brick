package brick

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bool64/brick/config"
	"github.com/bool64/dev/version"
	"github.com/swaggest/assertjson"
)

// StartOptions allows more control on application startup.
type StartOptions struct {
	// EnvPrefix is the prefix for environment variables.
	EnvPrefix string

	// EnvPrepare is called after all config loaders and before envconfig.Process,
	// it can be used to override defaults via env vars.
	EnvPrepare func() error

	// OnHTTPStart is called after the HTTP server is started.
	OnHTTPStart func(addr string)
}

// Start loads config and runs application with provided service locator and http router.
func Start(cfg WithBaseConfig, init func(docsMode bool) (*BaseLocator, http.Handler), options ...func(o *StartOptions)) {
	ver := flag.Bool("version", false, "Print application version and exit.")
	docs := flag.Bool("openapi", false, "Print application OpenAPI spec and exit.")
	confFile := flag.String("conf", "", "Config file with ENV variables to load.")
	flag.Parse()

	if ver != nil && *ver {
		fmt.Println(version.Info().Version)

		return
	}

	if docs != nil && *docs {
		loc, _ := init(true)

		j, err := assertjson.MarshalIndentCompact(loc.OpenAPI.Reflector().Spec, "", " ", 100)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(j))

		return
	}

	opt := StartOptions{}
	for _, o := range options {
		o(&opt)
	}

	loadConfig(opt.EnvPrefix, confFile, cfg, opt.EnvPrepare)

	loc, router := init(false)

	addr, err := loc.StartHTTPServer(router)
	if err != nil {
		loc.CtxdLogger().Error(context.Background(), "failed to start http server: %v", "error", err)
		os.Exit(1)
	}

	if opt.OnHTTPStart != nil {
		opt.OnHTTPStart(addr)
	}

	// Wait for service loc termination finished.
	err = <-loc.Wait()
	if err != nil {
		loc.CtxdLogger().Error(context.Background(), err.Error())
	}
}

func loadConfig(envPrefix string, conf *string, cfg WithBaseConfig, envPrepare func() error) {
	var cfgLoaders []func() error
	if conf != nil && *conf != "" && *conf != ".env" {
		cfgLoaders = append(cfgLoaders, config.WithEnvFiles(*conf))
	} else {
		cfgLoaders = config.DefaultLoaders(envPrefix)
	}

	cfgLoaders = append(cfgLoaders, envPrepare)

	if err := config.Load(envPrefix, cfg, cfgLoaders...); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
}
