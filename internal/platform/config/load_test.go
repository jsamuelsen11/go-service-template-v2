package config_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
)

// configDir returns the absolute path to the project's configs/ directory.
func configDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// thisFile is internal/platform/config/load_test.go — walk up 4 levels to project root.
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "configs")
}

func withDir(t *testing.T) config.Option {
	t.Helper()
	return config.WithConfigDir(configDir(t))
}

// --- Load tests ---

func TestLoad_LocalProfile(t *testing.T) {
	cfg, err := config.Load("local", withDir(t))
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
	cfg, err := config.Load("prod", withDir(t))
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
	cfg, err := config.Load("local", withDir(t))
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
	t.Setenv("APP_SERVER_PORT", "9090")

	cfg, err := config.Load("local", withDir(t))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090 (env override)", cfg.Server.Port)
	}
}

func TestLoad_EnvOverrideSnakeCaseKey(t *testing.T) {
	t.Setenv("APP_SERVER_READ_TIMEOUT", "15s")

	cfg, err := config.Load("local", withDir(t))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	want := 15 * time.Second
	if cfg.Server.ReadTimeout != want {
		t.Errorf("Server.ReadTimeout = %v, want %v (env override)", cfg.Server.ReadTimeout, want)
	}
}

func TestLoad_EnvOverrideDeeplyNestedKey(t *testing.T) {
	t.Setenv("APP_CLIENT_RETRY_MAX_ATTEMPTS", "7")

	cfg, err := config.Load("local", withDir(t))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Client.Retry.MaxAttempts != 7 {
		t.Errorf("Client.Retry.MaxAttempts = %d, want 7 (env override)", cfg.Client.Retry.MaxAttempts)
	}
}

func TestLoad_MissingProfile(t *testing.T) {
	_, err := config.Load("nonexistent", withDir(t))
	if err == nil {
		t.Fatal("Load(\"nonexistent\") returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "profile config") {
		t.Errorf("error = %q, want it to mention \"profile config\"", err.Error())
	}
}

func TestLoad_EmptyProfile(t *testing.T) {
	_, err := config.Load("", withDir(t))
	if err == nil {
		t.Fatal("Load(\"\") returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "profile must not be empty") {
		t.Errorf("error = %q, want it to mention \"profile must not be empty\"", err.Error())
	}
}

func TestLoad_ProfileWithPathSeparator(t *testing.T) {
	_, err := config.Load("../etc/passwd", withDir(t))
	if err == nil {
		t.Fatal("Load(\"../etc/passwd\") returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "path separator") {
		t.Errorf("error = %q, want it to mention \"path separator\"", err.Error())
	}
}

func TestLoad_ProfileWithPathTraversal(t *testing.T) {
	_, err := config.Load("..evil", withDir(t))
	if err == nil {
		t.Fatal("Load(\"..evil\") returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("error = %q, want it to mention \"path traversal\"", err.Error())
	}
}

// --- Validation tests ---

func TestValidate_InvalidPort(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Server.Port = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for port=0")
	}
	if !strings.Contains(err.Error(), "server.port") {
		t.Errorf("error = %q, want it to mention \"server.port\"", err.Error())
	}
}

func TestValidate_PortTooHigh(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Server.Port = 70000

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for port=70000")
	}
	if !strings.Contains(err.Error(), "server.port") {
		t.Errorf("error = %q, want it to mention \"server.port\"", err.Error())
	}
}

func TestValidate_ServerReadTimeoutNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Server.ReadTimeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for read_timeout=0")
	}
	if !strings.Contains(err.Error(), "server.read_timeout") {
		t.Errorf("error = %q, want it to mention \"server.read_timeout\"", err.Error())
	}
}

func TestValidate_ServerWriteTimeoutNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Server.WriteTimeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for write_timeout=0")
	}
	if !strings.Contains(err.Error(), "server.write_timeout") {
		t.Errorf("error = %q, want it to mention \"server.write_timeout\"", err.Error())
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Log.Level = "verbose"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for invalid log level")
	}
	if !strings.Contains(err.Error(), "log.level") {
		t.Errorf("error = %q, want it to mention \"log.level\"", err.Error())
	}
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Log.Format = "xml"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for invalid log format")
	}
	if !strings.Contains(err.Error(), "log.format") {
		t.Errorf("error = %q, want it to mention \"log.format\"", err.Error())
	}
}

func TestValidate_EmptyClientBaseURL(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.BaseURL = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for empty base_url")
	}
	if !strings.Contains(err.Error(), "client.base_url") {
		t.Errorf("error = %q, want it to mention \"client.base_url\"", err.Error())
	}
}

func TestValidate_ClientTimeoutNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Timeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for client timeout=0")
	}
	if !strings.Contains(err.Error(), "client.timeout") {
		t.Errorf("error = %q, want it to mention \"client.timeout\"", err.Error())
	}
}

func TestValidate_RetryMaxAttemptsLessThanOne(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Retry.MaxAttempts = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for max_attempts=0")
	}
	if !strings.Contains(err.Error(), "client.retry.max_attempts") {
		t.Errorf("error = %q, want it to mention \"client.retry.max_attempts\"", err.Error())
	}
}

func TestValidate_RetryInitialIntervalNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Retry.InitialInterval = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for initial_interval=0")
	}
	if !strings.Contains(err.Error(), "client.retry.initial_interval") {
		t.Errorf("error = %q, want it to mention \"client.retry.initial_interval\"", err.Error())
	}
}

func TestValidate_RetryMaxIntervalNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Retry.MaxInterval = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for max_interval=0")
	}
	if !strings.Contains(err.Error(), "client.retry.max_interval") {
		t.Errorf("error = %q, want it to mention \"client.retry.max_interval\"", err.Error())
	}
}

func TestValidate_RetryMultiplierNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Retry.Multiplier = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for multiplier=0")
	}
	if !strings.Contains(err.Error(), "client.retry.multiplier") {
		t.Errorf("error = %q, want it to mention \"client.retry.multiplier\"", err.Error())
	}
}

func TestValidate_RetryInitialIntervalExceedsMaxInterval(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.Retry.InitialInterval = 20 * time.Second
	cfg.Client.Retry.MaxInterval = 5 * time.Second

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for initial_interval > max_interval")
	}
	if !strings.Contains(err.Error(), "initial_interval") && !strings.Contains(err.Error(), "max_interval") {
		t.Errorf("error = %q, want it to mention interval mismatch", err.Error())
	}
}

func TestValidate_CircuitBreakerMaxFailuresLessThanOne(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.CircuitBreaker.MaxFailures = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for max_failures=0")
	}
	if !strings.Contains(err.Error(), "client.circuit_breaker.max_failures") {
		t.Errorf("error = %q, want it to mention \"client.circuit_breaker.max_failures\"", err.Error())
	}
}

func TestValidate_CircuitBreakerTimeoutNonPositive(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Client.CircuitBreaker.Timeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for circuit_breaker timeout=0")
	}
	if !strings.Contains(err.Error(), "client.circuit_breaker.timeout") {
		t.Errorf("error = %q, want it to mention \"client.circuit_breaker.timeout\"", err.Error())
	}
}

func TestValidate_OtlpWithoutEndpoint(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.Exporter = "otlp"
	cfg.Telemetry.Endpoint = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for otlp without endpoint")
	}
	if !strings.Contains(err.Error(), "telemetry.endpoint") {
		t.Errorf("error = %q, want it to mention \"telemetry.endpoint\"", err.Error())
	}
}

func TestValidate_InvalidTelemetryExporter(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.Exporter = "prometheus"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for invalid exporter")
	}
	if !strings.Contains(err.Error(), "telemetry.exporter") {
		t.Errorf("error = %q, want it to mention \"telemetry.exporter\"", err.Error())
	}
}

func TestValidate_TelemetryDisabledSkipsValidation(t *testing.T) {
	t.Parallel()

	cfg := validBaseConfig()
	cfg.Telemetry.Enabled = false
	cfg.Telemetry.Exporter = "invalid"
	cfg.Telemetry.Endpoint = ""

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error for disabled telemetry: %v", err)
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
