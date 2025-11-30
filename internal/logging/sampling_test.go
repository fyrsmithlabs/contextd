package logging

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewSampledCore_Disabled(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	cfg := SamplingConfig{Enabled: false}

	sampled := newSampledCore(core, cfg)

	// Should return original core unchanged
	assert.Equal(t, core, sampled)
}

func TestNewSampledCore_ErrorsNeverSampled(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	cfg := SamplingConfig{
		Enabled: true,
		Tick:    config.Duration(time.Second),
		Levels:  DefaultLevelSamplingConfig(),
	}

	sampled := newSampledCore(core, cfg)
	logger := &Logger{
		zap:    zap.New(sampled),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	// Log 100 errors (should never be sampled)
	for i := 0; i < 100; i++ {
		logger.Error(ctx, "error message")
	}

	logs := observed.FilterMessage("error message").All()
	assert.Equal(t, 100, len(logs), "all errors should be logged")
}

func TestNewSampledCore_InfoSampled(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	cfg := SamplingConfig{
		Enabled: true,
		Tick:    config.Duration(10 * time.Millisecond),
		Levels: map[zapcore.Level]LevelSamplingConfig{
			zapcore.InfoLevel: {Initial: 5, Thereafter: 0},
		},
	}

	sampled := newSampledCore(core, cfg)
	logger := &Logger{
		zap:    zap.New(sampled),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	// Log 20 info messages quickly
	for i := 0; i < 20; i++ {
		logger.Info(ctx, "info message")
	}

	// Should have ~5 (initial), rest dropped
	logs := observed.FilterMessage("info message").All()
	assert.LessOrEqual(t, len(logs), 7, "should sample info logs") // Allow some variance
	assert.GreaterOrEqual(t, len(logs), 3)
}

func TestLevelFilterCore_With(t *testing.T) {
	core, observed := observer.New(TraceLevel)

	// Create level filter that only allows Error and above
	filtered := &levelFilterCore{
		Core:     core,
		minLevel: zapcore.ErrorLevel,
	}

	logger := &Logger{
		zap:    zap.New(filtered),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	// Create child logger with With()
	child := logger.With(zap.String("component", "test"))

	// Log at various levels
	child.Info(ctx, "info message")   // Should be filtered
	child.Warn(ctx, "warn message")   // Should be filtered
	child.Error(ctx, "error message") // Should pass through

	// Verify filtering still works
	logs := observed.All()
	assert.Equal(t, 1, len(logs), "only error should pass through")
	assert.Equal(t, "error message", logs[0].Message)
	assert.Equal(t, zapcore.ErrorLevel, logs[0].Level)

	// Verify child logger has the field
	assert.Equal(t, "test", logs[0].ContextMap()["component"])
}

func TestSampling_ActualVolumeReduction(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	cfg := SamplingConfig{
		Enabled: true,
		Tick:    config.Duration(1 * time.Second),
		Levels: map[zapcore.Level]LevelSamplingConfig{
			zapcore.InfoLevel: {Initial: 5, Thereafter: 2},
		},
	}

	sampled := newSampledCore(core, cfg)
	logger := &Logger{
		zap:    zap.New(sampled),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	// Log 100 identical info messages rapidly
	for i := 0; i < 100; i++ {
		logger.Info(ctx, "repeated message")
	}

	// Should be significantly less than 100
	logged := observed.FilterMessage("repeated message").All()
	assert.Less(t, len(logged), 100, "Sampling should reduce log volume significantly")
	assert.Greater(t, len(logged), 5, "Should have sampling happening beyond initial")
}

func TestSampling_ErrorsNeverDropped(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	cfg := SamplingConfig{
		Enabled: true,
		Tick:    config.Duration(10 * time.Millisecond),
		Levels: map[zapcore.Level]LevelSamplingConfig{
			zapcore.InfoLevel: {Initial: 5, Thereafter: 0},
		},
	}

	sampled := newSampledCore(core, cfg)
	logger := &Logger{
		zap:    zap.New(sampled),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	// Log 100 errors
	for i := 0; i < 100; i++ {
		logger.Error(ctx, "error message")
	}

	// All 100 should be logged
	logged := observed.FilterMessage("error message").All()
	assert.Len(t, logged, 100, "Errors should NEVER be sampled")
}
