package logging

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSecretMarshaler(t *testing.T) {
	secret := config.Secret("super-secret-value")

	core, observed := observer.New(zapcore.InfoLevel)
	logger := &Logger{zap: zap.New(core), config: NewDefaultConfig()}

	logger.Info(context.Background(), "test secret",
		zap.Object("creds", &secretMarshaler{key: "password", val: secret}))

	logs := observed.All()
	require.Len(t, logs, 1)

	// Find password field
	var found bool
	for _, field := range logs[0].Context {
		if field.Key == "creds" {
			// Check redacted
			if enc, ok := field.Interface.(zapcore.ObjectMarshaler); ok {
				enc2 := zapcore.NewMapObjectEncoder()
				err := enc.MarshalLogObject(enc2)
				require.NoError(t, err)
				assert.Equal(t, "[REDACTED:18]", enc2.Fields["password"])
				found = true
			}
		}
	}
	assert.True(t, found, "creds field not found or not redacted")
}

func TestRedactedString(t *testing.T) {
	field := RedactedString("api_key", "sk-1234567890abcdef")

	core, observed := observer.New(zapcore.InfoLevel)
	logger := &Logger{zap: zap.New(core), config: NewDefaultConfig()}

	logger.Info(context.Background(), "test", field)

	logs := observed.All()
	require.Len(t, logs, 1)

	// Check field is redacted with length
	var found bool
	for _, f := range logs[0].Context {
		if f.Key == "api_key" {
			assert.Equal(t, "[REDACTED:19]", f.String)
			found = true
		}
	}
	assert.True(t, found, "api_key field not found")
}

func TestRedactingEncoder_FieldNames(t *testing.T) {
	cfg := NewDefaultConfig()
	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg.Redaction)

	require.NoError(t, err)
	assert.NotNil(t, encoder)
}

func TestRedactingEncoder_Patterns(t *testing.T) {
	cfg := NewDefaultConfig()
	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg.Redaction)

	require.NoError(t, err)
	assert.NotNil(t, encoder)
	assert.Len(t, encoder.redactFields, len(cfg.Redaction.Fields))
	assert.Len(t, encoder.redactRegex, len(cfg.Redaction.Patterns))
}

func TestNewRedactingEncoder_InvalidPattern(t *testing.T) {
	cfg := RedactionConfig{
		Enabled:  true,
		Fields:   []string{"password"},
		Patterns: []string{"(?i)bearer\\s+\\S+", "[invalid("},
	}

	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg)

	assert.Error(t, err)
	assert.Nil(t, encoder)
	assert.Contains(t, err.Error(), "invalid redaction pattern")
	assert.Contains(t, err.Error(), "[invalid(")
}

func TestNewRedactingEncoder_PatternTooLong(t *testing.T) {
	longPattern := string(make([]byte, 201))
	for i := range longPattern {
		longPattern = longPattern[:i] + "a" + longPattern[i+1:]
	}

	cfg := RedactionConfig{
		Enabled:  true,
		Patterns: []string{longPattern},
	}

	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg)

	assert.Error(t, err)
	assert.Nil(t, encoder)
	assert.Contains(t, err.Error(), "pattern too long")
}

func TestNewRedactingEncoder_DisabledSkipsValidation(t *testing.T) {
	// Invalid pattern but redaction disabled should succeed
	cfg := RedactionConfig{
		Enabled:  false,
		Patterns: []string{"[invalid("},
	}

	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg)

	assert.NoError(t, err)
	assert.NotNil(t, encoder)
}

func TestRedactingEncoder_AllMethodsImplemented(t *testing.T) {
	// Verify all encoder methods are implemented
	cfg := RedactionConfig{
		Enabled: true,
		Fields:  []string{"password", "token", "certificate", "credentials", "secret_array"},
	}

	base := newEncoder("json")
	encoder, err := NewRedactingEncoder(base, cfg)
	require.NoError(t, err)
	require.NotNil(t, encoder)

	// Test that all methods can be called without panicking
	assert.NotPanics(t, func() {
		encoder.AddString("password", "secret")
		encoder.AddByteString("token", []byte("token-value"))
		encoder.AddBinary("certificate", []byte{0x00})
		_ = encoder.AddReflected("safe_field", "value")
		_ = encoder.AddObject("credentials", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			return nil
		}))
		_ = encoder.AddArray("secret_array", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			return nil
		}))
	})
}
