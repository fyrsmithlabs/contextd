// Package telemetry provides OpenTelemetry instrumentation for contextd.
package telemetry

import (
	"fmt"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
)

// Config holds telemetry configuration.
type Config struct {
	Enabled     bool           `koanf:"enabled"`
	Endpoint    string         `koanf:"endpoint"`
	ServiceName string         `koanf:"service_name"`
	Sampling    SamplingConfig `koanf:"sampling"`
	Metrics     MetricsConfig  `koanf:"metrics"`
	Shutdown    ShutdownConfig `koanf:"shutdown"`
}

// SamplingConfig controls trace sampling behavior.
type SamplingConfig struct {
	Rate           float64 `koanf:"rate"`            // 0.0-1.0, default 1.0
	AlwaysOnErrors bool    `koanf:"always_on_errors"` // Always capture error traces
}

// MetricsConfig controls metrics export.
type MetricsConfig struct {
	Enabled        bool            `koanf:"enabled"`
	ExportInterval config.Duration `koanf:"export_interval"`
}

// ShutdownConfig controls graceful shutdown behavior.
type ShutdownConfig struct {
	Timeout config.Duration `koanf:"timeout"`
}

// NewDefaultConfig returns production-ready telemetry defaults.
// Telemetry is disabled by default for new users who don't have an OTEL collector.
// Set OTEL_ENABLE=true or configure telemetry in config.yaml to enable.
func NewDefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		Endpoint:    "localhost:4317",
		ServiceName: "contextd",
		Sampling: SamplingConfig{
			Rate:           1.0, // 100% in dev
			AlwaysOnErrors: true,
		},
		Metrics: MetricsConfig{
			Enabled:        true,
			ExportInterval: config.Duration(15 * time.Second),
		},
		Shutdown: ShutdownConfig{
			Timeout: config.Duration(5 * time.Second),
		},
	}
}

// Validate checks configuration for errors.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // No validation needed if disabled
	}

	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required when telemetry is enabled")
	}

	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required when telemetry is enabled")
	}

	if c.Sampling.Rate < 0 || c.Sampling.Rate > 1 {
		return fmt.Errorf("sampling.rate must be between 0 and 1, got %f", c.Sampling.Rate)
	}

	if c.Metrics.Enabled && c.Metrics.ExportInterval.Duration() <= 0 {
		return fmt.Errorf("metrics.export_interval must be positive when metrics enabled")
	}

	if c.Shutdown.Timeout.Duration() <= 0 {
		return fmt.Errorf("shutdown.timeout must be positive")
	}

	return nil
}
