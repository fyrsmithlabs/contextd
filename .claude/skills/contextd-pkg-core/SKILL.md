---
name: contextd-pkg-core
description: Use when working on config, telemetry, or logging packages - enforces environment-based config (never hardcode), sensitive data redaction in logs (never log tokens/secrets), and mandatory OpenTelemetry spans with defer pattern for observability
---

# contextd:pkg-core

## Overview

Core infrastructure packages (config, telemetry, logging) have critical security and reliability requirements. This skill enforces patterns that prevent credential leaks, ensure observability, and maintain configurability.

**Core Principle**: Infrastructure code must be production-ready from day one. No "fix later", no shortcuts.

## When to Use This Skill

Use when working with:
- `pkg/config` - Configuration management
- `pkg/telemetry` - OpenTelemetry integration
- `pkg/logging` - Structured logging
- Any package that handles environment variables, secrets, or observability

**Triggers**:
- Adding configuration options
- Adding logging statements
- Adding OpenTelemetry spans
- Debugging with logging
- "Just for demo" or "will fix later" thoughts

## Core Packages

### pkg/config - Configuration Management

**Requirements**:
- Environment variables ONLY (no hardcoded values)
- Use `getEnv()` helper for all config
- Validate on load (fail fast)
- Absolute paths or home directory relative
- No secrets in code (load from environment variables or files with 0600 permissions)

**Pattern**:
```go
// GOOD - Environment variable with default
type Config struct {
    CacheTTL      time.Duration
    HTTPPort      int
    QdrantURI     string
}

func Load() *Config {
    return &Config{
        CacheTTL:   parseDuration(getEnv("CACHE_TTL", "300s")),
        HTTPPort:   parseInt(getEnv("HTTP_PORT", "8080")),
        QdrantURI:  getEnv("QDRANT_URI", "localhost:6334"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// BAD - Hardcoded value
const CacheTTL = 300  // Never do this

// BAD - Magic number
cfg.CacheTTL = 300  // Where did 300 come from?

// BAD - TODO comment
const CacheTTL = 300  // TODO: Make configurable
```

**Validation**:
```go
func (c *Config) Validate() error {
    if c.CacheTTL < time.Second {
        return errors.New("cache TTL must be >= 1s")
    }
    if c.HTTPPort < 1 || c.HTTPPort > 65535 {
        return errors.New("HTTP port must be between 1-65535")
    }
    return nil
}
```

### pkg/telemetry - OpenTelemetry Integration

**Requirements**:
- ALWAYS create spans for operations
- Use `defer span.End()` immediately after creation
- Add attributes for context (IDs, paths, counts)
- Record errors with `span.RecordError(err)`
- Context propagation mandatory

**Pattern**:
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("contextd")

// GOOD - Complete span pattern
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "checkpoint.Save")
    defer span.End()  // MUST be immediately after Start

    span.SetAttributes(
        attribute.String("checkpoint.id", checkpoint.ID),
        attribute.String("project.path", checkpoint.Project),
    )

    if err := s.store.Upsert(ctx, checkpoint); err != nil {
        span.RecordError(err)  // MUST record errors
        return fmt.Errorf("upsert failed: %w", err)
    }

    return nil
}

// BAD - Missing span entirely
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    return s.store.Upsert(ctx, checkpoint)  // No observability
}

// BAD - Missing defer
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "checkpoint.Save")
    // Missing: defer span.End()
    return s.store.Upsert(ctx, checkpoint)  // Span never closes
}

// BAD - No attributes
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "checkpoint.Save")
    defer span.End()
    // Missing: span.SetAttributes() - no context about what's being saved
    return s.store.Upsert(ctx, checkpoint)
}

// BAD - Missing error recording
func (s *Service) Save(ctx context.Context, checkpoint *Checkpoint) error {
    ctx, span := tracer.Start(ctx, "checkpoint.Save")
    defer span.End()

    if err := s.store.Upsert(ctx, checkpoint); err != nil {
        // Missing: span.RecordError(err)
        return err
    }
    return nil
}
```

**Span Naming**: Use `Package.Function` format
- `checkpoint.Save`
- `remediation.Search`
- `auth.ValidateToken`

### pkg/logging - Structured Logging

**Requirements**:
- Structured logging ONLY (zap, zerolog)
- NEVER log sensitive data (tokens, passwords, API keys)
- Redact secrets explicitly
- Appropriate log levels (Debug/Info/Warn/Error)
- Context fields for correlation

**Pattern**:
```go
import "go.uber.org/zap"

// GOOD - Structured logging with redaction
logger.Info("Authentication attempt",
    zap.String("username", username),
    zap.String("token", "[REDACTED]"),  // NEVER log actual token
    zap.String("request_id", requestID),
)

logger.Error("Token validation failed",
    zap.String("username", username),
    zap.String("reason", "invalid signature"),
    zap.Error(err),
)

// BAD - Logs actual token
logger.Info("Validating token", zap.String("token", actualToken))  // LEAKED!

// BAD - Printf debugging
fmt.Printf("DEBUG: token=%s, expected=%s\n", token, expected)  // LEAKED!

// BAD - Logs sensitive comparison
logger.Debug("Token comparison",
    zap.String("provided", token),
    zap.String("expected", m.expectedToken),  // LEAKED!
)
```

**Redaction Pattern**:
```go
// GOOD - Explicit redaction helper
func redactToken(token string) string {
    if len(token) < 8 {
        return "[REDACTED]"
    }
    return token[:4] + "..." + "[REDACTED]"  // Show prefix only
}

logger.Info("Token received",
    zap.String("token_prefix", redactToken(token)),
)

// GOOD - Conditional logging (never in production)
if os.Getenv("DEBUG_TOKENS") == "true" {
    logger.Debug("UNSAFE: Full token", zap.String("token", token))
}
```

## Quick Reference

| Package | Must Do | Never Do |
|---------|---------|----------|
| config | Environment variables, validation | Hardcode values, TODOs, secrets in code |
| telemetry | `ctx, span := tracer.Start()` + `defer span.End()` | Skip spans, forget defer, no attributes |
| logging | Structured logs, redact secrets | Printf debugging, log tokens/passwords |

## Testing Requirements

**Configuration Tests**:
```go
func TestConfig_Load_EnvironmentOverride(t *testing.T) {
    os.Setenv("CACHE_TTL", "600s")
    defer os.Unsetenv("CACHE_TTL")

    cfg := Load()
    assert.Equal(t, 600*time.Second, cfg.CacheTTL)
}

func TestConfig_Validate_InvalidTTL(t *testing.T) {
    cfg := &Config{CacheTTL: 100 * time.Millisecond}
    err := cfg.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "must be >= 1s")
}
```

**Telemetry Tests**:
```go
func TestTelemetry_SpanCreation(t *testing.T) {
    ctx, span := tracer.Start(context.Background(), "test.Operation")
    defer span.End()

    assert.NotNil(t, span)
    // Verify span attributes, timing, etc.
}
```

**Logging Tests**:
```go
func TestLogging_SensitiveDataRedacted(t *testing.T) {
    // Use zap test observer to verify no tokens in logs
    core, observed := observer.New(zap.InfoLevel)
    logger := zap.New(core)

    token := "secret-token-123"
    logger.Info("Auth", zap.String("token", "[REDACTED]"))

    entries := observed.All()
    assert.Len(t, entries, 1)
    assert.NotContains(t, entries[0].Message, "secret-token")
}
```

## Common Mistakes

### Configuration Mistakes

**Mistake**: "Hardcode for demo, fix later"
```go
const CacheTTL = 300  // TODO: Make configurable
```

**Fix**: Use environment variable from day one
```go
CacheTTL: parseDuration(getEnv("CACHE_TTL", "300s"))
```

**Mistake**: "Partial compliance" (env var but no validation)
```go
cfg.CacheTTL = getEnv("CACHE_TTL", "300s")  // No validation
```

**Fix**: Validate on load
```go
cfg := Load()
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}
```

### Telemetry Mistakes

**Mistake**: "Spans are optional"
```go
// No spans at all
func (s *Service) Save(ctx context.Context, data Data) error {
    return s.store.Save(ctx, data)
}
```

**Fix**: Spans are mandatory
```go
func (s *Service) Save(ctx context.Context, data Data) error {
    ctx, span := tracer.Start(ctx, "service.Save")
    defer span.End()

    if err := s.store.Save(ctx, data); err != nil {
        span.RecordError(err)
        return err
    }
    return nil
}
```

**Mistake**: Forgot defer
```go
ctx, span := tracer.Start(ctx, "operation")
// Missing: defer span.End()
```

**Fix**: Always defer immediately
```go
ctx, span := tracer.Start(ctx, "operation")
defer span.End()  // IMMEDIATELY after Start
```

### Logging Mistakes

**Mistake**: "Need to see token for debugging"
```go
logger.Info("Token validation", zap.String("token", actualToken))
```

**Fix**: NEVER log secrets
```go
logger.Info("Token validation", zap.String("token", "[REDACTED]"))
```

**Mistake**: Printf debugging
```go
fmt.Printf("DEBUG: value=%v\n", sensitiveData)
```

**Fix**: Structured logging with redaction
```go
logger.Debug("Processing data", zap.String("id", data.ID))  // No sensitive fields
```

## Red Flags - STOP and Refactor

If you write any of these, STOP and refactor immediately:

**Configuration**:
- [ ] Hardcoded value (no environment variable)
- [ ] Magic number constant
- [ ] TODO comment about making configurable
- [ ] Secrets in code

**Telemetry**:
- [ ] Function without span
- [ ] Span without `defer span.End()`
- [ ] Span without attributes
- [ ] Error without `span.RecordError(err)`

**Logging**:
- [ ] `fmt.Printf()` for debugging
- [ ] Logging token, password, or API key values
- [ ] Logging without redaction of sensitive fields
- [ ] Unstructured log messages

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "Hardcoding for demo, will fix later" | Environment variables take 30 seconds. Do it now. |
| "Need to see token value to debug" | Token leakage is security incident. Redact always. |
| "Spans are just observability" | Observability is production requirement, not optional. |
| "Missing defer is minor bug" | Span leak causes memory leak. Always defer. |
| "I'll add validation later" | Config without validation fails silently. Validate now. |
| "This is just temporary logging" | Temporary logs get committed. Redact from start. |
| "Printf is faster for debugging" | Printf logs go nowhere in production. Use structured. |
| "Tests pass without spans" | Non-functional requirements still required. |

## Integration with Verification

When completing work on core packages:

**Use**: `contextd:completing-major-task` skill

**Verification must include**:
- [ ] Config uses environment variables (no hardcoded values)
- [ ] All secrets loaded from files (not in code)
- [ ] All operations have OpenTelemetry spans
- [ ] All spans have `defer span.End()`
- [ ] All logs redact sensitive data
- [ ] gosec passes (no security findings)
- [ ] Tests cover config validation, span creation, log redaction

**Security validation**:
- [ ] No credentials in code: `grep -r "api.*key\|token.*=\|password.*=" pkg/`
- [ ] No printf debugging: `grep -r "fmt.Printf\|fmt.Println" pkg/`
- [ ] All spans have defer: `grep -A1 "tracer.Start" pkg/ | grep -v "defer span.End"`

## Real-World Impact

**Configuration violations**:
- Hardcoded API key → $50K AWS bill from leaked credentials
- Missing validation → Silent failure in production for 3 hours

**Logging violations**:
- Logged OAuth token → Compromised user accounts
- Printf debugging → Lost all diagnostic data in production

**Telemetry violations**:
- Missing spans → 2 hours debugging with no trace data
- Forgot defer → Memory leak crashed production after 6 hours
