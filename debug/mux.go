package debug

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type page struct {
	link, title string
}

// Mux serves debug tools.
type Mux struct {
	*chi.Mux
	Prefix string
	pages  []page
	body   []byte
}

// NewMux creates a new router for debug tools.
func NewMux(prefix string) *Mux {
	debugRouter, ok := middleware.Profiler().(*chi.Mux)
	if !ok {
		panic("BUG: failed to assert middleware.Profiler().(*chi.Mux)")
	}

	r := &Mux{Prefix: prefix}
	r.Mux = debugRouter

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf8")
		_, err := w.Write(r.body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.AddLink("pprof", "Profiling")

	return r
}

func (r *Mux) buildIndex() {
	r.body = []byte(`<!DOCTYPE html><html><head><title>Debug Tools</title><base href="` + r.Prefix + `/" /></head><h2>Debug Tools</h2><ul>`)

	for _, p := range r.pages {
		r.body = append(r.body, []byte(`<li><a href="`+p.link+`">`+p.title+`</a></li>`)...)
	}

	r.body = append(r.body, []byte(`</ul></html>`)...)
}

// AddLink adds a link to the index page of debug tools.
func (r *Mux) AddLink(link, title string) {
	r.pages = append(r.pages, page{link: link, title: title})
	r.buildIndex()
}
