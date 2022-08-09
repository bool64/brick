package opencensus

import (
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Views lists OpenCensus views rendered as metrics.
func Views() []*view.View { //nolint:funlen
	return []*view.View{
		{
			Name:        "http/server/request_count",
			Description: "Count of HTTP requests started",
			Measure:     ochttp.ServerRequestCount,
			Aggregation: view.Count(),
		},
		{
			Name:        "http/server/request_bytes",
			Description: "Size distribution of HTTP request body",
			Measure:     ochttp.ServerRequestBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
		},
		{
			Name:        "http/server/response_bytes",
			Description: "Size distribution of HTTP response body",
			Measure:     ochttp.ServerResponseBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
		},
		{
			Name:        "http/server/latency",
			Description: "Latency distribution of HTTP requests",
			Measure:     ochttp.ServerLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
		},
		{
			Name:        "http/server/request_count_by_method",
			Description: "Server request count by HTTP method",
			TagKeys:     []tag.Key{ochttp.Method},
			Measure:     ochttp.ServerRequestCount,
			Aggregation: view.Count(),
		},
		{
			Name:        "http/server/response_count_by_status_code",
			Description: "Server response count by status code",
			TagKeys:     []tag.Key{ochttp.StatusCode},
			Measure:     ochttp.ServerLatency,
			Aggregation: view.Count(),
		},
		{
			Name:        "http/server/latency_by_path",
			Description: "Latency distribution of HTTP requests by route",
			TagKeys:     []tag.Key{ochttp.KeyServerRoute},
			Measure:     ochttp.ServerLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
		},
		{
			Name:        "http/client/completed_count",
			Measure:     ochttp.ClientRoundtripLatency,
			Aggregation: view.Count(),
			Description: "Count of completed requests, by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientHost, ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
		{
			Name:        "http/client/sent_bytes",
			Measure:     ochttp.ClientSentBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
			Description: "Total bytes sent in request body (not including headers), by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientHost, ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
		{
			Name:        "http/client/received_bytes",
			Measure:     ochttp.ClientReceivedBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
			Description: "Total bytes received in response bodies (not including headers but including error responses with bodies), by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientHost, ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
		{
			Name:        "http/client/latency_by_host",
			TagKeys:     []tag.Key{ochttp.KeyClientHost},
			Measure:     ochttp.ClientRoundtripLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
		},
	}
}
