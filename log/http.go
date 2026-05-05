package log

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/rest"
	"go.opentelemetry.io/otel/trace"
)

// HTTPTraceTransaction adds trace transaction info to request context.
func HTTPTraceTransaction(fields ctxd.FieldNames) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			if span != nil {
				sc := span.SpanContext()
				if !sc.IsValid() {
					h.ServeHTTP(w, r)

					return
				}

				ctx := ctxd.AddFields(r.Context(),
					fields.TraceID, sc.TraceID().String(),
					fields.TransactionID, sc.SpanID().String(),
				)
				r = r.WithContext(ctx)
			}

			h.ServeHTTP(w, r)
		})
	}
}

func panicResponse(rw http.ResponseWriter, resp rest.ErrResponse) error {
	if j, err := json.Marshal(resp); err == nil {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusInternalServerError)
		_, err = rw.Write(append(j, '\n'))

		return err
	}

	rw.WriteHeader(http.StatusInternalServerError)
	_, err := rw.Write([]byte(`request panicked` + "\n"))

	return err
}

// HTTPRecover logs http request and response details and recovers from panics.
type HTTPRecover struct {
	Logger      ctxd.Logger
	FieldNames  ctxd.FieldNames
	PrintPanic  bool
	ExposePanic bool
	OnPanic     []func(ctx context.Context, rcv any, stack []byte)
}

func (mw HTTPRecover) handlePanic(ctx context.Context, rvr any, msg string) {
	if !mw.PrintPanic {
		mw.Logger.Error(ctx, msg,
			"panic", rvr,
			"stack", strings.Split(string(debug.Stack()), "\n"),
		)
	} else {
		middleware.PrintPrettyStack(rvr)
	}
}

func (mw HTTPRecover) processPanic(ctx context.Context, rvr any, rw http.ResponseWriter) {
	mw.handlePanic(ctx, rvr, "request panicked")

	resp := rest.ErrResponse{ErrorText: "request panicked"}

	var stack []byte

	if mw.ExposePanic {
		stack = debug.Stack()

		resp.Context = map[string]any{
			"panic": rvr, "stack": strings.Split(string(stack), "\n"),
		}
	}

	if err := panicResponse(rw, resp); err != nil {
		mw.Logger.Error(ctx, "failed to write panic response", "error", err)
	}

	if len(mw.OnPanic) == 0 {
		return
	}

	defer func() {
		if rcv := recover(); rcv != nil {
			mw.handlePanic(ctx, rcv, "panic while handling panic %C")
		}
	}()

	if stack == nil {
		stack = debug.Stack()
	}

	for _, onPanic := range mw.OnPanic {
		onPanic(ctx, rvr, stack)
	}
}

// Middleware wraps http handler.
func (mw HTTPRecover) Middleware() func(handler http.Handler) http.Handler {
	logger := mw.Logger

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			defer func() {
				rvr := recover()

				if rvr == nil {
					return
				}

				if err, ok := rvr.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					return
				}

				mw.processPanic(ctx, rvr, rw)
			}()

			fields := mw.FieldNames

			ctx = ctxd.AddFields(ctx,
				fields.ClientIP, r.RemoteAddr,
				fields.UserAgentOriginal, r.UserAgent(),
				fields.URL, r.URL.String(),
				fields.HTTPMethod, r.Method,
			)

			logger.Debug(ctx, "http request started", "headers", headersMap(r.Header))

			w := middleware.NewWrapResponseWriter(rw, r.ProtoMajor)
			start := time.Now()

			next.ServeHTTP(w, r)

			elapsed := time.Since(start)

			ctx = ctxd.AddFields(ctx,
				fields.HTTPResponseStatus, w.Status(),
				fields.HTTPResponseBytes, w.BytesWritten(),
				"elapsed", elapsed.String(),
				"elapsed_ms", float64(elapsed.Nanoseconds())/1000000.0,
			)

			logger.Debug(ctx, "http request complete", "resp_headers", headersMap(w.Header()))
		})
	}
}

func headersMap(header http.Header) ctxd.DeferredJSON {
	return func() any {
		headers := make(map[string]string, len(header))

		for k := range header {
			v := header.Get(k)
			if k == "Authorization" || k == "Cookie" || k == "Set-Cookie" {
				v = "[redacted]"
			}

			headers[k] = v
		}

		return headers
	}
}
