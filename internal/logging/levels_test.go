package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestTraceLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    zapcore.Level
		expected int8
	}{
		{"trace below debug", TraceLevel, -2},
		{"debug level", zapcore.DebugLevel, -1},
		{"trace enabled at trace", TraceLevel, -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int8(tt.level))
		})
	}
}

func TestTraceLevelRegistration(t *testing.T) {
	// Verify Trace level value
	level := TraceLevel
	assert.Equal(t, zapcore.Level(-2), level)
	// Note: Without zapcore.RegisterLevel (added in later Zap versions),
	// level.String() returns "Level(-2)" instead of "trace"
	assert.Contains(t, level.String(), "-2")
}

func TestTraceLevelEnabler(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   zapcore.Level
		logLevel      zapcore.Level
		shouldBeLogged bool
	}{
		{"trace logged when trace enabled", TraceLevel, TraceLevel, true},
		{"debug logged when trace enabled", TraceLevel, zapcore.DebugLevel, true},
		{"trace not logged when debug enabled", zapcore.DebugLevel, TraceLevel, false},
		{"debug logged when debug enabled", zapcore.DebugLevel, zapcore.DebugLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled := tt.configLevel.Enabled(tt.logLevel)
			assert.Equal(t, tt.shouldBeLogged, enabled)
		})
	}
}

func TestLevelFromString_ValidLevels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected zapcore.Level
	}{
		{"trace", "trace", TraceLevel},
		{"debug", "debug", zapcore.DebugLevel},
		{"info", "info", zapcore.InfoLevel},
		{"warn", "warn", zapcore.WarnLevel},
		{"error", "error", zapcore.ErrorLevel},
		{"dpanic", "dpanic", zapcore.DPanicLevel},
		{"panic", "panic", zapcore.PanicLevel},
		{"fatal", "fatal", zapcore.FatalLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := LevelFromString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestLevelFromString_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected zapcore.Level
	}{
		{"uppercase", "INFO", zapcore.InfoLevel},
		{"mixed case", "InFo", zapcore.InfoLevel},
		{"Debug uppercase", "DEBUG", zapcore.DebugLevel},
		{"Error mixed", "ErRoR", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := LevelFromString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestLevelFromString_EmptyString(t *testing.T) {
	// Empty string defaults to info without error (zap behavior)
	level, err := LevelFromString("")
	assert.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, level)
}

func TestLevelFromString_InvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid level", "invalid"},
		{"numeric", "123"},
		{"extra text", "info extra"},
		{"special chars", "info@123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := LevelFromString(tt.input)
			assert.Error(t, err)
			// On error, should return InfoLevel as default
			assert.Equal(t, zapcore.InfoLevel, level)
		})
	}
}
