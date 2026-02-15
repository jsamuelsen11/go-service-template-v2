package config

import (
	"errors"
	"fmt"
)

// Validate checks all configuration values and returns aggregated errors.
func (c *Config) Validate() error {
	return errors.Join(
		c.Server.validate(),
		c.Log.validate(),
		c.Client.validate(),
		c.Telemetry.validate(),
	)
}

func (s *ServerConfig) validate() error {
	var errs []error

	if s.Port < 1 || s.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", s.Port))
	}
	if s.ReadTimeout <= 0 {
		errs = append(errs, errors.New("server.read_timeout must be positive"))
	}
	if s.WriteTimeout <= 0 {
		errs = append(errs, errors.New("server.write_timeout must be positive"))
	}

	return errors.Join(errs...)
}

func (l *LogConfig) validate() error {
	var errs []error

	switch l.Level {
	case "debug", "info", "warn", "error":
		// Valid levels.
	default:
		errs = append(errs, fmt.Errorf("log.level must be one of: debug, info, warn, error; got %q", l.Level))
	}

	switch l.Format {
	case "json", "text":
		// Valid formats.
	default:
		errs = append(errs, fmt.Errorf("log.format must be one of: json, text; got %q", l.Format))
	}

	return errors.Join(errs...)
}

func (cl *ClientConfig) validate() error {
	var errs []error

	if cl.BaseURL == "" {
		errs = append(errs, errors.New("client.base_url must not be empty"))
	}
	if cl.Timeout <= 0 {
		errs = append(errs, errors.New("client.timeout must be positive"))
	}
	if cl.Retry.MaxAttempts < 1 {
		errs = append(errs, fmt.Errorf("client.retry.max_attempts must be >= 1, got %d", cl.Retry.MaxAttempts))
	}
	if cl.Retry.Multiplier <= 0 {
		errs = append(errs, fmt.Errorf("client.retry.multiplier must be positive, got %f", cl.Retry.Multiplier))
	}
	if cl.CircuitBreaker.MaxFailures < 1 {
		errs = append(errs, fmt.Errorf("client.circuit_breaker.max_failures must be >= 1, got %d",
			cl.CircuitBreaker.MaxFailures))
	}

	return errors.Join(errs...)
}

func (t *TelemetryConfig) validate() error {
	if !t.Enabled {
		return nil
	}

	var errs []error

	switch t.Exporter {
	case "stdout", "otlp":
		// Valid exporters.
	default:
		errs = append(errs, fmt.Errorf("telemetry.exporter must be one of: stdout, otlp; got %q", t.Exporter))
	}

	if t.Exporter == "otlp" && t.Endpoint == "" {
		errs = append(errs, errors.New("telemetry.endpoint must not be empty when exporter is otlp"))
	}

	return errors.Join(errs...)
}
