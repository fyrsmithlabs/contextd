package telemetry

import (
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.False(t, cfg.Enabled) // Disabled by default for new users without OTEL collector
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.Equal(t, "contextd", cfg.ServiceName)
	assert.Equal(t, "0.1.0", cfg.ServiceVersion)
	assert.True(t, cfg.Insecure) // Insecure by default for local dev
	assert.Equal(t, 1.0, cfg.Sampling.Rate)
	assert.True(t, cfg.Sampling.AlwaysOnErrors)
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, 15*time.Second, cfg.Metrics.ExportInterval.Duration())
	assert.Equal(t, 5*time.Second, cfg.Shutdown.Timeout.Duration())
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  NewDefaultConfig(),
			wantErr: false,
		},
		{
			name: "disabled config skips validation",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: &Config{
				Enabled:        true,
				Endpoint:       "",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "missing service name",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "",
				ServiceVersion: "0.1.0",
			},
			wantErr: true,
			errMsg:  "service_name is required",
		},
		{
			name: "missing service version",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "",
			},
			wantErr: true,
			errMsg:  "service_version is required",
		},
		{
			name: "sampling rate too low",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Sampling:       SamplingConfig{Rate: -0.1},
				Shutdown:       ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: true,
			errMsg:  "sampling.rate must be between 0 and 1",
		},
		{
			name: "sampling rate too high",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Sampling:       SamplingConfig{Rate: 1.1},
				Shutdown:       ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: true,
			errMsg:  "sampling.rate must be between 0 and 1",
		},
		{
			name: "invalid metrics export interval",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Sampling:       SamplingConfig{Rate: 1.0},
				Metrics: MetricsConfig{
					Enabled:        true,
					ExportInterval: config.Duration(0),
				},
				Shutdown: ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: true,
			errMsg:  "metrics.export_interval must be positive",
		},
		{
			name: "invalid shutdown timeout",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Sampling:       SamplingConfig{Rate: 1.0},
				Metrics:     MetricsConfig{Enabled: false},
				Shutdown:    ShutdownConfig{Timeout: config.Duration(0)},
			},
			wantErr: true,
			errMsg:  "shutdown.timeout must be positive",
		},
		{
			name: "valid with custom values and TLS",
			config: &Config{
				Enabled:        true,
				Endpoint:       "collector.prod:4317",
				ServiceName:    "my-service",
				ServiceVersion: "1.2.3",
				Insecure:       false, // TLS enabled for remote endpoint
				Sampling: SamplingConfig{
					Rate:           0.5,
					AlwaysOnErrors: true,
				},
				Metrics: MetricsConfig{
					Enabled:        true,
					ExportInterval: config.Duration(30 * time.Second),
				},
				Shutdown: ShutdownConfig{
					Timeout: config.Duration(10 * time.Second),
				},
			},
			wantErr: false,
		},
		{
			name: "insecure allowed for localhost",
			config: &Config{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Insecure:       true,
				Sampling:       SamplingConfig{Rate: 1.0},
				Metrics:        MetricsConfig{Enabled: false},
				Shutdown:       ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: false,
		},
		{
			name: "insecure allowed for 127.0.0.1",
			config: &Config{
				Enabled:        true,
				Endpoint:       "127.0.0.1:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Insecure:       true,
				Sampling:       SamplingConfig{Rate: 1.0},
				Metrics:        MetricsConfig{Enabled: false},
				Shutdown:       ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: false,
		},
		{
			name: "insecure not allowed for remote endpoint",
			config: &Config{
				Enabled:        true,
				Endpoint:       "collector.prod:4317",
				ServiceName:    "test",
				ServiceVersion: "0.1.0",
				Insecure:       true, // Security violation: insecure to remote
				Sampling:       SamplingConfig{Rate: 1.0},
				Metrics:        MetricsConfig{Enabled: false},
				Shutdown:       ShutdownConfig{Timeout: config.Duration(time.Second)},
			},
			wantErr: true,
			errMsg:  "insecure connections to remote endpoints are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_IsLocalEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		isLocal  bool
	}{
		{"localhost:4317", true},
		{"localhost", true},
		{"127.0.0.1:4317", true},
		{"127.0.0.1", true},
		{"127.0.1.1:4317", true},
		{"::1:4317", true},
		{"::1", true},
		{"collector.prod:4317", false},
		{"otel.example.com:4317", false},
		{"192.168.1.1:4317", false},
		{"10.0.0.1:4317", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			cfg := &Config{Endpoint: tt.endpoint}
			assert.Equal(t, tt.isLocal, cfg.isLocalEndpoint())
		})
	}
}

func TestConfig_SamplingEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		rate    float64
		wantErr bool
	}{
		{"zero sampling", 0.0, false},
		{"full sampling", 1.0, false},
		{"half sampling", 0.5, false},
		{"tiny sampling", 0.001, false},
		{"almost full", 0.999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.Sampling.Rate = tt.rate

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
