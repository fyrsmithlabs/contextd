package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment and restore after test
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name: "default values",
			env:  map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 9090 {
					t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
				}
				if cfg.Server.ShutdownTimeout != 10*time.Second {
					t.Errorf("Server.ShutdownTimeout = %v, want 10s", cfg.Server.ShutdownTimeout)
				}
				if cfg.Observability.EnableTelemetry {
					t.Error("Observability.EnableTelemetry = true, want false (disabled by default)")
				}
				if cfg.Observability.ServiceName != "contextd" {
					t.Errorf("Observability.ServiceName = %q, want contextd", cfg.Observability.ServiceName)
				}
				// Prefetch defaults
				if !cfg.PreFetch.Enabled {
					t.Error("PreFetch.Enabled = false, want true")
				}
				if cfg.PreFetch.CacheTTL != 5*time.Minute {
					t.Errorf("PreFetch.CacheTTL = %v, want 5m", cfg.PreFetch.CacheTTL)
				}
				if cfg.PreFetch.CacheMaxEntries != 100 {
					t.Errorf("PreFetch.CacheMaxEntries = %d, want 100", cfg.PreFetch.CacheMaxEntries)
				}
				if !cfg.PreFetch.Rules.BranchDiff.Enabled {
					t.Error("PreFetch.Rules.BranchDiff.Enabled = false, want true")
				}
			},
		},
		{
			name: "environment variable overrides",
			env: map[string]string{
				"SERVER_PORT":             "9090",
				"SERVER_SHUTDOWN_TIMEOUT": "5s",
				"OTEL_ENABLE":             "false",
				"OTEL_SERVICE_NAME":       "test-service",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 9090 {
					t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
				}
				if cfg.Server.ShutdownTimeout != 5*time.Second {
					t.Errorf("Server.ShutdownTimeout = %v, want 5s", cfg.Server.ShutdownTimeout)
				}
				if cfg.Observability.EnableTelemetry {
					t.Error("Observability.EnableTelemetry = true, want false")
				}
				if cfg.Observability.ServiceName != "test-service" {
					t.Errorf("Observability.ServiceName = %q, want test-service", cfg.Observability.ServiceName)
				}
			},
		},
		{
			name: "prefetch environment overrides",
			env: map[string]string{
				"PREFETCH_ENABLED":             "false",
				"PREFETCH_CACHE_TTL":           "10m",
				"PREFETCH_CACHE_MAX_ENTRIES":   "50",
				"PREFETCH_BRANCH_DIFF_ENABLED": "false",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.PreFetch.Enabled {
					t.Error("PreFetch.Enabled = true, want false")
				}
				if cfg.PreFetch.CacheTTL != 10*time.Minute {
					t.Errorf("PreFetch.CacheTTL = %v, want 10m", cfg.PreFetch.CacheTTL)
				}
				if cfg.PreFetch.CacheMaxEntries != 50 {
					t.Errorf("PreFetch.CacheMaxEntries = %d, want 50", cfg.PreFetch.CacheMaxEntries)
				}
				if cfg.PreFetch.Rules.BranchDiff.Enabled {
					t.Error("PreFetch.Rules.BranchDiff.Enabled = true, want false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg := Load()
			if cfg == nil {
				t.Fatal("Load() returned nil")
			}

			tt.validate(t, cfg)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Server: ServerConfig{
					Port:            8080,
					ShutdownTimeout: 10 * time.Second,
				},
				Observability: ObservabilityConfig{
					EnableTelemetry: true,
					ServiceName:     "contextd",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			cfg: &Config{
				Server: ServerConfig{
					Port:            0,
					ShutdownTimeout: 10 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &Config{
				Server: ServerConfig{
					Port:            70000,
					ShutdownTimeout: 10 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid shutdown timeout",
			cfg: &Config{
				Server: ServerConfig{
					Port:            8080,
					ShutdownTimeout: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "empty service name",
			cfg: &Config{
				Server: ServerConfig{
					Port:            8080,
					ShutdownTimeout: 10 * time.Second,
				},
				Observability: ObservabilityConfig{
					EnableTelemetry: true,
					ServiceName:     "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoad_VectorStoreConfig tests VectorStore configuration loading
func TestLoad_VectorStoreConfig(t *testing.T) {
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name: "vectorstore defaults - chromem provider with 384d",
			env:  map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				// Default provider should be chromem
				if cfg.VectorStore.Provider != "chromem" {
					t.Errorf("VectorStore.Provider = %q, want chromem", cfg.VectorStore.Provider)
				}
				// Default path
				if cfg.VectorStore.Chromem.Path != "~/.config/contextd/vectorstore" {
					t.Errorf("VectorStore.Chromem.Path = %q, want ~/.config/contextd/vectorstore", cfg.VectorStore.Chromem.Path)
				}
				// Default compress (false to match existing uncompressed data)
				if cfg.VectorStore.Chromem.Compress {
					t.Error("VectorStore.Chromem.Compress should be false by default")
				}
				// Default collection
				if cfg.VectorStore.Chromem.DefaultCollection != "contextd_default" {
					t.Errorf("VectorStore.Chromem.DefaultCollection = %q, want contextd_default", cfg.VectorStore.Chromem.DefaultCollection)
				}
				// Default vector size - 384 for FastEmbed
				if cfg.VectorStore.Chromem.VectorSize != 384 {
					t.Errorf("VectorStore.Chromem.VectorSize = %d, want 384", cfg.VectorStore.Chromem.VectorSize)
				}
			},
		},
		{
			name: "vectorstore environment overrides",
			env: map[string]string{
				"CONTEXTD_VECTORSTORE_PROVIDER":            "qdrant",
				"CONTEXTD_VECTORSTORE_CHROMEM_PATH":        "/custom/path/vectorstore",
				"CONTEXTD_VECTORSTORE_CHROMEM_COMPRESS":    "false",
				"CONTEXTD_VECTORSTORE_CHROMEM_COLLECTION":  "custom_collection",
				"CONTEXTD_VECTORSTORE_CHROMEM_VECTOR_SIZE": "768",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.VectorStore.Provider != "qdrant" {
					t.Errorf("VectorStore.Provider = %q, want qdrant", cfg.VectorStore.Provider)
				}
				if cfg.VectorStore.Chromem.Path != "/custom/path/vectorstore" {
					t.Errorf("VectorStore.Chromem.Path = %q, want /custom/path/vectorstore", cfg.VectorStore.Chromem.Path)
				}
				if cfg.VectorStore.Chromem.Compress {
					t.Error("VectorStore.Chromem.Compress should be false when overridden")
				}
				if cfg.VectorStore.Chromem.DefaultCollection != "custom_collection" {
					t.Errorf("VectorStore.Chromem.DefaultCollection = %q, want custom_collection", cfg.VectorStore.Chromem.DefaultCollection)
				}
				if cfg.VectorStore.Chromem.VectorSize != 768 {
					t.Errorf("VectorStore.Chromem.VectorSize = %d, want 768", cfg.VectorStore.Chromem.VectorSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg := Load()
			if cfg == nil {
				t.Fatal("Load() returned nil")
			}

			tt.validate(t, cfg)
		})
	}
}

// TestChromemConfig_Validate tests ChromemConfig validation
func TestChromemConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ChromemConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - 384d",
			cfg: ChromemConfig{
				Path:              "~/.config/contextd/vectorstore",
				Compress:          true,
				DefaultCollection: "contextd_default",
				VectorSize:        384,
			},
			wantErr: false,
		},
		{
			name: "valid - 768d",
			cfg: ChromemConfig{
				Path:              "/custom/path",
				Compress:          false,
				DefaultCollection: "custom",
				VectorSize:        768,
			},
			wantErr: false,
		},
		{
			name: "invalid - zero vector size",
			cfg: ChromemConfig{
				Path:              "~/.config/contextd/vectorstore",
				DefaultCollection: "contextd_default",
				VectorSize:        0,
			},
			wantErr: true,
			errMsg:  "vector_size must be positive",
		},
		{
			name: "invalid - negative vector size",
			cfg: ChromemConfig{
				Path:              "~/.config/contextd/vectorstore",
				DefaultCollection: "contextd_default",
				VectorSize:        -1,
			},
			wantErr: true,
			errMsg:  "vector_size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestVectorStoreConfig_Validate tests VectorStoreConfig validation
func TestVectorStoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     VectorStoreConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid chromem config",
			cfg: VectorStoreConfig{
				Provider: "chromem",
				Chromem: ChromemConfig{
					Path:              "~/.config/contextd/vectorstore",
					Compress:          true,
					DefaultCollection: "contextd_default",
					VectorSize:        384,
				},
			},
			wantErr: false,
		},
		{
			name: "valid qdrant config",
			cfg: VectorStoreConfig{
				Provider: "qdrant",
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			cfg: VectorStoreConfig{
				Provider: "unknown",
			},
			wantErr: true,
			errMsg:  "unsupported provider",
		},
		{
			name: "chromem with invalid vector size",
			cfg: VectorStoreConfig{
				Provider: "chromem",
				Chromem: ChromemConfig{
					Path:              "~/.config/contextd/vectorstore",
					DefaultCollection: "contextd_default",
					VectorSize:        0, // Invalid
				},
			},
			wantErr: true,
			errMsg:  "vector_size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestLoad_EmbeddingsONNXVersion tests ONNX version configuration loading
func TestLoad_EmbeddingsONNXVersion(t *testing.T) {
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name: "onnx version default empty",
			env:  map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				// Default should be empty (uses DefaultONNXRuntimeVersion from embeddings)
				if cfg.Embeddings.ONNXVersion != "" {
					t.Errorf("Embeddings.ONNXVersion = %q, want empty string", cfg.Embeddings.ONNXVersion)
				}
			},
		},
		{
			name: "onnx version environment override",
			env: map[string]string{
				"EMBEDDINGS_ONNX_VERSION": "1.20.0",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Embeddings.ONNXVersion != "1.20.0" {
					t.Errorf("Embeddings.ONNXVersion = %q, want 1.20.0", cfg.Embeddings.ONNXVersion)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg := Load()
			if cfg == nil {
				t.Fatal("Load() returned nil")
			}

			tt.validate(t, cfg)
		})
	}
}

// TestLoad_ConsolidationScheduler tests consolidation scheduler configuration loading
func TestLoad_ConsolidationScheduler(t *testing.T) {
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name: "consolidation scheduler defaults",
			env:  map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				// Default should be disabled
				if cfg.ConsolidationScheduler.Enabled {
					t.Error("ConsolidationScheduler.Enabled = true, want false (disabled by default)")
				}
				// Default interval should be 24h
				if cfg.ConsolidationScheduler.Interval != 24*time.Hour {
					t.Errorf("ConsolidationScheduler.Interval = %v, want 24h", cfg.ConsolidationScheduler.Interval)
				}
				// Default threshold should be 0.8
				if cfg.ConsolidationScheduler.SimilarityThreshold != 0.8 {
					t.Errorf("ConsolidationScheduler.SimilarityThreshold = %v, want 0.8", cfg.ConsolidationScheduler.SimilarityThreshold)
				}
			},
		},
		{
			name: "consolidation scheduler environment overrides",
			env: map[string]string{
				"CONSOLIDATION_SCHEDULER_ENABLED":              "true",
				"CONSOLIDATION_SCHEDULER_INTERVAL":             "12h",
				"CONSOLIDATION_SCHEDULER_SIMILARITY_THRESHOLD": "0.85",
			},
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.ConsolidationScheduler.Enabled {
					t.Error("ConsolidationScheduler.Enabled = false, want true")
				}
				if cfg.ConsolidationScheduler.Interval != 12*time.Hour {
					t.Errorf("ConsolidationScheduler.Interval = %v, want 12h", cfg.ConsolidationScheduler.Interval)
				}
				if cfg.ConsolidationScheduler.SimilarityThreshold != 0.85 {
					t.Errorf("ConsolidationScheduler.SimilarityThreshold = %v, want 0.85", cfg.ConsolidationScheduler.SimilarityThreshold)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg := Load()
			if cfg == nil {
				t.Fatal("Load() returned nil")
			}

			tt.validate(t, cfg)
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper functions to save/restore environment
func saveEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		env[e] = os.Getenv(e)
	}
	return env
}

func restoreEnv(env map[string]string) {
	os.Clearenv()
	for k, v := range env {
		os.Setenv(k, v)
	}
}
