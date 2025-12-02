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
	"strings"
	"time"
)

// Config holds the complete contextd v2 configuration.
type Config struct {
	Server        ServerConfig
	Observability ObservabilityConfig
	PreFetch      PreFetchConfig
	Checkpoint    CheckpointConfig
	Qdrant        QdrantConfig
	Embeddings    EmbeddingsConfig
	Repository    RepositoryConfig
}

// RepositoryConfig holds repository indexing configuration.
type RepositoryConfig struct {
	// IgnoreFiles is a list of ignore file names to parse from project root.
	// Patterns from these files are used as exclude patterns during indexing.
	// Default: [".gitignore", ".dockerignore", ".contextdignore"]
	IgnoreFiles []string `koanf:"ignore_files"`

	// FallbackExcludes are used when no ignore files are found in the project.
	// Default: [".git/**", "node_modules/**", "vendor/**", "__pycache__/**"]
	FallbackExcludes []string `koanf:"fallback_excludes"`
}

// QdrantConfig holds Qdrant vector database configuration.
type QdrantConfig struct {
	Host           string `koanf:"host"`
	Port           int    `koanf:"port"`
	HTTPPort       int    `koanf:"http_port"`
	CollectionName string `koanf:"collection_name"`
	VectorSize     uint64 `koanf:"vector_size"`
	DataPath       string `koanf:"data_path"`
}

// EmbeddingsConfig holds embeddings service configuration.
type EmbeddingsConfig struct {
	Provider string `koanf:"provider"` // "fastembed" or "tei"
	BaseURL  string `koanf:"base_url"` // TEI URL (if using TEI)
	Model    string `koanf:"model"`
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
//
// Server:
//   - SERVER_PORT: HTTP server port (default: 9090)
//   - SERVER_SHUTDOWN_TIMEOUT: Graceful shutdown timeout (default: 10s)
//
// Qdrant:
//   - QDRANT_HOST: Qdrant host (default: localhost)
//   - QDRANT_PORT: Qdrant gRPC port (default: 6334)
//   - QDRANT_HTTP_PORT: Qdrant HTTP port (default: 6333)
//   - QDRANT_COLLECTION: Default collection name (default: contextd_default)
//   - QDRANT_VECTOR_SIZE: Vector dimensions (default: 384 for FastEmbed)
//   - CONTEXTD_DATA_PATH: Base data path (default: /data)
//
// Embeddings:
//   - EMBEDDINGS_PROVIDER: Provider type: fastembed or tei (default: fastembed)
//   - EMBEDDINGS_MODEL: Embedding model (default: BAAI/bge-small-en-v1.5)
//   - EMBEDDING_BASE_URL: TEI URL if using TEI (default: http://localhost:8080)
//
// Checkpoint:
//   - CHECKPOINT_MAX_CONTENT_SIZE_KB: Max checkpoint size in KB (default: 1024)
//
// Telemetry:
//   - OTEL_ENABLE: Enable OpenTelemetry (default: true)
//   - OTEL_SERVICE_NAME: Service name for traces (default: contextd)
//
// Pre-fetch:
//   - PREFETCH_ENABLED: Enable pre-fetch engine (default: true)
//   - PREFETCH_CACHE_TTL: Cache TTL (default: 5m)
//   - PREFETCH_CACHE_MAX_ENTRIES: Maximum cache entries (default: 100)
//
// Example:
//
//	cfg := config.Load()
//	fmt.Println("Qdrant host:", cfg.Qdrant.Host)
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

	// Qdrant configuration
	cfg.Qdrant = QdrantConfig{
		Host:           getEnvString("QDRANT_HOST", "localhost"),
		Port:           getEnvInt("QDRANT_PORT", 6334),
		HTTPPort:       getEnvInt("QDRANT_HTTP_PORT", 6333),
		CollectionName: getEnvString("QDRANT_COLLECTION", "contextd_default"),
		VectorSize:     uint64(getEnvInt("QDRANT_VECTOR_SIZE", 384)), // FastEmbed default
		DataPath:       getEnvString("CONTEXTD_DATA_PATH", "/data"),
	}

	// Embeddings configuration
	cfg.Embeddings = EmbeddingsConfig{
		Provider: getEnvString("EMBEDDINGS_PROVIDER", "fastembed"),
		BaseURL:  getEnvString("EMBEDDING_BASE_URL", "http://localhost:8080"),
		Model:    getEnvString("EMBEDDINGS_MODEL", "BAAI/bge-small-en-v1.5"),
	}

	// Repository indexing configuration
	cfg.Repository = RepositoryConfig{
		IgnoreFiles: getEnvStringSlice("REPOSITORY_IGNORE_FILES", []string{
			".gitignore",
			".dockerignore",
			".contextdignore",
		}),
		FallbackExcludes: getEnvStringSlice("REPOSITORY_FALLBACK_EXCLUDES", []string{
			".git/**",
			"node_modules/**",
			"vendor/**",
			"__pycache__/**",
		}),
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

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma, trim whitespace
		parts := make([]string, 0)
		for _, part := range splitAndTrim(value, ",") {
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		result = append(result, trimmed)
	}
	return result
}
