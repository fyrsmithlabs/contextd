# Secret Redaction

**Parent**: @./SPEC.md

---

## Overview

All log output passes through redaction to prevent secret leakage. Two mechanisms:
1. **Domain primitives**: `Secret` type auto-redacts on serialization
2. **Redacting encoder**: Filters field names and value patterns

---

## Domain Primitives (config.Secret)

The `config.Secret` type from `internal/config` provides automatic redaction:

```go
// From internal/config package
type Secret struct {
    value string
}

func (s Secret) Value() string { return s.value }
func (s Secret) String() string { return "[REDACTED]" }
func (s Secret) GoString() string { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }
```

---

## Zap Field Helpers

```go
// Secret creates a Zap field for config.Secret with redaction indicator
func Secret(key string, val config.Secret) zap.Field {
    return zap.Object(key, &secretMarshaler{key: key, val: val})
}

// RedactedString creates a Zap field with redacted value and length
func RedactedString(key, val string) zap.Field {
    return zap.String(key, "[REDACTED:"+strconv.Itoa(len(val))+"]")
}

// Usage
logger.Debug(ctx, "auth header received", RedactedString("authorization", authHeader))
logger.Info(ctx, "secret loaded", Secret("api_key", mySecret))
```

---

## Redacting Encoder

```go
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
        logging.RedactedString("password", "secret123"),
        zap.String("host", "localhost"))
    tl.AssertNoSecrets(t)
}
```

---

## Maintenance

**Update when**: New sensitive field types identified, pattern coverage expands

**Keep**: Field list current, patterns tested
