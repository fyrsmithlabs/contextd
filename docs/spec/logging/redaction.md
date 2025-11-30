# Secret Redaction

**Parent**: @./SPEC.md

---

## Overview

All log output passes through redaction to prevent secret leakage. Two mechanisms:
1. **Domain primitives**: `Secret` type auto-redacts on serialization
2. **Redacting encoder**: Filters field names and value patterns

---

## Domain Primitives

```go
type Secret string

func (s Secret) String() string      { return "[REDACTED]" }
func (s Secret) GoString() string    { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }
func (s Secret) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddString("value", "[REDACTED]")
    enc.AddInt("length", len(s))
    return nil
}
```

---

## Secret Type Usage

```go
type DatabaseConfig struct {
    Host     string `json:"host"`
    Password Secret `json:"password"`  // Auto-redacted
}

logger.Info(ctx, "database configured", zap.Object("config", &dbConfig))
```

Output: `{"config": {"host": "localhost", "password": "[REDACTED]"}}`

---

## Zap Field Helper

```go
func RedactedString(key string, val string) zap.Field {
    return zap.String(key, "[REDACTED:"+strconv.Itoa(len(val))+"]")
}

// Usage
logger.Debug(ctx, "auth header received", RedactedString("authorization", authHeader))
```

---

## Redacting Encoder

```go
type RedactingEncoder struct {
    zapcore.Encoder
    redactFields map[string]bool
    redactRegex  []*regexp.Regexp
}

func NewRedactingEncoder(base zapcore.Encoder, cfg RedactionConfig) *RedactingEncoder {
    fields := make(map[string]bool)
    for _, f := range cfg.Fields {
        fields[strings.ToLower(f)] = true
    }
    var patterns []*regexp.Regexp
    for _, p := range cfg.Patterns {
        if re, err := regexp.Compile(p); err == nil {
            patterns = append(patterns, re)
        }
    }
    return &RedactingEncoder{Encoder: base, redactFields: fields, redactRegex: patterns}
}

func (e *RedactingEncoder) AddString(key, val string) {
    if e.redactFields[strings.ToLower(key)] {
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
```

---

## Configuration

```yaml
redaction:
  enabled: true
  fields: ["password", "secret", "token", "api_key", "authorization", "bearer", "credential", "private_key"]
  patterns:
    - "(?i)bearer\\s+[a-zA-Z0-9_-]+"
    - "(?i)api[_-]?key[=:]\\s*\\S+"
```

---

## Default Redacted Fields

| Field Name | Reason |
|------------|--------|
| password | Authentication credentials |
| secret | Generic secrets |
| token | Auth tokens, API tokens |
| api_key | Service API keys |
| authorization | HTTP auth headers |
| bearer | Bearer tokens |
| credential | Generic credentials |
| private_key | Cryptographic keys |

---

## Default Redacted Patterns

| Pattern | Matches |
|---------|---------|
| `(?i)bearer\s+\S+` | Bearer tokens in values |
| `(?i)api[_-]?key[=:]\s*\S+` | API keys in values |

---

## Testing Redaction

```go
func TestLogger_RedactsSensitiveFields(t *testing.T) {
    tl := logging.NewTestLogger()
    tl.Info(context.Background(), "config loaded",
        zap.Object("database", &DatabaseConfig{Host: "localhost", Password: logging.Secret("secret")}))
    tl.AssertNoSecrets(t)
}
```

---

## Maintenance

**Update when**: New sensitive field types identified, pattern coverage expands

**Keep**: Field list current, patterns tested
