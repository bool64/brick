package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GrafanaStyleDefaults(t *testing.T) {
	cfg := Config{
		BaseURL:  "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp",
		Username: "12345",
		Password: "glc_abcdef",
	}

	assert.Equal(t, "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces", cfg.TraceEndpoint())
	assert.Equal(t, "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/metrics", cfg.MetricEndpoint())
	assert.Equal(t, "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/logs", cfg.LogEndpoint())
	assert.Equal(t, "Basic MTIzNDU6Z2xjX2FiY2RlZg==", cfg.AuthorizationHeader())
	assert.Equal(t, map[string]string{
		"Authorization": "Basic MTIzNDU6Z2xjX2FiY2RlZg==",
	}, cfg.Headers())
}
