// Package config provides configuration loading for contextd v2.
//
// Configuration is loaded from environment variables with sensible defaults.
// This package supports server, observability, and application-specific settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the complete contextd v2 configuration.
type Config struct {
	Server        ServerConfig
	Observability ObservabilityConfig
	PreFetch      PreFetchConfig
	Checkpoint    CheckpointConfig
}

// CheckpointConfig holds checkpoint service configuration.
type CheckpointConfig struct {
	MaxContentSizeKB int `koanf:"max_content_size_kb"` // Maximum content size in KB (default: 1024 = 1MB)
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port            int           `koanf:"http_port"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

// ObservabilityConfig holds OpenTelemetry configuration.
type ObservabilityConfig struct {
	EnableTelemetry bool   `koanf:"enable_telemetry"`
	ServiceName     string `koanf:"service_name"`
}

// PreFetchConfig holds pre-fetch engine configuration.
type PreFetchConfig struct {
	Enabled         bool
	CacheTTL        time.Duration
	CacheMaxEntries int
	Rules           PreFetchRulesConfig
}

// PreFetchRulesConfig holds configuration for individual pre-fetch rules.
type PreFetchRulesConfig struct {
	BranchDiff   RuleConfig
	RecentCommit RuleConfig
	CommonFiles  RuleConfig
}

// RuleConfig holds configuration for a single pre-fetch rule.
type RuleConfig struct {
	Enabled   bool
	MaxFiles  int
	MaxSizeKB int
	TimeoutMS int
}

// Load loads configuration from environment variables with defaults.
//
// Environment variables:
//   - SERVER_PORT: HTTP server port (default: 9090)
//   - SERVER_SHUTDOWN_TIMEOUT: Graceful shutdown timeout (default: 10s)
//   - OTEL_ENABLE: Enable OpenTelemetry (default: true)
//   - OTEL_SERVICE_NAME: Service name for traces (default: contextd)
//   - PREFETCH_ENABLED: Enable pre-fetch engine (default: true)
//   - PREFETCH_CACHE_TTL: Cache TTL (default: 5m)
//   - PREFETCH_CACHE_MAX_ENTRIES: Maximum cache entries (default: 100)
//   - PREFETCH_BRANCH_DIFF_ENABLED: Enable branch diff rule (default: true)
//   - PREFETCH_RECENT_COMMIT_ENABLED: Enable recent commit rule (default: true)
//   - PREFETCH_COMMON_FILES_ENABLED: Enable common files rule (default: true)
//
// Example:
//
//	cfg := config.Load()
//	fmt.Println("Server port:", cfg.Server.Port)
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 9090),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Observability: ObservabilityConfig{
			EnableTelemetry: getEnvBool("OTEL_ENABLE", true),
			ServiceName:     getEnvString("OTEL_SERVICE_NAME", "contextd"),
		},
		PreFetch: PreFetchConfig{
			Enabled:         getEnvBool("PREFETCH_ENABLED", true),
			CacheTTL:        getEnvDuration("PREFETCH_CACHE_TTL", 5*time.Minute),
			CacheMaxEntries: getEnvInt("PREFETCH_CACHE_MAX_ENTRIES", 100),
			Rules: PreFetchRulesConfig{
				BranchDiff: RuleConfig{
					Enabled:   getEnvBool("PREFETCH_BRANCH_DIFF_ENABLED", true),
					MaxFiles:  getEnvInt("PREFETCH_BRANCH_DIFF_MAX_FILES", 10),
					MaxSizeKB: getEnvInt("PREFETCH_BRANCH_DIFF_MAX_SIZE_KB", 50),
					TimeoutMS: getEnvInt("PREFETCH_BRANCH_DIFF_TIMEOUT_MS", 1000),
				},
				RecentCommit: RuleConfig{
					Enabled:   getEnvBool("PREFETCH_RECENT_COMMIT_ENABLED", true),
					MaxFiles:  0, // Not used for commit rule
					MaxSizeKB: getEnvInt("PREFETCH_RECENT_COMMIT_MAX_SIZE_KB", 20),
					TimeoutMS: getEnvInt("PREFETCH_RECENT_COMMIT_TIMEOUT_MS", 500),
				},
				CommonFiles: RuleConfig{
					Enabled:   getEnvBool("PREFETCH_COMMON_FILES_ENABLED", true),
					MaxFiles:  getEnvInt("PREFETCH_COMMON_FILES_MAX_FILES", 3),
					MaxSizeKB: 0, // Not used for common files
					TimeoutMS: getEnvInt("PREFETCH_COMMON_FILES_TIMEOUT_MS", 500),
				},
			},
		},
	}

	// Checkpoint configuration
	cfg.Checkpoint = CheckpointConfig{
		MaxContentSizeKB: getEnvInt("CHECKPOINT_MAX_CONTENT_SIZE_KB", 1024), // Default 1MB
	}

	return cfg
}

// Validate validates the configuration.
//
// Returns an error if:
//   - Server port is not between 1 and 65535
//   - Shutdown timeout is not positive
//   - Service name is empty (when telemetry is enabled)
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be 1-65535)", c.Server.Port)
	}

	if c.Server.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}

	// Validate observability configuration
	if c.Observability.EnableTelemetry && c.Observability.ServiceName == "" {
		return errors.New("service name required when telemetry is enabled")
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
