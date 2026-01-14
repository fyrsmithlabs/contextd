// internal/logging/integration_test.go
package logging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestIntegration_FullLoggingPipeline(t *testing.T) {
	// Create config
	cfg := NewDefaultConfig()
	cfg.Level = TraceLevel
	cfg.Format = "json"
	cfg.Output.Stdout = true
	cfg.Output.OTEL = false
	cfg.Sampling.Enabled = false // Disable for predictable test

	// Create logger (no OTEL provider)
	logger, err := NewLogger(cfg, nil)
	require.NoError(t, err)
	defer func() {
		// Ignore sync errors on stdout/stderr (common on some systems)
		_ = logger.Sync()
	}()

	// Create test context
	tenant := &Tenant{
		OrgID:     "acme",
		TeamID:    "platform",
		ProjectID: "api",
	}
	ctx := WithTenant(context.Background(), tenant)
	ctx = WithSessionID(ctx, "sess_integration_123")
	ctx = WithRequestID(ctx, "req_456")

	// Log at all levels with various fields
	logger.Trace(ctx, "trace message", zap.String("detail", "ultra-verbose"))
	logger.Debug(ctx, "debug message", zap.String("cache", "hit"))
	logger.Info(ctx, "info message", zap.Duration("duration", 45*time.Millisecond))
	logger.Warn(ctx, "warn message", zap.Int("retry_attempt", 2))
	logger.Error(ctx, "error message", zap.Error(fmt.Errorf("test error")))

	// Test secret redaction
	logger.Info(ctx, "config loaded",
		zap.Object("db", &testDBConfig{
			Host:     "localhost",
			Password: config.Secret("super-secret"),
		}),
	)

	// Test child logger
	child := logger.With(zap.String("component", "grpc"))
	child.Info(ctx, "child log")

	// Test named logger
	named := logger.Named("subsystem")
	named.Info(ctx, "named log")

	// Sync may fail on stdout/stderr in some environments (e.g., CI, testing frameworks)
	// This is expected behavior - zap's Sync() attempts to fsync stdout which fails
	// when stdout is not a regular file. We just ensure no panic occurs.
	_ = logger.Sync()
}

// testDBConfig for testing Secret marshaling
type testDBConfig struct {
	Host     string
	Password config.Secret
}

func (c *testDBConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", c.Host)
	// Use secretMarshaler for proper redaction
	if err := (&secretMarshaler{key: "password", val: c.Password}).MarshalLogObject(enc); err != nil {
		return err
	}
	return nil
}

func TestIntegration_ContextFieldInjection(t *testing.T) {
	tl := NewTestLogger()

	tenant := &Tenant{OrgID: "acme", TeamID: "platform", ProjectID: "api"}
	ctx := WithTenant(context.Background(), tenant)
	ctx = WithSessionID(ctx, "sess_123")

	tl.Info(ctx, "request", zap.String("method", "GET"))

	tl.AssertLogged(t, zapcore.InfoLevel, "request")
	tl.AssertField(t, "request", "tenant.org", "acme")
	tl.AssertField(t, "request", "tenant.team", "platform")
	tl.AssertField(t, "request", "session.id", "sess_123")
	tl.AssertField(t, "request", "method", "GET")
}

func TestIntegration_SecretRedaction(t *testing.T) {
	tl := NewTestLogger()

	secret := config.Secret("my-secret-token")
	tl.Info(context.Background(), "auth",
		Secret("credentials", secret),
	)

	tl.AssertLogged(t, zapcore.InfoLevel, "auth")
	tl.AssertNoSecrets(t)
}
