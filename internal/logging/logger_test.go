package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Output.OTEL = false // Skip OTEL for basic test

	logger, err := NewLogger(cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, logger)

	assert.NotNil(t, logger.zap)
	assert.Equal(t, cfg, logger.config)
}

func TestLogger_ContextAwareMethods(t *testing.T) {
	core, observed := observer.New(TraceLevel)
	logger := &Logger{
		zap:    zap.New(core),
		config: NewDefaultConfig(),
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		logFunc func()
		level   zapcore.Level
		message string
	}{
		{
			name:    "trace",
			logFunc: func() { logger.Trace(ctx, "trace message", zap.String("key", "val")) },
			level:   TraceLevel,
			message: "trace message",
		},
		{
			name:    "debug",
			logFunc: func() { logger.Debug(ctx, "debug message", zap.String("key", "val")) },
			level:   zapcore.DebugLevel,
			message: "debug message",
		},
		{
			name:    "info",
			logFunc: func() { logger.Info(ctx, "info message", zap.String("key", "val")) },
			level:   zapcore.InfoLevel,
			message: "info message",
		},
		{
			name:    "warn",
			logFunc: func() { logger.Warn(ctx, "warn message", zap.String("key", "val")) },
			level:   zapcore.WarnLevel,
			message: "warn message",
		},
		{
			name:    "error",
			logFunc: func() { logger.Error(ctx, "error message", zap.String("key", "val")) },
			level:   zapcore.ErrorLevel,
			message: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observed.TakeAll() // Clear previous logs
			tt.logFunc()

			logs := observed.All()
			require.Len(t, logs, 1)
			assert.Equal(t, tt.level, logs[0].Level)
			assert.Equal(t, tt.message, logs[0].Message)
			assert.Len(t, logs[0].Context, 1) // "key" field
		})
	}
}

func TestLogger_With(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := &Logger{
		zap:    zap.New(core),
		config: NewDefaultConfig(),
	}

	child := logger.With(zap.String("child_field", "value"))
	child.Info(context.Background(), "child log")

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "child log", logs[0].Message)

	// Check for child_field
	found := false
	for _, field := range logs[0].Context {
		if field.Key == "child_field" && field.String == "value" {
			found = true
			break
		}
	}
	assert.True(t, found, "child_field not found in context")
}

func TestLogger_Named(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := &Logger{
		zap:    zap.New(core),
		config: NewDefaultConfig(),
	}

	named := logger.Named("subsystem")
	named.Info(context.Background(), "named log")

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "subsystem", logs[0].LoggerName)
}

func TestLogger_Enabled(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Level = zapcore.InfoLevel

	core, _ := observer.New(cfg.Level)
	logger := &Logger{
		zap:    zap.New(core),
		config: cfg,
	}

	assert.False(t, logger.Enabled(TraceLevel))
	assert.False(t, logger.Enabled(zapcore.DebugLevel))
	assert.True(t, logger.Enabled(zapcore.InfoLevel))
	assert.True(t, logger.Enabled(zapcore.ErrorLevel))
}

func TestLogger_AutoInjectContextFields(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := &Logger{zap: zap.New(core), config: NewDefaultConfig()}

	tenant := &Tenant{OrgID: "acme", TeamID: "platform", ProjectID: "api"}
	ctx := WithTenant(context.Background(), tenant)
	ctx = WithSessionID(ctx, "sess_123")

	logger.Info(ctx, "test message", zap.String("key", "value"))

	logs := observed.All()
	require.Len(t, logs, 1)

	// Check for tenant fields (uses assertFieldExists from context_test.go)
	assertFieldExists(t, logs[0].Context, "tenant.org", "acme")
	assertFieldExists(t, logs[0].Context, "session.id", "sess_123")
}
