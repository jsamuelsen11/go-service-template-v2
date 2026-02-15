package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	env "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const envPrefix = "APP_"

// Load reads configuration using a 4-layer hierarchy (highest precedence last):
//
//  1. Hardcoded defaults
//  2. Base config (configs/base.yaml)
//  3. Profile config (configs/{profile}.yaml)
//  4. Environment variables (APP_ prefix)
//
// Environment variable mapping uses key matching against loaded config keys
// to resolve ambiguity between nesting separators and field-internal underscores:
//
//	APP_SERVER_PORT           -> server.port
//	APP_SERVER_READ_TIMEOUT   -> server.read_timeout
//	APP_LOG_LEVEL             -> log.level
//	APP_CLIENT_RETRY_MAX_ATTEMPTS -> client.retry.max_attempts
func Load(profile string) (*Config, error) {
	k := koanf.New(".")

	// Layer 1: Hardcoded defaults.
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// Layer 2: Base config (shared across all profiles).
	basePath := filepath.Join("configs", "base.yaml")
	if err := k.Load(file.Provider(basePath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("loading base config %s: %w", basePath, err)
	}

	// Layer 3: Profile-specific config.
	profilePath := filepath.Join("configs", profile+".yaml")
	if err := k.Load(file.Provider(profilePath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("loading profile config %s: %w", profilePath, err)
	}

	// Layer 4: Environment variables with APP_ prefix.
	// Build a reverse lookup from known koanf keys so that env vars like
	// APP_SERVER_READ_TIMEOUT correctly resolve to "server.read_timeout"
	// instead of being ambiguously split as "server.read.timeout".
	envLookup := buildEnvLookup(k.Keys())

	if err := k.Load(env.Provider(".", env.Opt{
		Prefix: envPrefix,
		TransformFunc: func(key, value string) (string, any) {
			key = strings.TrimPrefix(key, envPrefix)
			key = strings.ToLower(key)

			if koanfKey, ok := envLookup[key]; ok {
				return koanfKey, value
			}

			// Fallback: simple underscore-to-dot replacement.
			return strings.ReplaceAll(key, "_", "."), value
		},
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	// Unmarshal into Config struct.
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// Validate.
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// buildEnvLookup creates a reverse mapping from env-style keys to koanf dotted keys.
// For each koanf key like "server.read_timeout", the env form "server_read_timeout"
// is computed by replacing dots with underscores. This allows unambiguous matching
// when an env var arrives (e.g. APP_SERVER_READ_TIMEOUT -> "server.read_timeout").
func buildEnvLookup(keys []string) map[string]string {
	lookup := make(map[string]string, len(keys))
	for _, key := range keys {
		envKey := strings.ReplaceAll(key, ".", "_")
		lookup[envKey] = key
	}
	return lookup
}
