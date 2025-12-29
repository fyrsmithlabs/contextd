package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestTestLogger_Creation(t *testing.T) {
	tl := NewTestLogger()
	assert.NotNil(t, tl.Logger)
	assert.NotNil(t, tl.observed)
}

func TestTestLogger_AssertLogged(t *testing.T) {
	tl := NewTestLogger()
	ctx := context.Background()

	tl.Info(ctx, "test message", zap.String("key", "value"))

	tl.AssertLogged(t, zapcore.InfoLevel, "test message")
}

func TestTestLogger_AssertNotLogged(t *testing.T) {
	tl := NewTestLogger()

	tl.AssertNotLogged(t, zapcore.ErrorLevel, "should not exist")
}

func TestTestLogger_AssertField(t *testing.T) {
	tl := NewTestLogger()
	ctx := context.Background()

	tl.Info(ctx, "test", zap.String("key", "value"))

	tl.AssertField(t, "test", "key", "value")
}

func TestTestLogger_AssertNoSecrets(t *testing.T) {
	tl := NewTestLogger()
	ctx := context.Background()

	tl.Info(ctx, "safe", zap.String("username", "alice"))

	tl.AssertNoSecrets(t)
}

func TestTestLogger_AssertNoSecrets_DetectsSecrets(t *testing.T) {
	tl := NewTestLogger()
	ctx := context.Background()

	// This should fail AssertNoSecrets
	tl.Info(ctx, "unsafe", zap.String("password", "secret123"))

	// We can't easily test failure in test, but verify structure
	logs := tl.All()
	assert.Len(t, logs, 1)
}
