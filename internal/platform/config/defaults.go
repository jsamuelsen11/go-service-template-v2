package config

const (
	defaultServerPort = 8080

	defaultRetryMaxAttempts = 3
	defaultRetryMultiplier  = 2.0

	defaultCircuitBreakerMaxFailures = 5
	defaultCircuitBreakerHalfOpen    = 1
)

// defaults returns the default configuration values.
// These are loaded first and can be overridden by base.yaml, profile YAML, and env vars.
func defaults() map[string]any {
	return map[string]any{
		"server.host":          "0.0.0.0",
		"server.port":          defaultServerPort,
		"server.read_timeout":  "5s",
		"server.write_timeout": "10s",
		"server.idle_timeout":  "120s",

		"log.level":  "info",
		"log.format": "json",

		"client.base_url":                        "http://localhost:8081",
		"client.timeout":                         "30s",
		"client.retry.max_attempts":              defaultRetryMaxAttempts,
		"client.retry.initial_interval":          "100ms",
		"client.retry.max_interval":              "10s",
		"client.retry.multiplier":                defaultRetryMultiplier,
		"client.circuit_breaker.max_failures":    defaultCircuitBreakerMaxFailures,
		"client.circuit_breaker.timeout":         "30s",
		"client.circuit_breaker.half_open_limit": defaultCircuitBreakerHalfOpen,

		"telemetry.enabled":  false,
		"telemetry.exporter": "stdout",
		"telemetry.endpoint": "",
	}
}
