package telemetry

import (
	"encoding/base64"
	"strings"
	"time"
)

// Config configures OpenTelemetry export.
type Config struct {
	// BaseURL is the OTLP gateway base URL.
	// For Grafana Cloud this is typically https://otlp-gateway-<region>.grafana.net/otlp.
	BaseURL string `split_words:"true"`

	// TracesURL overrides traces export URL.
	TracesURL string `split_words:"true"`

	// MetricsURL overrides metrics export URL.
	MetricsURL string `split_words:"true"`

	// LogsURL overrides logs export URL.
	LogsURL string `split_words:"true"`

	// Username is used to build Authorization header when Authorization is empty.
	Username string `split_words:"true"`

	// Password is used to build Authorization header when Authorization is empty.
	Password string `split_words:"true"`

	// Authorization sets the Authorization header directly.
	Authorization string `split_words:"true"`

	// MetricsInterval configures periodic OTLP metrics export interval.
	MetricsInterval time.Duration `split_words:"true" default:"15s"`
}

func (c Config) TraceEndpoint() string {
	if c.TracesURL != "" {
		return c.TracesURL
	}

	if c.BaseURL == "" {
		return ""
	}

	return strings.TrimRight(c.BaseURL, "/") + "/v1/traces"
}

func (c Config) MetricEndpoint() string {
	if c.MetricsURL != "" {
		return c.MetricsURL
	}

	if c.BaseURL == "" {
		return ""
	}

	return strings.TrimRight(c.BaseURL, "/") + "/v1/metrics"
}

func (c Config) LogEndpoint() string {
	if c.LogsURL != "" {
		return c.LogsURL
	}

	if c.BaseURL == "" {
		return ""
	}

	return strings.TrimRight(c.BaseURL, "/") + "/v1/logs"
}

func (c Config) AuthorizationHeader() string {
	if c.Authorization != "" {
		return c.Authorization
	}

	if c.Username == "" && c.Password == "" {
		return ""
	}

	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.Username+":"+c.Password))
}

func (c Config) Headers() map[string]string {
	auth := c.AuthorizationHeader()
	if auth == "" {
		return nil
	}

	return map[string]string{"Authorization": auth}
}

// Enabled reports whether any remote telemetry export is configured.
func (c Config) Enabled() bool {
	return c.TraceEndpoint() != "" || c.LogEndpoint() != "" || c.MetricEndpoint() != ""
}
