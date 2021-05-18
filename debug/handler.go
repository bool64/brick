package debug

import (
	"net/http"
	"sort"

	"github.com/bool64/brick/debug/zpages"
	"github.com/bool64/dev/version"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterConfig defines debug tools router options.
type RouterConfig struct {
	// Prefix sets mounting path for debug index page, default "/debug".
	Prefix string

	// Links is an additional map of href to title to show in debug index.
	Links map[string]string

	// Routes are executed on debug router.
	Routes []func(r *chi.Mux)

	// TraceToURL allows converting sampled trace ids into clickable links using function result as URL.
	// URL may point to Jaeger UI for example.
	TraceToURL func(traceID string) string
}

// Links adds a map of href to title to show in debug index.
func Links(links map[string]string) func(config *RouterConfig) {
	return func(config *RouterConfig) {
		config.Links = links
	}
}

// Prefix is a DebugHandler option.
func Prefix(prefix string) func(config *RouterConfig) {
	return func(config *RouterConfig) {
		config.Prefix = prefix
	}
}

// Index is a http handler that provides a page with debug tools.
func Index(cfg RouterConfig) *chi.Mux {
	debugRouter, ok := middleware.Profiler().(*chi.Mux)
	if !ok {
		panic("BUG: failed to assert middleware.Profiler().(*chi.Mux)")
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "/debug"
	}

	type page struct {
		link, title string
	}

	pages := []page{
		{link: "pprof", title: "Profiling"},
		{link: "zpages/tracez", title: "Trace Spans"},
		{link: "version", title: "Version"},
	}

	body := []byte(`<!DOCTYPE html><html><head><title>Debug Tools</title><base href="` + cfg.Prefix + `/" /></head><h2>Debug Tools</h2><ul>`)

	debugRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf8")
		_, err := w.Write(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	debugRouter.Get("/version", version.Handler)
	debugRouter.Mount("/zpages", zpages.Mux(cfg.Prefix+"/zpages", cfg.TraceToURL))

	for l, t := range cfg.Links {
		pages = append(pages, page{link: l, title: t})
	}

	for _, r := range cfg.Routes {
		r(debugRouter)
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].title < pages[j].title
	})

	for _, p := range pages {
		body = append(body, []byte(`<li><a href="`+p.link+`">`+p.title+`</a></li>`)...)
	}

	body = append(body, []byte(`</ul></html>`)...)

	return debugRouter
}
