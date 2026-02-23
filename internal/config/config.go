// Package config provides configuration loading for contextd v2.
//
// Configuration is loaded from environment variables with sensible defaults.
// This package supports server, observability, and application-specific settings.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds the complete contextd v2 configuration.
type Config struct {
	Production             ProductionConfig
	Server                 ServerConfig
	Observability          ObservabilityConfig
	PreFetch               PreFetchConfig
	Checkpoint             CheckpointConfig
	VectorStore            VectorStoreConfig
	Qdrant                 QdrantConfig
	Embeddings             EmbeddingsConfig
	Repository             RepositoryConfig
	Statusline             StatuslineConfig
	ConsolidationScheduler ConsolidationSchedulerConfig
	ReasoningBank          ReasoningBankConfig
	Fallback               FallbackConfig
}

// StatuslineConfig holds statusline display configuration.
type StatuslineConfig struct {
	Enabled    bool                 `koanf:"enabled"`
	Endpoint   string               `koanf:"endpoint"` // HTTP endpoint for status
	Show       StatuslineShowConfig `koanf:"show"`
	Thresholds StatuslineThresholds `koanf:"thresholds"`
}

// StatuslineShowConfig controls which items to display.
type StatuslineShowConfig struct {
	Service     bool `koanf:"service"`     // 🟢/🟡/🔴
	Memories    bool `koanf:"memories"`    // 🧠12
	Checkpoints bool `koanf:"checkpoints"` // 💾3
	Context     bool `koanf:"context"`     // 📊68%
	Confidence  bool `koanf:"confidence"`  // C:.85
	Compression bool `koanf:"compression"` // F:2.1x
}

// StatuslineThresholds controls warning thresholds.
type StatuslineThresholds struct {
	ContextWarning  int `koanf:"context_warning"`  // Yellow threshold (default: 70)
	ContextCritical int `koanf:"context_critical"` // Red threshold (default: 85)
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

// VectorStoreConfig holds vectorstore provider configuration.
type VectorStoreConfig struct {
	Provider string         `koanf:"provider"` // "chromem" or "qdrant" (default: "chromem")
	Chromem  ChromemConfig  `koanf:"chromem"`
	Fallback FallbackConfig `koanf:"fallback"`
}

// Validate validates VectorStoreConfig.
func (c *VectorStoreConfig) Validate() error {
	switch c.Provider {
	case "chromem":
		return c.Chromem.Validate()
	case "qdrant":
		// Qdrant validation handled elsewhere
		return nil
	default:
		return fmt.Errorf("unsupported provider: %s (supported: chromem, qdrant)", c.Provider)
	}
}

// ChromemConfig holds chromem-go embedded vector database configuration.
// chromem-go is a pure Go, embedded vector database with zero third-party dependencies.
type ChromemConfig struct {
	// Path is the directory for persistent storage.
	// Default: "~/.config/contextd/vectorstore"
	Path string `koanf:"path"`

	// Compress enables gzip compression for stored data.
	// Default: true
	Compress bool `koanf:"compress"`

	// DefaultCollection is the default collection name.
	// Default: "contextd_default"
	DefaultCollection string `koanf:"default_collection"`

	// VectorSize is the expected embedding dimension.
	// Must match the embedder's output dimension.
	// Default: 384 (for FastEmbed bge-small-en-v1.5)
	VectorSize int `koanf:"vector_size"`
}

// FallbackConfig holds configuration for fallback storage.
type FallbackConfig struct {
	// Enabled enables fallback storage (default: false).
	Enabled bool `koanf:"enabled"`

	// LocalPath is the path for local fallback storage.
	// Default: .claude/contextd/store
	LocalPath string `koanf:"local_path"`

	// SyncOnConnect triggers immediate sync when remote becomes available (default: true).
	SyncOnConnect bool `koanf:"sync_on_connect"`

	// HealthCheckInterval is the interval for periodic health checks (default: 30s).
	HealthCheckInterval string `koanf:"health_check_interval"`

	// WALPath is the directory for write-ahead log.
	// Default: .claude/contextd/wal
	WALPath string `koanf:"wal_path"`

	// WALRetentionDays is how long to keep synced entries in WAL (default: 7).
	WALRetentionDays int `koanf:"wal_retention_days"`
}

// Validate validates ChromemConfig.
func (c *ChromemConfig) Validate() error {
	if c.VectorSize <= 0 {
		return fmt.Errorf("vector_size must be positive, got %d", c.VectorSize)
	}
	return nil
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
	Provider    string `koanf:"provider"` // "fastembed" or "tei"
	BaseURL     string `koanf:"base_url"` // TEI URL (if using TEI)
	Model       string `koanf:"model"`
	CacheDir    string `koanf:"cache_dir"`    // Model cache directory (for fastembed)
	ONNXVersion string `koanf:"onnx_version"` // Optional ONNX runtime version override
}

// CheckpointConfig holds checkpoint service configuration.
type CheckpointConfig struct {
	MaxContentSizeKB int `koanf:"max_content_size_kb"` // Maximum content size in KB (default: 1024 = 1MB)
}

// ReasoningBankConfig holds memory granularity and buffering configuration.
type ReasoningBankConfig struct {
	// Granularity controls memory storage granularity.
	// "turn" (default): store each memory individually on record.
	// "session": buffer turns and flush as session summary on session end.
	Granularity string `koanf:"granularity"`

	// MaxBufferedTurns is the maximum number of turns to buffer per session.
	// When exceeded, oldest turns are dropped. Default: 500.
	MaxBufferedTurns int `koanf:"max_buffered_turns"`
}

// ConsolidationSchedulerConfig holds automatic memory consolidation configuration.
type ConsolidationSchedulerConfig struct {
	Enabled             bool          `koanf:"enabled"`              // Enable automatic consolidation (default: false)
	Interval            time.Duration `koanf:"interval"`             // Time between consolidation runs (default: 24h)
	SimilarityThreshold float64       `koanf:"similarity_threshold"` // Similarity threshold for consolidation (default: 0.8)
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port            int           `koanf:"http_port"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

// ObservabilityConfig holds OpenTelemetry configuration.
type ObservabilityConfig struct {
	EnableTelemetry   bool   `koanf:"enable_telemetry"`
	ServiceName       string `koanf:"service_name"`
	OTLPEndpoint      string `koanf:"otlp_endpoint"`        // OTLP endpoint (default: localhost:4317)
	OTLPProtocol      string `koanf:"otlp_protocol"`        // "grpc" or "http/protobuf" (default: grpc)
	OTLPInsecure      bool   `koanf:"otlp_insecure"`        // Use insecure connection (default: true for localhost)
	OTLPTLSSkipVerify bool   `koanf:"otlp_tls_skip_verify"` // Skip TLS verification for internal CAs
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
// Quick Start - Most commonly configured env vars:
//
//   - CONTEXTD_DATA_PATH: Base data path (default: /data)
//   - EMBEDDINGS_PROVIDER: fastembed (default, local) or tei (remote)
//   - EMBEDDINGS_CACHE_DIR: Model cache directory (default: ./local_cache)
//   - VECTORSTORE_PROVIDER: chromem (default, embedded) or qdrant (external)
//   - CHECKPOINT_MAX_CONTENT_SIZE_KB: Max checkpoint size in KB (default: 1024)
//   - CONTEXTD_PRODUCTION_MODE: Enable production safety checks (default: false)
//
// All environment variables:
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
//   - EMBEDDINGS_CACHE_DIR: Model cache directory for fastembed (default: ./local_cache)
//
// Checkpoint:
//   - CHECKPOINT_MAX_CONTENT_SIZE_KB: Max checkpoint size in KB (default: 1024)
//
// Consolidation Scheduler:
//   - CONSOLIDATION_SCHEDULER_ENABLED: Enable automatic consolidation (default: false)
//   - CONSOLIDATION_SCHEDULER_INTERVAL: Time between runs (default: 24h)
//   - CONSOLIDATION_SCHEDULER_SIMILARITY_THRESHOLD: Similarity threshold (default: 0.8)
//
// Telemetry:
//   - OTEL_ENABLE: Enable OpenTelemetry (default: false, requires OTEL collector)
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
		Production: ProductionConfig{
			Enabled:               getEnvBool("CONTEXTD_PRODUCTION_MODE", false),
			LocalModeAcknowledged: getEnvBool("CONTEXTD_LOCAL_MODE", false),
			RequireAuthentication: getEnvBool("CONTEXTD_REQUIRE_AUTH", false),
			RequireTLS:            getEnvBool("CONTEXTD_REQUIRE_TLS", false),
			AllowNoIsolation:      getEnvBool("CONTEXTD_ALLOW_NO_ISOLATION", false),
		},
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 9090),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Observability: ObservabilityConfig{
			EnableTelemetry: getEnvBool("OTEL_ENABLE", false),
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

	// Consolidation Scheduler configuration
	cfg.ConsolidationScheduler = ConsolidationSchedulerConfig{
		Enabled:             getEnvBool("CONSOLIDATION_SCHEDULER_ENABLED", false),             // Default: disabled
		Interval:            getEnvDuration("CONSOLIDATION_SCHEDULER_INTERVAL", 24*time.Hour), // Default: 24h
		SimilarityThreshold: getEnvFloat("CONSOLIDATION_SCHEDULER_SIMILARITY_THRESHOLD", 0.8), // Default: 0.8
	}

	// ReasoningBank configuration
	cfg.ReasoningBank = ReasoningBankConfig{
		Granularity:      getEnvString("CONTEXTD_REASONINGBANK_GRANULARITY", "turn"),
		MaxBufferedTurns: getEnvInt("CONTEXTD_REASONINGBANK_MAX_BUFFERED_TURNS", 500),
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
		Provider:    getEnvString("EMBEDDINGS_PROVIDER", "fastembed"),
		BaseURL:     getEnvString("EMBEDDING_BASE_URL", "http://localhost:8080"),
		Model:       getEnvString("EMBEDDINGS_MODEL", "BAAI/bge-small-en-v1.5"),
		CacheDir:    getEnvString("EMBEDDINGS_CACHE_DIR", ""),
		ONNXVersion: getEnvString("EMBEDDINGS_ONNX_VERSION", ""),
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

	// VectorStore configuration (chromem is default - embedded, no external deps)
	cfg.VectorStore = VectorStoreConfig{
		Provider: getEnvString("CONTEXTD_VECTORSTORE_PROVIDER", "chromem"),
		Chromem: ChromemConfig{
			Path:              getEnvString("CONTEXTD_VECTORSTORE_CHROMEM_PATH", "~/.config/contextd/vectorstore"),
			Compress:          getEnvBool("CONTEXTD_VECTORSTORE_CHROMEM_COMPRESS", false),
			DefaultCollection: getEnvString("CONTEXTD_VECTORSTORE_CHROMEM_COLLECTION", "contextd_default"),
			VectorSize:        getEnvInt("CONTEXTD_VECTORSTORE_CHROMEM_VECTOR_SIZE", 384),
		},
	}

	// Statusline configuration
	cfg.Statusline = StatuslineConfig{
		Enabled:  getEnvBool("CONTEXTD_STATUSLINE_ENABLED", true),
		Endpoint: getEnvString("CONTEXTD_STATUSLINE_ENDPOINT", "http://localhost:9090"),
		Show: StatuslineShowConfig{
			Service:     getEnvBool("CONTEXTD_STATUSLINE_SHOW_SERVICE", true),
			Memories:    getEnvBool("CONTEXTD_STATUSLINE_SHOW_MEMORIES", true),
			Checkpoints: getEnvBool("CONTEXTD_STATUSLINE_SHOW_CHECKPOINTS", true),
			Context:     getEnvBool("CONTEXTD_STATUSLINE_SHOW_CONTEXT", true),
			Confidence:  getEnvBool("CONTEXTD_STATUSLINE_SHOW_CONFIDENCE", true),
			Compression: getEnvBool("CONTEXTD_STATUSLINE_SHOW_COMPRESSION", true),
		},
		Thresholds: StatuslineThresholds{
			ContextWarning:  getEnvInt("CONTEXTD_STATUSLINE_CONTEXT_WARNING", 70),
			ContextCritical: getEnvInt("CONTEXTD_STATUSLINE_CONTEXT_CRITICAL", 85),
		},
	}

	// Fallback storage configuration
	cfg.Fallback = FallbackConfig{
		Enabled:             getEnvBool("CONTEXTD_FALLBACK_ENABLED", false),
		LocalPath:           getEnvString("CONTEXTD_FALLBACK_LOCAL_PATH", ".claude/contextd/store"),
		SyncOnConnect:       getEnvBool("CONTEXTD_FALLBACK_SYNC_ON_CONNECT", true),
		HealthCheckInterval: getEnvString("CONTEXTD_FALLBACK_HEALTH_INTERVAL", "30s"),
		WALPath:             getEnvString("CONTEXTD_FALLBACK_WAL_PATH", ".claude/contextd/wal"),
		WALRetentionDays:    getEnvInt("CONTEXTD_FALLBACK_WAL_RETENTION_DAYS", 7),
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

	// Validate environment variable inputs
	if err := validateHostname(c.Qdrant.Host); err != nil {
		return fmt.Errorf("invalid QDRANT_HOST: %w", err)
	}

	if err := validatePath(c.Qdrant.DataPath); err != nil {
		return fmt.Errorf("invalid CONTEXTD_DATA_PATH: %w", err)
	}

	if err := validatePath(c.VectorStore.Chromem.Path); err != nil {
		return fmt.Errorf("invalid CONTEXTD_VECTORSTORE_CHROMEM_PATH: %w", err)
	}

	if c.Embeddings.CacheDir != "" {
		if err := validatePath(c.Embeddings.CacheDir); err != nil {
			return fmt.Errorf("invalid EMBEDDINGS_CACHE_DIR: %w", err)
		}
	}

	if c.Embeddings.BaseURL != "" {
		if err := validateURL(c.Embeddings.BaseURL); err != nil {
			return fmt.Errorf("invalid EMBEDDING_BASE_URL: %w", err)
		}
	}

	// Validate production configuration
	if err := c.Production.Validate(); err != nil {
		return fmt.Errorf("production config validation failed: %w", err)
	}

	// Validate ReasoningBank configuration
	switch c.ReasoningBank.Granularity {
	case "turn", "session":
		// Valid
	case "":
		// Empty defaults to "turn" at runtime
	default:
		return fmt.Errorf("invalid CONTEXTD_REASONINGBANK_GRANULARITY: %q (must be 'turn' or 'session')", c.ReasoningBank.Granularity)
	}
	if c.ReasoningBank.MaxBufferedTurns < 0 {
		return fmt.Errorf("CONTEXTD_REASONINGBANK_MAX_BUFFERED_TURNS must be non-negative, got %d", c.ReasoningBank.MaxBufferedTurns)
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
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

// ProductionConfig holds production deployment configuration.
type ProductionConfig struct {
	// Enabled indicates whether production mode is active.
	// Set via CONTEXTD_PRODUCTION_MODE=1 environment variable.
	Enabled bool `koanf:"enabled"`

	// LocalModeAcknowledged allows development features in production mode.
	// Set via CONTEXTD_LOCAL_MODE=1 environment variable.
	// Use only for local development/testing.
	LocalModeAcknowledged bool `koanf:"local_mode_acknowledged"`

	// RequireAuthentication enforces authentication in production.
	RequireAuthentication bool `koanf:"require_authentication"`

	// AuthenticationConfigured indicates if auth is properly set up.
	AuthenticationConfigured bool `koanf:"authentication_configured"`

	// RequireTLS enforces TLS for external services (Qdrant, OTEL).
	RequireTLS bool `koanf:"require_tls"`

	// AllowNoIsolation permits NoIsolation mode (testing only).
	// Always false in production mode.
	AllowNoIsolation bool `koanf:"allow_no_isolation"`
}

// IsProduction returns true if running in production mode.
func (c *ProductionConfig) IsProduction() bool {
	return c.Enabled
}

// IsLocal returns true if local mode is acknowledged.
func (c *ProductionConfig) IsLocal() bool {
	return c.LocalModeAcknowledged
}

// Validate checks production configuration for security issues.
func (c *ProductionConfig) Validate() error {
	if !c.Enabled {
		return nil // Not in production, skip validation
	}

	if c.AllowNoIsolation {
		return fmt.Errorf("SECURITY: NoIsolation mode cannot be enabled in production")
	}

	if c.RequireAuthentication && !c.AuthenticationConfigured {
		return fmt.Errorf("SECURITY: RequireAuthentication enabled but authentication not configured")
	}

	return nil
}

// validateHostname checks if a hostname is safe (no command injection attempts).
// Uses positive validation with net.ParseIP for IP addresses and regexp for hostnames.
func validateHostname(host string) error {
	// Empty hostname is allowed (config may use defaults)
	if host == "" {
		return nil
	}

	// Try parsing as IP first
	if net.ParseIP(host) != nil {
		return nil // Valid IP address
	}

	// Validate hostname format (RFC 1123)
	// Allow alphanumeric, dots, hyphens. Must not start/end with dash.
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(host) {
		return fmt.Errorf("invalid hostname format: %s", host)
	}

	// Additional blacklist check for shell metacharacters (defense in depth)
	invalidChars := []string{";", "\n", "\r", "$", "`", "|", "&", "<", ">", "(", ")"}
	for _, char := range invalidChars {
		if strings.Contains(host, char) {
			return fmt.Errorf("invalid hostname: contains forbidden character %q", char)
		}
	}
	return nil
}

// validatePath checks if a path is safe (no path traversal)
func validatePath(path string) error {
	// Check for path traversal sequences
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains traversal sequence: %s", path)
	}

	// For absolute paths, verify the cleaned path doesn't escape
	if filepath.IsAbs(path) {
		clean := filepath.Clean(path)
		// Count directory depth - compare original vs cleaned
		// If cleaned has fewer separators, upward traversal occurred
		origDepth := strings.Count(path, string(filepath.Separator))
		cleanDepth := strings.Count(clean, string(filepath.Separator))

		if cleanDepth < origDepth-1 {
			return fmt.Errorf("path traversal detected: %s (resolves to %s)", path, clean)
		}
	}

	return nil
}

// validateURL checks if a URL uses allowed schemes (http/https only)
func validateURL(urlStr string) error {
	// Only allow http and https schemes
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("URL must use http:// or https:// scheme, got: %s", urlStr)
	}
	return nil
}
