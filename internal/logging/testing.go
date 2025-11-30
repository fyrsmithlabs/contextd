// internal/logging/testing.go
package logging

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestLogger wraps Logger with test observation capabilities.
type TestLogger struct {
	*Logger
	observed *observer.ObservedLogs
}

// NewTestLogger creates a logger for testing with full observation.
func NewTestLogger() *TestLogger {
	core, observed := observer.New(TraceLevel)
	return &TestLogger{
		Logger: &Logger{
			zap:    zap.New(core),
			config: NewDefaultConfig(),
		},
		observed: observed,
	}
}

// All returns all logged entries.
func (t *TestLogger) All() []observer.LoggedEntry {
	return t.observed.All()
}

// FilterMessage returns entries matching message substring.
func (t *TestLogger) FilterMessage(msg string) *observer.ObservedLogs {
	return t.observed.FilterMessage(msg)
}

// Reset clears all logged entries.
func (t *TestLogger) Reset() {
	t.observed.TakeAll()
}

// AssertLogged verifies a log at level containing message was logged.
func (t *TestLogger) AssertLogged(tb testing.TB, level zapcore.Level, msgContains string) {
	tb.Helper()
	for _, entry := range t.observed.All() {
		if entry.Level == level && strings.Contains(entry.Message, msgContains) {
			return
		}
	}
	tb.Errorf("expected log at %v containing %q, logs: %+v", level, msgContains, t.observed.All())
}

// AssertNotLogged verifies no log at level containing message was logged.
func (t *TestLogger) AssertNotLogged(tb testing.TB, level zapcore.Level, msgContains string) {
	tb.Helper()
	for _, entry := range t.observed.All() {
		if entry.Level == level && strings.Contains(entry.Message, msgContains) {
			tb.Errorf("unexpected log at %v containing %q", level, msgContains)
		}
	}
}

// AssertField verifies a field with key and value exists in message.
func (t *TestLogger) AssertField(tb testing.TB, msg, key string, expected interface{}) {
	tb.Helper()
	for _, entry := range t.observed.FilterMessage(msg).All() {
		for _, field := range entry.Context {
			if field.Key == key {
				// Compare based on field type
				if field.Type == zapcore.StringType && field.String == expected {
					return
				}
				if reflect.DeepEqual(field.Interface, expected) {
					return
				}
			}
		}
	}
	tb.Errorf("field %q=%v not found in message %q", key, expected, msg)
}

// AssertNoSecrets verifies no sensitive data leaked in logs.
func (t *TestLogger) AssertNoSecrets(tb testing.TB) {
	tb.Helper()
	sensitiveKeys := []string{"password", "secret", "token", "api_key", "authorization", "bearer", "credential", "private_key"}
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)bearer\s+\S+`),
		regexp.MustCompile(`(?i)api[_-]?key[=:]\s*\S+`),
	}

	for _, entry := range t.observed.All() {
		// Check message for patterns
		for _, re := range sensitivePatterns {
			if re.MatchString(entry.Message) {
				tb.Errorf("sensitive pattern in message: %q", entry.Message)
			}
		}

		// Check fields
		for _, field := range entry.Context {
			keyLower := strings.ToLower(field.Key)
			for _, sensitive := range sensitiveKeys {
				if strings.Contains(keyLower, sensitive) {
					// If field is string and NOT redacted, fail
					if field.Type == zapcore.StringType {
						if !strings.Contains(field.String, "[REDACTED]") && field.String != "" {
							tb.Errorf("sensitive field %q not redacted: %q", field.Key, field.String)
						}
					}
				}
			}

			// Check string values for patterns
			if field.Type == zapcore.StringType {
				for _, re := range sensitivePatterns {
					if re.MatchString(field.String) {
						tb.Errorf("sensitive pattern in field %q: %q", field.Key, field.String)
					}
				}
			}
		}
	}
}

// AssertTraceCorrelation verifies trace_id present in message.
func (t *TestLogger) AssertTraceCorrelation(tb testing.TB, msg string) {
	tb.Helper()
	for _, entry := range t.observed.FilterMessage(msg).All() {
		for _, field := range entry.Context {
			if field.Key == "trace_id" {
				return
			}
		}
	}
	tb.Errorf("message %q missing trace_id", msg)
}
