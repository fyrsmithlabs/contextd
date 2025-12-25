// Package config provides configuration loading for contextd.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	maxConfigFileSize = 1024 * 1024 // 1MB
)

// LoadWithFile loads configuration from YAML file, then overrides with environment variables.
//
// Configuration precedence (highest to lowest):
//  1. Environment variables (SERVER_HTTP_PORT, OBSERVABILITY_SERVICE_NAME, etc.)
//  2. YAML config file (~/.config/contextd/config.yaml)
//  3. Hardcoded defaults
//
// The configPath parameter specifies the YAML file to load. If empty, uses default path.
// Default path: ~/.config/contextd/config.yaml
//
// # Security Considerations
//
// File Permissions: Configuration file MUST have 0600 permissions (owner read/write only).
// Files with weaker permissions (e.g., 0644 world-readable) will be rejected.
//
// Path Validation: Only configuration files in allowed directories can be loaded:
//   - ~/.config/contextd/ (user's config directory)
//   - /etc/contextd/ (system-wide config directory)
//
// Absolute paths outside these directories are rejected to prevent path traversal attacks.
//
// File Size Limit: Configuration files larger than 1MB are rejected to prevent
// resource exhaustion attacks.
//
// # Environment Variable Mapping
//
// Environment variables use underscore separator and are uppercased.
// The transformer maps environment variables to YAML field names:
//
//	SERVER_HTTP_PORT -> server.http_port
//	OBSERVABILITY_SERVICE_NAME -> observability.service_name
//	PREFETCH_CACHE_TTL -> prefetch.cache_ttl
//
// # Example
//
//	cfg, err := config.LoadWithFile("")  // Use default path
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadWithFile(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Use default config path if not specified
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".config", "contextd", "config.yaml")
	}

	// Validate config path (even if file doesn't exist)
	if err := validateConfigPath(configPath); err != nil {
		return nil, fmt.Errorf("config path validation failed: %w", err)
	}
	// Load from YAML file if it exists
	if _, err := os.Stat(configPath); err == nil {
		// Open file once and validate using file descriptor to avoid TOCTOU race
		f, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer f.Close()

		// Validate file properties using already-opened file descriptor
		info, err := f.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat config file: %w", err)
		}

		if err := validateConfigFileProperties(info); err != nil {
			return nil, fmt.Errorf("config file validation failed: %w", err)
		}

		// Read content from already-opened file
		content, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Use rawbytes provider to avoid re-opening the file
		if err := k.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
	}

	// Override with environment variables
	// Environment variables use underscore separator and are uppercased
	// Example: SERVER_HTTP_PORT -> server.http_port
	if err := k.Load(env.Provider("", ".", func(s string) string {
		// Custom transformer for contextd config
		// Handles both simple fields and compound underscore fields
		//
		// Examples:
		//   SERVER_HTTP_PORT -> server.http_port
		//   OBSERVABILITY_SERVICE_NAME -> observability.service_name
		//   PREFETCH_CACHE_TTL -> prefetch.cache_ttl
		//
		// Strategy: Split on first underscore only (section.field_name pattern)

		lower := strings.ToLower(s)
		parts := strings.SplitN(lower, "_", 2)

		if len(parts) == 1 {
			// No underscore: simple field (unlikely for config)
			return lower
		}

		// Two parts: section and field_name
		// Replace remaining underscores in section with dots (rare)
		// Keep underscores in field name
		section := parts[0]
		fieldName := parts[1]

		return section + "." + fieldName
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for missing values
	applyDefaults(&cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// EnsureConfigDir creates the contextd config directory if it doesn't exist.
// This is called during startup to ensure new users have the config directory ready.
// The directory is created with 0700 permissions (owner read/write/execute only).
func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	return nil
}

// validateConfigPath checks if path is in allowed directories.
// This validation runs even if the file doesn't exist yet.
func validateConfigPath(path string) error {
	// Resolve to absolute path and follow symlinks to prevent path traversal
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Resolve symlinks to prevent attackers from using symlinks to escape allowed directories
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink evaluation fails, continue with absPath
		// This allows validation of paths that dont exist yet
		resolvedPath = absPath
	}

	// Check if path is in allowed directories
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	allowedDirs := []string{
		filepath.Join(home, ".config", "contextd"),
		"/etc/contextd",
	}

	allowed := false
	for _, dir := range allowedDirs {
		if strings.HasPrefix(resolvedPath, dir) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("config file must be in ~/.config/contextd/ or /etc/contextd/")
	}

	return nil
}

// validateConfigFileProperties checks file permissions and size.
// This validation only runs if the file exists.
// Takes FileInfo from an already-opened file descriptor to avoid TOCTOU race.
func validateConfigFileProperties(info os.FileInfo) error {

	// Check file permissions (must be 0600 or 0400)
	// Skip on Windows (different permission model)
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 && perm != 0400 {
			return fmt.Errorf("insecure config file permissions: %v (expected 0600 or 0400)", perm)
		}
	}

	// Check file size (max 1MB)
	if info.Size() > maxConfigFileSize {
		return fmt.Errorf("config file too large: %d bytes (max %d)", info.Size(), maxConfigFileSize)
	}

	return nil
}

// applyDefaults sets default values for missing configuration fields.
func applyDefaults(cfg *Config) {
	// Server defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9090
	}
	if cfg.Server.ShutdownTimeout == 0 {

	// Production defaults (loaded from environment)
	cfg.Production = loadProductionConfig()
		cfg.Server.ShutdownTimeout = 10 * time.Second
	}

	// Observability defaults
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = "contextd"
	}

	// PreFetch defaults (only if enabled but values not set)
	if cfg.PreFetch.Enabled {
		if cfg.PreFetch.CacheTTL == 0 {
			cfg.PreFetch.CacheTTL = 5 * time.Minute
		}
		if cfg.PreFetch.CacheMaxEntries == 0 {
			cfg.PreFetch.CacheMaxEntries = 100
		}
	}

	// Qdrant defaults
	if cfg.Qdrant.Host == "" {
		cfg.Qdrant.Host = "localhost"
	}
	if cfg.Qdrant.Port == 0 {
		cfg.Qdrant.Port = 6334
	}
	if cfg.Qdrant.CollectionName == "" {
		cfg.Qdrant.CollectionName = "contextd_default"
	}
	if cfg.Qdrant.VectorSize == 0 {
		cfg.Qdrant.VectorSize = 384 // bge-small-en-v1.5 dimensions
	}

	// VectorStore defaults (chromem is default - embedded, no external deps)
	if cfg.VectorStore.Provider == "" {
		cfg.VectorStore.Provider = "chromem"
	}
	if cfg.VectorStore.Chromem.Path == "" {
		cfg.VectorStore.Chromem.Path = "~/.config/contextd/vectorstore"
	}
	if cfg.VectorStore.Chromem.DefaultCollection == "" {
		cfg.VectorStore.Chromem.DefaultCollection = "contextd_default"
	}
	if cfg.VectorStore.Chromem.VectorSize == 0 {
		cfg.VectorStore.Chromem.VectorSize = 384 // bge-small-en-v1.5 dimensions
	}
	// Compress defaults to true (set explicitly since zero value is false)
	// Note: This is handled in Load() with getEnvBool

	// Embeddings defaults
	if cfg.Embeddings.BaseURL == "" {
		cfg.Embeddings.BaseURL = "http://localhost:8080"
	}
	if cfg.Embeddings.Model == "" {
		cfg.Embeddings.Model = "BAAI/bge-small-en-v1.5"
	}
}

// loadProductionConfig loads production configuration from environment variables.
func loadProductionConfig() ProductionConfig {
	prodMode := os.Getenv("CONTEXTD_PRODUCTION_MODE") == "1"
	localMode := os.Getenv("CONTEXTD_LOCAL_MODE") == "1"
	
	return ProductionConfig{
		Enabled:               prodMode,
		LocalModeAcknowledged: localMode,
		RequireAuthentication: prodMode && !localMode, // Require auth in prod unless local override
		RequireTLS:            prodMode && !localMode, // Require TLS in prod unless local override
		AllowNoIsolation:      false,                  // Never allow NoIsolation in production
	}
}
