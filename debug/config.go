package debug

import "context"

// Config keeps debug settings.
type Config struct {
	// TraceSamplingProbability is probability of exporting of OpenCensus trace.
	TraceSamplingProbability float64 `split_words:"true" default:"0.1"`

	// TraceURL allows providing URL to {trace_id}, example http://jaeger.myservice.com/trace/{trace_id}.
	TraceURL string `split_words:"true"`

	// DevTools enables developer tools for documentation and debug.
	DevTools bool `split_words:"true" default:"true"`

	// RouterConfig allows control of developer tools router.
	RouterConfig []func(options *RouterConfig) `ignored:"true"`

	// DevPassword enables password protection for dev tools.
	DevPassword string `split_words:"true"`

	// URL used as an entry point to mount dev tools debug router.
	URL string `split_words:"true" default:"/debug"`

	// ExposePanic allows showing panic messages and traces in API response,
	// can be useful for non-production environments.
	ExposePanic bool `split_words:"true"`

	OnPanic []func(ctx context.Context, rcv interface{}, stack []byte) `ignored:"true"`
}
