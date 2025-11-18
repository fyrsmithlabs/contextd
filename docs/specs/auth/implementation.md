# Authentication Implementation

**Parent**: [../SPEC.md](../SPEC.md)

## API Specification

### Public Functions

#### `GenerateToken`

Generates a cryptographically secure random token and saves it to a file.

```go
func GenerateToken(tokenPath string) error
```

**Parameters:**
- `tokenPath` (string): Absolute path where token will be saved

**Returns:**
- `error`: Error if token generation or file write fails, nil on success

**Behavior:**
1. Generates 32 random bytes using `crypto/rand`
2. Encodes bytes as hexadecimal string (64 characters)
3. Writes token to file with 0600 permissions

**Errors:**
- Random generation failure: `"failed to generate random token: %w"`
- File write failure: `"failed to write token file: %w"`

**Example:**
```go
if err := auth.GenerateToken("/home/user/.config/contextd/token"); err != nil {
    log.Fatal(err)
}
```

#### `LoadOrGenerateToken`

Loads existing token or generates a new one if it doesn't exist. Validates format and fixes permissions.

```go
func LoadOrGenerateToken(tokenPath string) error
```

**Parameters:**
- `tokenPath` (string): Absolute path to token file

**Returns:**
- `error`: Error if token is invalid or cannot be generated/read, nil on success

**Behavior:**
1. If token file doesn't exist: generates new token
2. If token file exists:
   - Checks permissions (fixes if not 0600)
   - Reads token content
   - Validates length (64 characters)
   - Validates format (hex regex)

**Errors:**
- File stat failure: `"failed to stat token file: %w"`
- Permission fix failure: `"failed to fix token file permissions: %w"`
- Read failure: `"failed to read token file: %w"`
- Invalid length: `"invalid token length: expected %d, got %d (regenerate token)"`
- Invalid format: `"invalid token format: must be 64 hexadecimal characters (regenerate token)"`

**Example:**
```go
// Idempotent - safe to call on every startup
if err := auth.LoadOrGenerateToken(tokenPath); err != nil {
    log.Fatal("Failed to setup token:", err)
}
```

#### `TokenAuth`

Creates Echo middleware for bearer token authentication.

```go
func TokenAuth(tokenPath string) echo.MiddlewareFunc
```

**Parameters:**
- `tokenPath` (string): Absolute path to token file

**Returns:**
- `echo.MiddlewareFunc`: Echo middleware function

**Behavior:**
1. **Initialization (called once)**:
   - Loads expected token from file
   - Validates token length (64 characters)
   - Panics if token cannot be loaded or is invalid

2. **Request Processing (called per-request)**:
   - Extracts Authorization header
   - Validates Bearer prefix
   - Extracts token
   - Validates length (≤ 64 chars, DoS protection)
   - Validates format (hex regex)
   - Compares token using constant-time algorithm
   - Returns 401 if invalid, continues if valid

**Panics:**
- Token file cannot be read
- Token length is not 64 characters

**Example:**
```go
e := echo.New()

// Add TokenAuth middleware to protected routes
api := e.Group("/api/v1")
api.Use(auth.TokenAuth("/home/user/.config/contextd/token"))

// All routes under /api/v1 now require authentication
api.GET("/checkpoints", handleCheckpoints)
```

### Constants

```go
const (
    // MaxTokenLength is the maximum expected token length (64 hex chars)
    MaxTokenLength = 64

    // TokenBytes is the number of random bytes for token generation
    TokenBytes = 32
)
```

### Variables

```go
var (
    // hexTokenRegex validates that token is valid hex
    hexTokenRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)
)
```

## Authentication Protocol

### HTTP Bearer Token Authentication

**Standard:** [RFC 6750 - The OAuth 2.0 Authorization Framework: Bearer Token Usage](https://www.rfc-editor.org/rfc/rfc6750)

### Request Format

```http
GET /api/v1/checkpoints HTTP/1.1
Host: localhost
Authorization: Bearer a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6
```

### Response Formats

#### Success (200 OK)
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "data": [...]
}
```

#### Unauthorized (401 Unauthorized)
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "message": "Unauthorized"
}
```

### Protocol Validation

The middleware validates:
1. ✅ Authorization header present
2. ✅ Header starts with "Bearer "
3. ✅ Token is not empty
4. ✅ Token length ≤ 64 characters (DoS protection)
5. ✅ Token is valid hexadecimal
6. ✅ Token matches expected value (constant-time)

## Middleware Integration

### Recommended Middleware Order

```go
e := echo.New()

// 1. Logger (log all requests)
e.Use(middleware.Logger())

// 2. Recover (recover from panics)
e.Use(middleware.Recover())

// 3. RequestID (add request ID)
e.Use(middleware.RequestID())

// 4. OTEL Tracing (OpenTelemetry)
e.Use(otelecho.Middleware("contextd"))

// 5. TokenAuth (authenticate) - Apply to protected group
api := e.Group("/api/v1")
api.Use(auth.TokenAuth(tokenPath))

// 6. Route handlers
api.GET("/checkpoints", handleCheckpoints)
```

### Route Protection

**Public Routes (no authentication):**
```go
e.GET("/health", handleHealth)
e.GET("/ready", handleReady)
```

**Protected Routes (authentication required):**
```go
api := e.Group("/api/v1")
api.Use(auth.TokenAuth(tokenPath))

// All routes under /api/v1 require authentication
api.POST("/checkpoints", handleCheckpointCreate)
api.GET("/checkpoints", handleCheckpointList)
api.POST("/remediations", handleRemediationCreate)
// ... etc
```

## Error Handling

### Error Categories

#### 1. Initialization Errors (Panics)

These errors prevent insecure startup:

```go
// Panic: Token file cannot be read
panic("failed to load authentication token from /path: error")

// Panic: Invalid token length
panic("invalid token length: expected 64, got 32")
```

**Recovery:** Not recoverable. Server will not start.

#### 2. Token Generation Errors

```go
// Error: Random generation failed
GenerateToken(): "failed to generate random token: %w"

// Error: File write failed
GenerateToken(): "failed to write token file: %w"
```

**Recovery:** Check filesystem permissions, retry.

#### 3. Token Validation Errors

```go
// Error: Token file doesn't exist (auto-generated)
LoadOrGenerateToken(): calls GenerateToken()

// Error: Invalid token length
LoadOrGenerateToken(): "invalid token length: expected 64, got X (regenerate token)"

// Error: Invalid token format
LoadOrGenerateToken(): "invalid token format: must be 64 hexadecimal characters (regenerate token)"
```

**Recovery:** Delete token file, restart server (will auto-generate).

#### 4. Authentication Errors (HTTP 401)

```go
// All authentication failures return:
return echo.ErrUnauthorized  // HTTP 401
```

**No specific error details** to prevent information leakage.

## Testing Requirements

### Unit Tests

**Coverage Requirements:** ≥ 95% (critical security component)

**Test Categories:**

#### 1. Token Generation Tests
- ✅ Generates valid 64-character hex token
- ✅ Token uniqueness (100 sequential generations)
- ✅ File created with 0600 permissions
- ✅ Error handling for invalid directory
- ✅ Error handling for read-only directory

#### 2. Token Loading Tests
- ✅ Generates token if not exists
- ✅ Loads existing valid token
- ✅ Fixes incorrect permissions (0644 → 0600)
- ✅ Rejects token with invalid length
- ✅ Rejects token with invalid format (non-hex)

#### 3. Middleware Tests
- ✅ Accepts valid bearer token
- ✅ Rejects missing Authorization header
- ✅ Rejects empty bearer token
- ✅ Rejects invalid token
- ✅ Rejects token with wrong length
- ✅ Rejects non-hex token
- ✅ Rejects oversized token (DoS protection)
- ✅ Rejects missing Bearer prefix
- ✅ Panics on missing token file
- ✅ Panics on invalid token length

#### 4. Security Tests
- ✅ Constant-time comparison (timing variance test)
- ✅ Concurrency safety (100 concurrent requests)
- ✅ Token not exposed in error messages

### Integration Tests

#### 1. Full Server Authentication Flow
```go
func TestServerAuthentication(t *testing.T) {
    // 1. Start server with token auth
    // 2. Make authenticated request
    // 3. Verify 200 OK
    // 4. Make unauthenticated request
    // 5. Verify 401 Unauthorized
}
```

#### 2. Token Regeneration Flow
```go
func TestTokenRegeneration(t *testing.T) {
    // 1. Generate initial token
    // 2. Authenticate with token
    // 3. Delete token file
    // 4. Restart server (auto-generates new token)
    // 5. Verify old token no longer works
    // 6. Verify new token works
}
```

### Performance Tests

**Benchmark Requirements:**

```go
func BenchmarkTokenGeneration(b *testing.B) {
    // Target: < 200µs per operation
}

func BenchmarkTokenValidation(b *testing.B) {
    // Target: < 2µs per operation
}

func BenchmarkConstantTimeComparison(b *testing.B) {
    // Verify: Timing variance < 10%
}
```

### Test Execution

```bash
# Run all tests
go test -v ./pkg/auth/

# Run with coverage
go test -v -cover ./pkg/auth/

# Run with race detection
go test -v -race ./pkg/auth/

# Run benchmarks
go test -v -bench=. ./pkg/auth/

# Check coverage threshold
go test -cover ./pkg/auth/ | grep "coverage: [0-9]*\.[0-9]*%" | awk '{if ($2 < 95.0) exit 1}'
```

## Implementation Files

- **Main Implementation**: `/pkg/auth/auth.go`
- **Test Suite**: `/pkg/auth/auth_test.go`
- **Server Integration**: `/cmd/contextd/main.go` (lines 209, 456)
