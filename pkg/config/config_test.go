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
				if cfg.Server.Port != 8080 {
					t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
				}
				if cfg.Server.ShutdownTimeout != 10*time.Second {
					t.Errorf("Server.ShutdownTimeout = %v, want 10s", cfg.Server.ShutdownTimeout)
				}
				if !cfg.Observability.EnableTelemetry {
					t.Error("Observability.EnableTelemetry = false, want true")
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
