# contextd Architecture Specification

## Core Philosophy

**PRIMARY GOALS:**
1. **Security First**: Authentication, input validation, no vulnerabilities
2. **Context Optimization**: Minimal token usage, efficient operations
3. **Local First**: Instant operations, background sync

## Project Structure

```
contextd/
├── cmd/
│   ├── contextd/       # Main server (API + MCP modes)
│   └── ctxd/          # CLI client
├── pkg/               # Public packages
│   ├── auth/          # Bearer token authentication
│   ├── checkpoint/    # Checkpoint service
│   ├── remediation/   # Error remediation
│   ├── embedding/     # Embedding generation (OpenAI/TEI)
│   ├── security/      # Secret redaction
│   ├── validation/    # Input validation
│   ├── telemetry/     # OpenTelemetry setup
│   └── config/        # Configuration management
├── internal/          # Private packages
│   └── handlers/      # HTTP handlers
└── deployments/       # Systemd, Docker configs
```

## Communication Architecture

### Unix Socket

**Location:** `~/.config/contextd/api.sock`
**Permissions:** 0600 (owner only)
**Protocol:** HTTP over Unix socket

```go
// Server setup
listener, err := net.Listen("unix", socketPath)
os.Chmod(socketPath, 0600)

e := echo.New()
e.Listener = listener
e.Start("")
```

**Why Unix Socket:**
- No network exposure
- Filesystem-based access control
- 35% faster than TCP localhost
- Immune to network-based attacks

### Authentication

**Mechanism:** Bearer token
**Storage:** `~/.config/contextd/token`
**Permissions:** 0600
**Generation:** 32 bytes random → hex (64 chars)

```go
// Validation (constant-time to prevent timing attacks)
if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
    return ErrUnauthorized
}
```

**Never:**
- Log tokens
- Include in error messages
- Transmit over network
- Store in plaintext config files

## Data Flow

### Checkpoint Creation

```
1. User → Claude Code → MCP tool call
2. MCP → contextd (Unix socket + bearer token)
3. contextd → Input validation
4. contextd → Security redaction
5. contextd → OpenAI/TEI (generate embedding)
7. contextd → Response to MCP
8. MCP → Claude Code → User
```

### Semantic Search

```
1. User query → Claude Code → MCP
2. MCP → contextd
3. contextd → Generate query embedding
5. contextd → Filter + rank results
6. contextd → Return top K matches
7. MCP → Claude Code (formatted results)
```

## Configuration Management

### Environment Variables

```bash

# Embeddings (choose one)
OPENAI_API_KEY=sk-xxx          # OpenAI
EMBEDDING_BASE_URL=http://localhost:8080/v1  # TEI
EMBEDDING_MODEL=BAAI/bge-small-en-v1.5       # TEI model

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.example.com
OTEL_SERVICE_NAME=contextd
OTEL_ENVIRONMENT=production

# Server
CONTEXTD_SOCKET=~/.config/contextd/api.sock
CONTEXTD_TOKEN_PATH=~/.config/contextd/token
```

### File-Based Secrets

```
~/.config/contextd/
├── token              # Bearer token (0600)
├── openai_api_key     # OpenAI key (0600)
└── config.yaml        # Non-sensitive config (0644)
```

**Never** store secrets in:
- Git repository
- Environment files committed to repo
- Log files
- Error messages

## Error Handling Conventions

### Error Wrapping

```go
// Always wrap with context
if err != nil {
    return fmt.Errorf("failed to create checkpoint: %w", err)
}

// Use errors.Is for comparison
    // Handle specific error
}

// Use errors.As for error types
var netErr *net.OpError
if errors.As(err, &netErr) {
    // Handle network error
}
```

### Error Messages

```go
// ✅ GOOD: Safe for user
return echo.NewHTTPError(http.StatusBadRequest, "invalid project name")

// ❌ BAD: Exposes internals
return echo.NewHTTPError(http.StatusInternalServerError, err.Error())

// ✅ GOOD: Generic error, logged details
return echo.NewHTTPError(http.StatusInternalServerError, "search failed")
```

### Logging Patterns

```go
// Structured logging
log.WithFields(log.Fields{
    "request_id": reqID,
    "operation":  "checkpoint.create",
    "project":    project,
    "duration_ms": elapsed,
}).Info("checkpoint created")

// Error logging (with context)
log.WithError(err).WithFields(log.Fields{
    "collection": "checkpoints",
}).Error("search failed")
```

## Security Patterns

### Input Validation

**Required for ALL external inputs:**

```go
type CreateCheckpointRequest struct {
    Summary string   `json:"summary" validate:"required,min=1,max=500,no_sql"`
    Content string   `json:"content" validate:"required,min=1,max=50000"`
    Project string   `json:"project" validate:"required,valid_path"`
    Tags    []string `json:"tags" validate:"dive,min=1,max=50"`
}

// Validate
if err := c.Validate(req); err != nil {
    return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}
```

**Custom validators:**
```go
// No SQL injection patterns
func (cv *CustomValidator) NoSQL(fl validator.FieldLevel) bool {
    sqlPatterns := []string{"--", ";", "/*", "*/", "xp_", "sp_", "exec"}
    value := fl.Field().String()
    for _, pattern := range sqlPatterns {
        if strings.Contains(strings.ToLower(value), pattern) {
            return false
        }
    }
    return true
}

// Valid path (no traversal)
func (cv *CustomValidator) ValidPath(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    return filepath.IsLocal(path) && !strings.Contains(path, "..")
}
```

### Secret Redaction

**Always redact before external API calls:**

```go
import "contextd/pkg/security"

// Redact secrets from text
sanitized := security.Redact(text)

// Send to OpenAI/TEI
embedding, err := embeddingClient.Generate(ctx, sanitized)
```

**Patterns detected (20+):**
- API keys (sk-*, openai_api_key, etc.)
- Bearer tokens
- Passwords
- Database URLs
- AWS credentials
- Private keys
- OAuth tokens

### Path Traversal Prevention

```go
// Use filepath.IsLocal (Go 1.20+)
if !filepath.IsLocal(userPath) {
    return fmt.Errorf("path must be local")
}

fullPath := filepath.Join(baseDir, userPath)

// Verify with EvalSymlinks
realPath, err := filepath.EvalSymlinks(fullPath)
if !strings.HasPrefix(realPath, baseDir) {
    return fmt.Errorf("path outside base directory")
}
```

## Performance Patterns

### Embedding Batch Processing

```go
const (
    OpenAIBatchSize = 100  // OpenAI limit
    TEIBatchSize    = 32   // Optimal for local TEI
)

// Batch items
for i := 0; i < len(items); i += batchSize {
    end := min(i+batchSize, len(items))
    batch := items[i:end]

    embeddings, err := generateEmbeddings(ctx, batch)
    if err != nil {
        return err
    }

    // Process batch
}
```


```go
// Insert in batches
const insertBatchSize = 500

// Search with filters (not post-filter)
filter := fmt.Sprintf("project == '%s'", sanitize(project))

// Use HNSW index for >500k vectors
index := entity.NewIndexHNSW(entity.L2, 16, 64)
```

### Connection Reuse

```go
var (
    once         sync.Once
)

    once.Do(func() {
    })
}
```

## OpenTelemetry Instrumentation

### Required Spans

```go
// HTTP handlers (automatic via otelecho)
e.Use(otelecho.Middleware("contextd"))

// Database operations
defer span.End()
span.SetAttributes(
    attribute.String("collection", "checkpoints"),
    attribute.Int("topK", 10),
)

// External API calls
ctx, span := tracer.Start(ctx, "openai.embedding")
defer span.End()
```

### Attribute Naming

```go
// Use namespace
attribute.String("contextd.operation", "checkpoint.create")
attribute.String("contextd.project", project)

// Follow semantic conventions for standard operations
semconv.HTTPMethod("POST")
```

## Testing Standards

### Coverage Requirements

- **Critical packages**: 100% (auth, security, validation)
- **Medium priority**: >80% (everything else)

### Test Structure

```go
// Table-driven tests
func TestValidateInput(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "test", false},
        {"empty", "", true},
        {"too long", strings.Repeat("a", 1001), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInput(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("wantErr %v, got %v", tt.wantErr, err)
            }
        })
    }
}
```

### Integration Tests

```go
//go:build integration


    // Test operations
}
```

## Deployment

### Systemd Service

```ini
[Service]
Type=simple
User=contextd
ExecStart=/usr/local/bin/contextd

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

# Resources
MemoryMax=512M
CPUQuota=50%
```

### File Structure

```
/usr/local/bin/contextd          # Binary
~/.config/contextd/
├── token                        # Auth token
├── openai_api_key              # OpenAI key
└── config.yaml                 # Config
/var/log/contextd/              # Logs (systemd)
```

## Development Workflow

```bash
# 1. Write code
# 2. Format and lint
make lint

# 3. Run tests with race detector
make test

# 4. Check coverage
make coverage

# 5. Security scan
make security-check

# 6. Build
make build

# 7. Install locally
./ctxd install
```

## Best Practices Summary

### ✅ DO

1. Use Unix socket (not TCP)
2. Constant-time auth comparison
3. Validate ALL inputs
4. Redact secrets before external APIs
5. Wrap errors with context
6. Use structured logging
8. Instrument with OpenTelemetry
9. Write table-driven tests
10. Check coverage >80%

### ❌ DON'T

1. Expose Unix socket over network
2. Log tokens or API keys
3. Skip input validation
4. Return detailed errors to clients
5. Commit secrets to git
6. Use TCP for local communication
7. Insert items one at a time
8. Forget to defer span.End()
9. Skip security tests
10. Ignore golangci-lint errors

---

**This spec defines the contextd way.** All code must conform to these patterns.
