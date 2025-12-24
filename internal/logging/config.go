// internal/logging/config.go
package logging

import (
	"fmt"
	"regexp"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"go.uber.org/zap/zapcore"
)

// Config holds logging configuration.
type Config struct {
	Level      zapcore.Level     `koanf:"level"`
	Format     string            `koanf:"format"`
	Output     OutputConfig      `koanf:"output"`
	Sampling   SamplingConfig    `koanf:"sampling"`
	Caller     CallerConfig      `koanf:"caller"`
	Stacktrace StacktraceConfig  `koanf:"stacktrace"`
	Fields     map[string]string `koanf:"fields"`
	Redaction  RedactionConfig   `koanf:"redaction"`
}

// OutputConfig controls where logs are written.
type OutputConfig struct {
	Stdout bool `koanf:"stdout"`
	OTEL   bool `koanf:"otel"`
}

// SamplingConfig controls log volume reduction.
type SamplingConfig struct {
	Enabled bool            `koanf:"enabled"`
	Tick    config.Duration `koanf:"tick"`
	Levels  map[zapcore.Level]LevelSamplingConfig `koanf:"levels"`
}

// LevelSamplingConfig defines sampling rate per level.
type LevelSamplingConfig struct {
	Initial    int `koanf:"initial"`
	Thereafter int `koanf:"thereafter"`
}

// CallerConfig controls caller information in logs.
type CallerConfig struct {
	Enabled bool `koanf:"enabled"`
	Skip    int  `koanf:"skip"`
}

// StacktraceConfig controls stacktrace inclusion.
type StacktraceConfig struct {
	Level zapcore.Level `koanf:"level"`
}

// RedactionConfig controls sensitive data redaction.
type RedactionConfig struct {
	Enabled  bool     `koanf:"enabled"`
	Fields   []string `koanf:"fields"`
	Patterns []string `koanf:"patterns"`
}

// NewDefaultConfig returns config with production-ready defaults.
func NewDefaultConfig() *Config {
	return &Config{
		Level:  zapcore.InfoLevel,
		Format: "json",
		Output: OutputConfig{
			Stdout: true,
			OTEL:   false,
		},
		Sampling: SamplingConfig{
			Enabled: true,
			Tick:    config.Duration(time.Second),
			Levels:  DefaultLevelSamplingConfig(),
		},
		Caller: CallerConfig{
			Enabled: true,
			Skip:    1,
		},
		Stacktrace: StacktraceConfig{
			Level: zapcore.ErrorLevel,
		},
		Fields: map[string]string{
			"service": "contextd",
		},
		Redaction: RedactionConfig{
			Enabled: true,
			Fields: []string{
				"password", "secret", "token", "api_key",
				"authorization", "bearer", "credential", "private_key",
			},
			Patterns: []string{
				`(?i)bearer\s+\S+`,
				`(?i)api[_-]?key[=:]\s*\S+`,
			},
		},
	}
}

// DefaultLevelSamplingConfig returns default sampling config by level.
func DefaultLevelSamplingConfig() map[zapcore.Level]LevelSamplingConfig {
	return map[zapcore.Level]LevelSamplingConfig{
		TraceLevel: {Initial: 1, Thereafter: 0},
		zapcore.DebugLevel: {Initial: 10, Thereafter: 0},
		zapcore.InfoLevel: {Initial: 100, Thereafter: 10},
		zapcore.WarnLevel: {Initial: 100, Thereafter: 100},
		// Error+ never sampled
	}
}

// Validate checks config for errors.
func (c *Config) Validate() error {
	if c.Format != "json" && c.Format != "console" {
		return fmt.Errorf("format must be 'json' or 'console', got %q", c.Format)
	}
	if !c.Output.Stdout && !c.Output.OTEL {
		return fmt.Errorf("at least one output must be enabled (stdout or otel)")
	}
	if c.Sampling.Enabled && c.Sampling.Tick.Duration() <= 0 {
		return fmt.Errorf("sampling tick must be > 0 when sampling enabled")
	}

	// Validate Caller config
	if c.Caller.Enabled && c.Caller.Skip < 0 {
		return fmt.Errorf("caller skip must be >= 0, got %d", c.Caller.Skip)
	}

	// Validate redaction patterns (compile to check validity)
	if c.Redaction.Enabled {
		for _, pattern := range c.Redaction.Patterns {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("invalid redaction pattern %q: %w", pattern, err)
			}
			if len(pattern) > 1000 {
				return fmt.Errorf("redaction pattern too long (max 1000 chars): %q", pattern)
			}
		}
	}

	// Validate constant fields
	if c.Fields != nil {
		for k, v := range c.Fields {
			if k == "" {
				return fmt.Errorf("field key cannot be empty")
			}
			if v == "" {
				return fmt.Errorf("field %q has empty value", k)
			}
		}
	}

	return nil
}
