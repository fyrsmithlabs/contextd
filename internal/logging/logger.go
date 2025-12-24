// internal/logging/logger.go
package logging

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps Zap with context-aware methods.
type Logger struct {
	zap    *zap.Logger
	config *Config
}

// NewLogger creates a logger from config.
// otelProvider can be nil to disable OTEL output.
func NewLogger(cfg *Config, otelProvider log.LoggerProvider) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	core, err := newDualCore(cfg, otelProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create core: %w", err)
	}

	// Build zap logger with core
	opts := []zap.Option{}
	if cfg.Caller.Enabled {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(cfg.Caller.Skip))
	}
	if cfg.Stacktrace.Level != 0 {
		opts = append(opts, zap.AddStacktrace(cfg.Stacktrace.Level))
	}

	zapLogger := zap.New(core, opts...)

	// Add constant fields from config
	if len(cfg.Fields) > 0 {
		fields := make([]zap.Field, 0, len(cfg.Fields))
		for k, v := range cfg.Fields {
			fields = append(fields, zap.String(k, v))
		}
		zapLogger = zapLogger.With(fields...)
	}

	return &Logger{
		zap:    zapLogger,
		config: cfg,
	}, nil
}

// newEncoder creates JSON or console encoder.
func newEncoder(format string) zapcore.Encoder {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	if format == "console" {
		return zapcore.NewConsoleEncoder(encoderCfg)
	}
	return zapcore.NewJSONEncoder(encoderCfg)
}

// Context-aware logging methods

func (l *Logger) Trace(ctx context.Context, msg string, fields ...zap.Field) {
	if l.Enabled(TraceLevel) {
		allFields := append(ContextFields(ctx), fields...)
		l.zap.Log(TraceLevel, msg, allFields...)
	}
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.Debug(msg, allFields...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.Info(msg, allFields...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.Warn(msg, allFields...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.Error(msg, allFields...)
}

func (l *Logger) DPanic(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.DPanic(msg, allFields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(ContextFields(ctx), fields...)
	l.zap.Fatal(msg, allFields...)
}

// Child logger creation

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zap:    l.zap.With(fields...),
		config: l.config,
	}
}

func (l *Logger) Named(name string) *Logger {
	return &Logger{
		zap:    l.zap.Named(name),
		config: l.config,
	}
}

// Enabled returns true if the given level is enabled.
func (l *Logger) Enabled(level zapcore.Level) bool {
	return l.zap.Core().Enabled(level)
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	err := l.zap.Sync()
	// Ignore sync errors on stdout/stderr (common on Linux)
	if err != nil && isStdoutSyncError(err) {
		return nil
	}
	return err
}

// Underlying returns the underlying zap.Logger.
// Useful when integrating with libraries that require a *zap.Logger.
func (l *Logger) Underlying() *zap.Logger {
	return l.zap
}

// isStdoutSyncError checks if error is harmless stdout/stderr sync error.
// On Linux, syncing stdout/stderr returns EINVAL or ENOTTY which are safe to ignore.
func isStdoutSyncError(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.EINVAL || errno == syscall.ENOTTY
	}
	return false
}
