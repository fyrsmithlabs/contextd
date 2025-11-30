// internal/logging/redact.go
package logging

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// secretMarshaler wraps config.Secret for Zap object marshaling.
type secretMarshaler struct {
	key string
	val config.Secret
}

// MarshalLogObject implements zapcore.ObjectMarshaler.
func (s *secretMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString(s.key, fmt.Sprintf("[REDACTED:%d]", len(s.val.Value())))
	return nil
}

// Secret creates a Zap field for config.Secret with redaction indicator.
func Secret(key string, val config.Secret) zap.Field {
	return zap.Object(key, &secretMarshaler{key: key, val: val})
}

// RedactedString creates a Zap field with redacted value and length.
func RedactedString(key, val string) zap.Field {
	return zap.String(key, "[REDACTED:"+strconv.Itoa(len(val))+"]")
}

// RedactingEncoder wraps a zapcore.Encoder to redact sensitive fields.
type RedactingEncoder struct {
	zapcore.Encoder
	redactFields map[string]bool
	redactRegex  []*regexp.Regexp
}

// NewRedactingEncoder wraps an encoder with redaction rules.
// Returns error if any redaction pattern fails to compile.
func NewRedactingEncoder(base zapcore.Encoder, cfg RedactionConfig) (*RedactingEncoder, error) {
	if !cfg.Enabled {
		return &RedactingEncoder{Encoder: base}, nil
	}

	fields := make(map[string]bool)
	for _, f := range cfg.Fields {
		fields[strings.ToLower(f)] = true
	}

	// Compile patterns, fail fast on error
	var patterns []*regexp.Regexp
	for _, p := range cfg.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid redaction pattern %q: %w", p, err)
		}
		// Basic ReDoS protection: reject patterns longer than 200 chars
		if len(p) > 200 {
			return nil, fmt.Errorf("redaction pattern too long (max 200 chars): %q", p)
		}
		patterns = append(patterns, re)
	}

	return &RedactingEncoder{
		Encoder:      base,
		redactFields: fields,
		redactRegex:  patterns,
	}, nil
}

// shouldRedactKey returns true if the key should be redacted.
func (e *RedactingEncoder) shouldRedactKey(key string) bool {
	return e.redactFields[strings.ToLower(key)]
}

// AddString redacts sensitive field names and value patterns.
func (e *RedactingEncoder) AddString(key, val string) {
	if e.shouldRedactKey(key) {
		e.Encoder.AddString(key, "[REDACTED]")
		return
	}
	for _, re := range e.redactRegex {
		if re.MatchString(val) {
			e.Encoder.AddString(key, "[REDACTED:pattern]")
			return
		}
	}
	e.Encoder.AddString(key, val)
}

// AddByteString redacts sensitive field names.
func (e *RedactingEncoder) AddByteString(key string, val []byte) {
	if e.shouldRedactKey(key) {
		e.Encoder.AddByteString(key, []byte("[REDACTED]"))
		return
	}
	e.Encoder.AddByteString(key, val)
}

// AddBinary redacts sensitive field names.
func (e *RedactingEncoder) AddBinary(key string, val []byte) {
	if e.shouldRedactKey(key) {
		e.Encoder.AddBinary(key, []byte("[REDACTED]"))
		return
	}
	e.Encoder.AddBinary(key, val)
}

// AddReflected redacts sensitive field names.
// Note: This redacts the entire reflected value if the key is sensitive.
// For deep inspection of reflected structs/maps, use explicit zap.Object() with custom marshalers.
func (e *RedactingEncoder) AddReflected(key string, val interface{}) error {
	if e.shouldRedactKey(key) {
		e.Encoder.AddString(key, "[REDACTED]")
		return nil
	}
	return e.Encoder.AddReflected(key, val)
}

// AddArray redacts sensitive field names.
func (e *RedactingEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	if e.shouldRedactKey(key) {
		e.Encoder.AddString(key, "[REDACTED]")
		return nil
	}
	return e.Encoder.AddArray(key, arr)
}

// AddObject redacts sensitive field names.
func (e *RedactingEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	if e.shouldRedactKey(key) {
		e.Encoder.AddString(key, "[REDACTED]")
		return nil
	}
	return e.Encoder.AddObject(key, obj)
}

// Clone creates a copy of the encoder.
func (e *RedactingEncoder) Clone() zapcore.Encoder {
	return &RedactingEncoder{
		Encoder:      e.Encoder.Clone(),
		redactFields: e.redactFields,
		redactRegex:  e.redactRegex,
	}
}
