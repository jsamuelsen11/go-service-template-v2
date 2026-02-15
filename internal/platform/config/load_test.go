package config_test

import (
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
)

func TestLoad_LocalProfile(t *testing.T) {
	t.Chdir("../../..")

	cfg, err := config.Load("local")
	if err != nil {
		t.Fatalf("Load(\"local\") error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want \"debug\"", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want \"text\"", cfg.Log.Format)
	}
	if cfg.Telemetry.Enabled {
		t.Error("Telemetry.Enabled = true, want false for local")
	}
}

func TestLoad_ProdProfile(t *testing.T) {
	t.Chdir("../../..")

	cfg, err := config.Load("prod")
	if err != nil {
		t.Fatalf("Load(\"prod\") error: %v", err)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want \"info\"", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want \"json\"", cfg.Log.Format)
	}
	if !cfg.Telemetry.Enabled {
		t.Error("Telemetry.Enabled = false, want true for prod")
	}
	if cfg.Telemetry.Exporter != "otlp" {
		t.Errorf("Telemetry.Exporter = %q, want \"otlp\"", cfg.Telemetry.Exporter)
	}
	if cfg.Telemetry.Endpoint == "" {
		t.Error("Telemetry.Endpoint is empty, want non-empty for prod")
	}
}

func TestLoad_BaseConfigInheritance(t *testing.T) {
	t.Chdir("../../..")

	cfg, err := config.Load("local")
	if err != nil {
		t.Fatalf("Load(\"local\") error: %v", err)
	}

	// These come from base.yaml, not overridden by local.yaml.
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want \"0.0.0.0\" (from base)", cfg.Server.Host)
	}
	if cfg.Client.Retry.MaxAttempts != 3 {
		t.Errorf("Client.Retry.MaxAttempts = %d, want 3 (from base)", cfg.Client.Retry.MaxAttempts)
	}
	if cfg.Client.CircuitBreaker.MaxFailures != 5 {
		t.Errorf("Client.CircuitBreaker.MaxFailures = %d, want 5 (from base)",
			cfg.Client.CircuitBreaker.MaxFailures)
	}
}

func TestLoad_EnvOverrideSimpleKey(t *testing.T) {
	t.Chdir("../../..")
	t.Setenv("APP_SERVER_PORT", "9090")

	cfg, err := config.Load("local")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090 (env override)", cfg.Server.Port)
	}
}

func TestLoad_EnvOverrideSnakeCaseKey(t *testing.T) {
	t.Chdir("../../..")
	t.Setenv("APP_SERVER_READ_TIMEOUT", "15s")

	cfg, err := config.Load("local")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	want := 15 * time.Second
	if cfg.Server.ReadTimeout != want {
		t.Errorf("Server.ReadTimeout = %v, want %v (env override)", cfg.Server.ReadTimeout, want)
	}
}

func TestLoad_EnvOverrideDeeplyNestedKey(t *testing.T) {
	t.Chdir("../../..")
	t.Setenv("APP_CLIENT_RETRY_MAX_ATTEMPTS", "7")

	cfg, err := config.Load("local")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Client.Retry.MaxAttempts != 7 {
		t.Errorf("Client.Retry.MaxAttempts = %d, want 7 (env override)", cfg.Client.Retry.MaxAttempts)
	}
}

func TestLoad_MissingProfile(t *testing.T) {
	t.Chdir("../../..")

	_, err := config.Load("nonexistent")
	if err == nil {
		t.Fatal("Load(\"nonexistent\") returned nil error, want error")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Server.Port = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() returned nil, want error for port=0")
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Log.Level = "verbose"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() returned nil, want error for invalid log level")
	}
}

func TestValidate_OtlpWithoutEndpoint(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.Exporter = "otlp"
	cfg.Telemetry.Endpoint = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() returned nil, want error for otlp without endpoint")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error for valid config: %v", err)
	}
}

// validBaseConfig returns a Config with all fields set to valid values.
func validBaseConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
		Client: config.ClientConfig{
			BaseURL: "http://localhost:8081",
			Timeout: 30 * time.Second,
			Retry: config.RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			CircuitBreaker: config.CircuitBreakerConfig{
				MaxFailures:   5,
				Timeout:       30 * time.Second,
				HalfOpenLimit: 1,
			},
		},
		Telemetry: config.TelemetryConfig{
			Enabled:  false,
			Exporter: "stdout",
		},
	}
}
