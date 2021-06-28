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

// Start loads config and runs application with provided service locator and http router.
func Start(envPrefix string, cfg WithBaseConfig, init func(docsMode bool) (*BaseLocator, http.Handler)) {
	ver := flag.Bool("version", false, "Print application version and exit.")
	docs := flag.Bool("openapi", false, "Print application OpenAPI spec and exit.")
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

	if err := config.Load(envPrefix, cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	loc, router := init(false)

	_, err := loc.StartHTTPServer(router)
	if err != nil {
		loc.CtxdLogger().Error(context.Background(), "failed to start http server: %v", "error", err)
		os.Exit(1)
	}

	// Wait for service loc termination finished.
	err = <-loc.Wait()
	if err != nil {
		loc.CtxdLogger().Error(context.Background(), err.Error())
	}
}
