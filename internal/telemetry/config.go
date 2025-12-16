// Package telemetry provides OpenTelemetry instrumentation for contextd.
package telemetry

import (
	"fmt"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
)

// Config holds telemetry configuration.
type Config struct {
	Enabled        bool           `koanf:"enabled"`
	Endpoint       string         `koanf:"endpoint"`
	ServiceName    string         `koanf:"service_name"`
	ServiceVersion string         `koanf:"service_version"`
	Insecure       bool           `koanf:"insecure"` // Use insecure connection (no TLS)
	Sampling       SamplingConfig `koanf:"sampling"`
	Metrics        MetricsConfig  `koanf:"metrics"`
	Shutdown       ShutdownConfig `koanf:"shutdown"`
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
		Enabled:        false,
		Endpoint:       "localhost:4317",
		ServiceName:    "contextd",
		ServiceVersion: "0.1.0",
		Insecure:       true, // Insecure by default for local dev; set false for production TLS
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

	if c.ServiceVersion == "" {
		return fmt.Errorf("service_version is required when telemetry is enabled")
	}

	// Security: Prevent insecure connections to remote endpoints
	if c.Insecure && !c.isLocalEndpoint() {
		return fmt.Errorf("insecure connections to remote endpoints are not allowed; set insecure=false for TLS or use a local endpoint (localhost/127.0.0.1)")
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

// isLocalEndpoint checks if the endpoint is a local address.
func (c *Config) isLocalEndpoint() bool {
	host := c.Endpoint

	// Handle IPv6 addresses (may be bracketed like [::1]:4317)
	if strings.HasPrefix(host, "[") {
		// Bracketed IPv6: [::1]:4317
		if idx := strings.Index(host, "]:"); idx != -1 {
			host = host[1:idx] // Extract between [ and ]
		} else if strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1] // [::1] without port
		}
	} else if strings.Count(host, ":") == 1 {
		// IPv4 or hostname with port: localhost:4317
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
	}
	// For IPv6 without brackets (::1, ::1:4317), we check the full string

	// Check for common local addresses
	return host == "localhost" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		strings.HasPrefix(host, "127.") ||
		strings.HasPrefix(c.Endpoint, "::1")
}
