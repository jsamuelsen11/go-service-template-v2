// Package config provides configuration loading and validation for the service.
// Configuration is loaded from YAML files with environment variable overrides
// using a layered system: defaults -> base.yaml -> {profile}.yaml -> env vars.
package config

import "time"

// Config holds all configuration for the service.
type Config struct {
	Server    ServerConfig    `koanf:"server"`
	Log       LogConfig       `koanf:"log"`
	Client    ClientConfig    `koanf:"client"`
	Telemetry TelemetryConfig `koanf:"telemetry"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string        `koanf:"host"`
	Port         int           `koanf:"port"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	IdleTimeout  time.Duration `koanf:"idle_timeout"`
}

// LogConfig holds structured logging settings.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// ClientConfig holds downstream HTTP client settings.
type ClientConfig struct {
	BaseURL        string               `koanf:"base_url"`
	Timeout        time.Duration        `koanf:"timeout"`
	Retry          RetryConfig          `koanf:"retry"`
	CircuitBreaker CircuitBreakerConfig `koanf:"circuit_breaker"`
}

// RetryConfig holds retry policy settings with exponential backoff.
type RetryConfig struct {
	MaxAttempts     int           `koanf:"max_attempts"`
	InitialInterval time.Duration `koanf:"initial_interval"`
	MaxInterval     time.Duration `koanf:"max_interval"`
	Multiplier      float64       `koanf:"multiplier"`
}

// CircuitBreakerConfig holds circuit breaker settings.
type CircuitBreakerConfig struct {
	MaxFailures   int           `koanf:"max_failures"`
	Timeout       time.Duration `koanf:"timeout"`
	HalfOpenLimit int           `koanf:"half_open_limit"`
}

// TelemetryConfig holds OpenTelemetry settings.
type TelemetryConfig struct {
	Enabled     bool   `koanf:"enabled"`
	Exporter    string `koanf:"exporter"`
	Endpoint    string `koanf:"endpoint"`
	ServiceName string `koanf:"service_name"`
}
