# Testing Helpers

**Parent**: @./SPEC.md

---

## Overview

Test utilities for capturing, filtering, and asserting on log output. Uses Zap's observer for zero-overhead log capture.

---

## TestLogger

```go
import "go.uber.org/zap/zaptest/observer"

type TestLogger struct {
    *Logger
    observed *observer.ObservedLogs
}

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

func (t *TestLogger) All() []observer.LoggedEntry { return t.observed.All() }
func (t *TestLogger) FilterMessage(msg string) *observer.ObservedLogs {
    return t.observed.FilterMessage(msg)
}
func (t *TestLogger) Reset() { t.observed.TakeAll() }
```

---

## Assertion Methods

### AssertLogged / AssertNotLogged

```go
func (t *TestLogger) AssertLogged(tb testing.TB, level zapcore.Level, msgContains string) {
    tb.Helper()
    for _, entry := range t.observed.All() {
        if entry.Level == level && strings.Contains(entry.Message, msgContains) {
            return
        }
    }
    tb.Errorf("expected log at %v containing %q", level, msgContains)
}

func (t *TestLogger) AssertNotLogged(tb testing.TB, level zapcore.Level, msgContains string) {
    tb.Helper()
    for _, entry := range t.observed.All() {
        if entry.Level == level && strings.Contains(entry.Message, msgContains) {
            tb.Errorf("unexpected log at %v containing %q", level, msgContains)
        }
    }
}
```

### AssertField

```go
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
```

### AssertNoSecrets

```go
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
```

### AssertTraceCorrelation

```go
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
```

---

## Test Examples

### Basic Logging Test

```go
func TestBashService_LogsExecution(t *testing.T) {
    tl := logging.NewTestLogger()
    svc := NewBashService(tl.Logger)

    ctx := WithSession(context.Background(), "sess_123")
    ctx = WithTenant(ctx, &Tenant{OrgID: "acme"})

    _, err := svc.Execute(ctx, "echo hello")
    require.NoError(t, err)

    tl.AssertLogged(t, zapcore.InfoLevel, "tool executed")
    tl.AssertField(t, "tool executed", "tool", "bash")
    tl.AssertNoSecrets(t)
}
```

### Redaction Test

```go
func TestLogger_RedactsSensitiveFields(t *testing.T) {
    tl := logging.NewTestLogger()

    tl.Info(context.Background(), "config loaded",
        logging.RedactedString("password", "super-secret"),
        zap.String("host", "localhost"),
    )

    tl.AssertNoSecrets(t)
}
```

---

## Assertion Summary

| Method | Purpose |
|--------|---------|
| `AssertLogged` | Verify log at level with message |
| `AssertNotLogged` | Verify no log at level with message |
| `AssertField` | Verify field key/value in message |
| `AssertNoSecrets` | Verify no sensitive data leaked |
| `AssertTraceCorrelation` | Verify trace_id present |

---

## Maintenance

**Update when**: New assertion patterns needed, sensitive field list changes

**Keep**: Examples runnable, assertions comprehensive
