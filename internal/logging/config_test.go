package logging

import (
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.Equal(t, zapcore.InfoLevel, cfg.Level)
	assert.Equal(t, "json", cfg.Format)
	assert.True(t, cfg.Output.Stdout)
	assert.False(t, cfg.Output.OTEL)
	assert.True(t, cfg.Sampling.Enabled)
	assert.Equal(t, time.Second, cfg.Sampling.Tick.Duration())
	assert.True(t, cfg.Redaction.Enabled)
	assert.True(t, cfg.Caller.Enabled)
	assert.Equal(t, 1, cfg.Caller.Skip)
	assert.Equal(t, zapcore.ErrorLevel, cfg.Stacktrace.Level)
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
			name: "invalid format",
			config: &Config{
				Level:  zapcore.InfoLevel,
				Format: "xml",
			},
			wantErr: true,
			errMsg:  "format must be 'json' or 'console'",
		},
		{
			name: "no output enabled",
			config: &Config{
				Level:  zapcore.InfoLevel,
				Format: "json",
				Output: OutputConfig{Stdout: false, OTEL: false},
			},
			wantErr: true,
			errMsg:  "at least one output must be enabled",
		},
		{
			name: "invalid sampling tick",
			config: &Config{
				Level:  zapcore.InfoLevel,
				Format: "json",
				Output: OutputConfig{Stdout: true},
				Sampling: SamplingConfig{
					Enabled: true,
					Tick:    config.Duration(0),
				},
			},
			wantErr: true,
			errMsg:  "sampling tick must be > 0",
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

func TestLevelSamplingConfig_Defaults(t *testing.T) {
	defaults := DefaultLevelSamplingConfig()

	// Trace: first 1, drop rest
	assert.Equal(t, 1, defaults[TraceLevel].Initial)
	assert.Equal(t, 0, defaults[TraceLevel].Thereafter)

	// Debug: first 10, drop rest
	assert.Equal(t, 10, defaults[zapcore.DebugLevel].Initial)
	assert.Equal(t, 0, defaults[zapcore.DebugLevel].Thereafter)

	// Info: first 100, then 1 every 10
	assert.Equal(t, 100, defaults[zapcore.InfoLevel].Initial)
	assert.Equal(t, 10, defaults[zapcore.InfoLevel].Thereafter)

	// Warn: first 100, then 1 every 100
	assert.Equal(t, 100, defaults[zapcore.WarnLevel].Initial)
	assert.Equal(t, 100, defaults[zapcore.WarnLevel].Thereafter)

	// Error+ never sampled (not in map)
	_, exists := defaults[zapcore.ErrorLevel]
	assert.False(t, exists)
}

func TestConfig_ValidateCallerSkip(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		skip    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid: caller disabled",
			enabled: false,
			skip:    -1,
			wantErr: false,
		},
		{
			name:    "valid: caller enabled with skip 0",
			enabled: true,
			skip:    0,
			wantErr: false,
		},
		{
			name:    "valid: caller enabled with skip 1",
			enabled: true,
			skip:    1,
			wantErr: false,
		},
		{
			name:    "invalid: caller enabled with negative skip",
			enabled: true,
			skip:    -1,
			wantErr: true,
			errMsg:  "caller skip must be >= 0",
		},
		{
			name:    "invalid: caller enabled with skip -5",
			enabled: true,
			skip:    -5,
			wantErr: true,
			errMsg:  "caller skip must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Level:  zapcore.InfoLevel,
				Format: "json",
				Output: OutputConfig{Stdout: true},
				Caller: CallerConfig{
					Enabled: tt.enabled,
					Skip:    tt.skip,
				},
			}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ValidateRedactionPattern(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		patterns []string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid: redaction disabled",
			enabled:  false,
			patterns: []string{"[invalid("},
			wantErr:  false,
		},
		{
			name:    "valid: redaction enabled with valid patterns",
			enabled: true,
			patterns: []string{
				`(?i)bearer\s+\S+`,
				`(?i)api[_-]?key[=:]\s*\S+`,
			},
			wantErr: false,
		},
		{
			name:     "valid: redaction enabled with no patterns",
			enabled:  true,
			patterns: []string{},
			wantErr:  false,
		},
		{
			name:     "invalid: unclosed bracket in pattern",
			enabled:  true,
			patterns: []string{"[invalid("},
			wantErr:  true,
			errMsg:   "invalid redaction pattern",
		},
		{
			name:     "invalid: bad regex syntax",
			enabled:  true,
			patterns: []string{"(?P<incomplete)"},
			wantErr:  true,
			errMsg:   "invalid redaction pattern",
		},
		{
			name:    "invalid: pattern too long",
			enabled: true,
			patterns: []string{
				string(make([]byte, 1001)),
			},
			wantErr: true,
			errMsg:  "pattern too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Level:  zapcore.InfoLevel,
				Format: "json",
				Output: OutputConfig{Stdout: true},
				Redaction: RedactionConfig{
					Enabled:  tt.enabled,
					Patterns: tt.patterns,
				},
			}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ValidateEmptyFieldKey(t *testing.T) {
	cfg := &Config{
		Level:  zapcore.InfoLevel,
		Format: "json",
		Output: OutputConfig{Stdout: true},
		Fields: map[string]string{"": "value"},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field key cannot be empty")
}

func TestConfig_ValidateEmptyFieldValue(t *testing.T) {
	cfg := &Config{
		Level:  zapcore.InfoLevel,
		Format: "json",
		Output: OutputConfig{Stdout: true},
		Fields: map[string]string{"key": ""},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty value")
}

func TestConfig_ValidateFieldsNil(t *testing.T) {
	cfg := &Config{
		Level:  zapcore.InfoLevel,
		Format: "json",
		Output: OutputConfig{Stdout: true},
		Fields: nil,
	}
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfig_ValidateValidFields(t *testing.T) {
	cfg := &Config{
		Level:  zapcore.InfoLevel,
		Format: "json",
		Output: OutputConfig{Stdout: true},
		Fields: map[string]string{
			"service":     "contextd",
			"environment": "production",
			"version":     "1.0.0",
		},
	}
	err := cfg.Validate()
	require.NoError(t, err)
}
