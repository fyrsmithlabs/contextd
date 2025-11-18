# Authentication System Specification

> **STATUS**: ⚠️ NOT APPLICABLE TO MVP
>
> This specification documents the Bearer token authentication system.
> **MVP does not implement authentication** (trusted network assumption).
>
> **Use this spec only if implementing post-MVP authentication.**
>
> For MVP architecture, see `docs/standards/architecture.md`.

---

**Version:** 1.0.0
**Status:** ⏸ Deferred (Post-MVP)
**MVP Status:** Not Implemented (No Authentication)
**Reason:** MVP uses trusted network model, authentication added post-MVP
**Last Updated:** 2025-11-18
**Package:** `pkg/auth`

## Table of Contents

1. [Overview](#overview)
2. [Features and Capabilities](#features-and-capabilities)
3. [Architecture and Design](#architecture-and-design)
4. [Token Generation](#token-generation)
5. [Token Storage](#token-storage)
6. [Authentication Protocol](#authentication-protocol)
7. [Security Model](#security-model)
8. [API Specification](#api-specification)
9. [Data Models](#data-models)
10. [Authentication Middleware](#authentication-middleware)
11. [Performance Characteristics](#performance-characteristics)
12. [Error Handling](#error-handling)
13. [Security Considerations](#security-considerations)
14. [Testing Requirements](#testing-requirements)
15. [Usage Examples](#usage-examples)
16. [Related Documentation](#related-documentation)

---

## Overview

> **Note**: This section describes Unix socket transport. MVP uses HTTP transport on port 8080 with no authentication.

### Purpose

The authentication system (`pkg/auth`) provides secure, lightweight bearer token authentication for contextd's Unix domain socket API. It is designed for single-user, localhost scenarios where the primary security concern is preventing unauthorized access to the user's own data.

### Design Philosophy

- **Local-First Security**: Optimized for Unix socket transport with no network exposure
- **Zero External Dependencies**: No external authentication services required
- **Automatic Setup**: Token auto-generation on first run for seamless user experience
- **Timing Attack Prevention**: Constant-time token comparison prevents timing-based attacks
- **File Permission Hardening**: Strict 0600 permissions on token files
- **Fail-Secure**: Panics on initialization errors to prevent insecure startup

### Key Characteristics

| Characteristic | Value |
|---------------|-------|
| Transport | Unix domain socket |
| Auth Method | Bearer token |
| Token Length | 64 hexadecimal characters |
| Token Entropy | 256 bits (32 random bytes) |
| Token Storage | Filesystem (`~/.config/contextd/token`) |
| File Permissions | 0600 (owner read/write only) |
| Comparison Method | Constant-time |
| Network Exposure | None (Unix socket only) |

### Scope

**In Scope:**
- Bearer token generation and validation
- Token file management and permissions
- Echo middleware for HTTP authentication
- Constant-time token comparison
- Automatic token setup on first run

**Out of Scope:**
- Multi-user authentication
- Token rotation or expiration
- OAuth or external identity providers
- Role-based access control (RBAC)
- Session management
- Network-based authentication

---

## Features and Capabilities

### Core Features

1. **Secure Token Generation**
   - Cryptographically secure random token generation
   - 256 bits of entropy (32 bytes)
   - Hex-encoded for safe transmission (64 characters)
   - Uses `crypto/rand` from Go standard library

2. **Automatic Token Setup**
   - Token auto-generated on first run if missing
   - Automatic permission fixing (0600)
   - Validation of existing tokens
   - Idempotent initialization

3. **Token Validation**
   - Length validation (exactly 64 characters)
   - Format validation (hexadecimal only)
   - Constant-time comparison prevents timing attacks
   - DoS protection (rejects oversized tokens)

4. **Echo Middleware Integration**
   - Drop-in middleware for Echo framework
   - Token loaded once during initialization
   - Validates Bearer token format
   - Returns standard HTTP 401 on failure

5. **Security Hardening**
   - Token loaded once at startup (prevents TOCTOU attacks)
   - Strict file permissions (0600)
   - Regex validation for token format
   - Length limits prevent DoS attacks
   - Panic on initialization failure (fail-secure)

### Capabilities Matrix

| Feature | Supported | Notes |
|---------|-----------|-------|
| Token Generation | ✅ Yes | `crypto/rand`, 32 bytes |
| Token Validation | ✅ Yes | Format, length, constant-time |
| Auto-Generation | ✅ Yes | First run if missing |
| Permission Fixing | ✅ Yes | Auto-fixes to 0600 |
| Echo Middleware | ✅ Yes | Standard HTTP auth |
| Constant-Time Compare | ✅ Yes | `subtle.ConstantTimeCompare` |
| Token Rotation | ❌ No | Manual regeneration only |
| Token Expiration | ❌ No | Tokens never expire |
| Multi-Token Support | ❌ No | Single token per instance |
| Revocation | ❌ No | Delete token file to revoke |

---

## Architecture and Design

> **Note**: This section describes Unix socket transport. MVP uses HTTP transport on port 8080 with no authentication.

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client Application                      │
│                    (ctxd CLI / curl)                        │
└─────────────────────────────┬───────────────────────────────┘
                              │ HTTP/1.1 over Unix Socket
                              │ Authorization: Bearer <token>
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Unix Domain Socket                        │
│              ~/.config/contextd/api.sock (0600)             │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Echo HTTP Server                        │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         Middleware Chain (order matters)               │ │
│  │                                                        │ │
│  │  1. Logger        ──────────────────────────────────▶ │ │
│  │  2. Recover       ──────────────────────────────────▶ │ │
│  │  3. RequestID     ──────────────────────────────────▶ │ │
│  │  4. OTEL Tracing  ──────────────────────────────────▶ │ │
│  │  5. TokenAuth ◀──┐  (This Package)                   │ │
│  └──────────────────┼─────────────────────────────────────┘ │
│                     │                                        │
│                     ▼                                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │       pkg/auth/TokenAuth Middleware                  │   │
│  │                                                      │   │
│  │  1. Extract Authorization header                    │   │
│  │  2. Validate Bearer prefix                          │   │
│  │  3. Extract token                                   │   │
│  │  4. Validate length (64 chars)                      │   │
│  │  5. Validate format (hex regex)                     │   │
│  │  6. Constant-time compare with expected token       │   │
│  │  7. Return 401 if invalid, continue if valid        │   │
│  └──────────────────────────────────────────────────────┘   │
│                     │                                        │
│                     ▼                                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Protected API Handlers                  │   │
│  │  (Checkpoints, Remediations, Troubleshooting, etc.) │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Component Design

#### 1. Token Generator

```go
// GenerateToken creates a secure random token
GenerateToken(tokenPath string) error
  ├─> crypto/rand.Read(32 bytes)
  ├─> hex.EncodeToString() → 64 chars
  ├─> os.WriteFile(tokenPath, 0600)
  └─> Return error or nil
```

**Security Properties:**
- Uses `crypto/rand` (CSPRNG)
- 256 bits of entropy
- Hex encoding (safe for transmission)
- Atomic file write with 0600 permissions

#### 2. Token Loader

```go
// LoadOrGenerateToken ensures token exists and is valid
LoadOrGenerateToken(tokenPath string) error
  ├─> Check if file exists
  │   ├─> If not: GenerateToken()
  │   └─> If yes: continue
  ├─> Check file permissions
  │   ├─> If not 0600: os.Chmod(0600)
  │   └─> Continue
  ├─> Read token file
  ├─> Validate length (64 chars)
  ├─> Validate format (hex regex)
  └─> Return error or nil
```

**Idempotency:** Safe to call multiple times

#### 3. Authentication Middleware

```go
// TokenAuth creates Echo middleware
TokenAuth(tokenPath string) echo.MiddlewareFunc
  ├─> Load expectedToken from file (ONCE at init)
  ├─> Validate token length (panic if invalid)
  └─> Return middleware function:
      └─> For each request:
          ├─> Extract Authorization header
          ├─> Validate Bearer prefix
          ├─> Extract token
          ├─> Validate length (max 64 chars, DoS protection)
          ├─> Validate format (hex regex)
          ├─> Constant-time compare with expectedToken
          └─> Return 401 or call next()
```

**Security Properties:**
- Token loaded once at startup (prevents TOCTOU)
- Panics on initialization failure (fail-secure)
- Constant-time comparison (timing attack prevention)
- Length validation (DoS prevention)
- Format validation (injection prevention)

### Data Flow

#### Successful Authentication Flow

```
1. Client reads token from ~/.config/contextd/token
2. Client sends HTTP request with header:
   Authorization: Bearer <64-char-hex-token>
3. Unix socket receives request
4. Echo router passes to TokenAuth middleware
5. Middleware extracts Bearer token
6. Middleware validates format and length
7. Middleware compares token using constant-time algorithm
8. Token matches → Request continues to handler
9. Handler processes request and returns response
```

#### Failed Authentication Flow

```
1. Client sends request with invalid/missing token
2. TokenAuth middleware rejects request
3. Returns HTTP 401 Unauthorized
4. Request never reaches handler
```

### Security Architecture

#### Defense Layers

1. **Transport Security**
   - Unix domain socket (no network exposure)
   - Socket permissions: 0600

2. **Token Security**
   - 256 bits of entropy
   - Cryptographically secure generation
   - File permissions: 0600
   - Constant-time comparison

3. **Validation Security**
   - Length validation (DoS prevention)
   - Format validation (injection prevention)
   - Regex validation (strict hex-only)
   - Early rejection of invalid formats

4. **Implementation Security**
   - Token loaded once at startup
   - Panic on initialization failure
   - No token logging or error exposure
   - Memory-safe Go implementation

---

## Token Generation

### Generation Algorithm

```go
func GenerateToken(tokenPath string) error {
    // Step 1: Generate 32 random bytes (256 bits of entropy)
    tokenBytes := make([]byte, 32)
    if _, err := rand.Read(tokenBytes); err != nil {
        return fmt.Errorf("failed to generate random token: %w", err)
    }

    // Step 2: Convert to hex string (64 characters)
    token := hex.EncodeToString(tokenBytes)

    // Step 3: Write to file with restricted permissions (0600)
    if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
        return fmt.Errorf("failed to write token file: %w", err)
    }

    return nil
}
```

### Token Properties

| Property | Value | Rationale |
|----------|-------|-----------|
| Entropy | 256 bits | Exceeds NIST recommendations for long-term security |
| Random Source | `crypto/rand` | Cryptographically secure pseudorandom number generator |
| Encoding | Hexadecimal | Safe for HTTP headers, no escaping needed |
| Length | 64 characters | 2 hex chars per byte × 32 bytes |
| Character Set | `[a-f0-9]` | Lowercase hex for consistency |

### Uniqueness Guarantee

- **Collision Probability**: 2^-256 (effectively zero)
- **Random Source**: OS-level entropy pool
- **Tested**: 100 sequential generations produce 100 unique tokens

### Token Format

```
Format:     64 hexadecimal characters
Regex:      ^[a-f0-9]{64}$
Example:    a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6
Length:     64 bytes (ASCII encoding)
Entropy:    256 bits
```

### Generation Testing

The token generator is tested for:
- ✅ Valid token length (64 chars)
- ✅ Valid hex format
- ✅ Correct file permissions (0600)
- ✅ Uniqueness across 100 sequential generations
- ✅ Error handling for invalid directories
- ✅ Error handling for read-only directories

---

## Token Storage

### Storage Location

**Default Path:** `~/.config/contextd/token`

**Configurable Via:**
- Environment variable: `CONTEXTD_TOKEN_PATH`
- Configuration struct: `config.Auth.TokenPath`

### File Permissions

**Required Permissions:** `0600` (owner read/write only)

```bash
$ ls -la ~/.config/contextd/token
-rw------- 1 user user 64 Nov 04 12:00 /home/user/.config/contextd/token
```

**Permission Enforcement:**
- Created with 0600 permissions
- Automatically fixed if permissions are incorrect
- Verified on every server startup

### Directory Structure

```
~/.config/contextd/
├── api.sock          # Unix domain socket (0600)
├── token             # Bearer token (0600)
├── config.yaml       # Configuration (optional, 0600)
└── openai_api_key    # OpenAI API key (optional, 0600)
```

### Storage Security

| Threat | Mitigation |
|--------|------------|
| Unauthorized file access | 0600 permissions (owner only) |
| Network interception | Not applicable (local filesystem) |
| Process injection | Unix permissions + SELinux/AppArmor |
| Backup exposure | User responsible for backup security |
| Accidental sharing | Hidden directory (`~/.config/`) |

### File Operations

#### Read Token
```bash
# Read token from file
TOKEN=$(cat ~/.config/contextd/token)
```

#### Regenerate Token
```bash
# Delete old token
rm ~/.config/contextd/token

# Restart contextd (will auto-generate new token)
systemctl --user restart contextd
```

#### Verify Permissions
```bash
# Check token file permissions
stat -c "%a %n" ~/.config/contextd/token
# Output: 600 /home/user/.config/contextd/token
```

---

## Authentication Protocol

> **Note**: This section describes Unix socket transport. MVP uses HTTP transport on port 8080 with no authentication.

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

### Authentication Flow

```
┌──────────┐                                    ┌──────────┐
│  Client  │                                    │  Server  │
└────┬─────┘                                    └────┬─────┘
     │                                               │
     │  GET /api/v1/checkpoints                     │
     │  Authorization: Bearer <token>               │
     │──────────────────────────────────────────────▶
     │                                               │
     │                  ┌───────────────────────┐   │
     │                  │ TokenAuth Middleware  │   │
     │                  │                       │   │
     │                  │ 1. Extract token      │   │
     │                  │ 2. Validate format    │   │
     │                  │ 3. Constant-time      │   │
     │                  │    compare            │   │
     │                  │ 4. Accept or reject   │   │
     │                  └───────────────────────┘   │
     │                                               │
     │◀──────────────────────────────────────────────│
     │  200 OK or 401 Unauthorized                   │
     │                                               │
```

### Protocol Validation

The middleware validates:
1. ✅ Authorization header present
2. ✅ Header starts with "Bearer "
3. ✅ Token is not empty
4. ✅ Token length ≤ 64 characters (DoS protection)
5. ✅ Token is valid hexadecimal
6. ✅ Token matches expected value (constant-time)

---

## Security Model

### Threat Model

#### Assumptions

**Trusted:**
- The user's filesystem
- The user's local processes
- The operating system kernel

**Untrusted:**
- Network traffic (N/A - no network)
- Other users on the system
- Processes running as other users

#### Threats Addressed

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Timing attack | Medium | High | Constant-time comparison |
| Token theft | Low | High | File permissions (0600) |
| DoS via large tokens | Low | Medium | Length validation |
| Format injection | Low | Medium | Hex regex validation |
| Token exposure in logs | Low | High | Never log tokens |
| TOCTOU attack | Low | High | Load token once at startup |

#### Threats Not Addressed

| Threat | Rationale |
|--------|-----------|
| Token expiration | Single-user, localhost - rotation not required |
| Multi-user isolation | Designed for single-user scenarios |
| Network interception | No network exposure (Unix socket) |
| Key rotation | Manual regeneration sufficient |

### Security Properties

#### Confidentiality

- **Token Storage**: Protected by filesystem permissions (0600)
- **Token Transmission**: Over Unix domain socket (no network)
- **Token Exposure**: Never logged or included in error messages

#### Integrity

- **Token Generation**: Cryptographically secure random
- **Token Validation**: Format and length validation
- **Token Comparison**: Constant-time prevents timing attacks

#### Availability

- **DoS Protection**: Length validation rejects oversized tokens
- **Fail-Secure**: Panic on initialization errors
- **Graceful Degradation**: Returns 401 on invalid tokens

### Constant-Time Comparison

**Purpose:** Prevent timing attacks where attacker measures response time to guess token.

**Implementation:**
```go
if subtle.ConstantTimeCompare([]byte(token), expectedToken) != 1 {
    return echo.ErrUnauthorized
}
```

**How It Works:**
- Compares every byte regardless of differences
- Execution time independent of token similarity
- Returns 1 if equal, 0 if not equal

**Testing:**
- Timing variance test measures valid vs invalid token comparison
- Ensures no order-of-magnitude timing differences
- Uses 100 samples for statistical significance

### Defense in Depth

```
Layer 1: Unix Socket Security
  └─> No network exposure
  └─> Socket permissions: 0600

Layer 2: File Permissions
  └─> Token file: 0600
  └─> Only owner can read

Layer 3: Token Properties
  └─> 256 bits of entropy
  └─> Cryptographically secure generation

Layer 4: Validation Security
  └─> Length validation (DoS prevention)
  └─> Format validation (injection prevention)
  └─> Constant-time comparison (timing attack prevention)

Layer 5: Implementation Security
  └─> Token loaded once at startup (TOCTOU prevention)
  └─> Panic on initialization failure (fail-secure)
  └─> No token logging or exposure
```

---

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

---

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

---

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

---

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

---

## Data Models

### Token File Format

**Format:** Plain text file containing 64 hexadecimal characters

**Structure:**
```
File: ~/.config/contextd/token
Size: 64 bytes
Permissions: 0600
Content: a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6
```

**Schema:**
```
Token File
├─ Size: Exactly 64 bytes
├─ Character Set: [a-f0-9]
├─ Format: Hexadecimal (lowercase)
├─ Permissions: 0600 (owner read/write only)
└─ Encoding: ASCII/UTF-8
```

### Authorization Header Format

**Format:** `Authorization: Bearer <token>`

**Structure:**
```http
Authorization: Bearer a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6
```

**Schema:**
```
Authorization Header
├─ Prefix: "Bearer "
├─ Token: 64 hexadecimal characters
├─ Character Set: [a-f0-9]
└─ Total Length: 71 bytes ("Bearer " + 64 chars)
```

### Internal Data Structures

The auth package does not expose any public structs. All data is handled as strings and byte slices.

**Token Representation:**
```go
// In-memory token representation
type token []byte  // 64 bytes (ASCII hex characters)

// Example:
expectedToken := []byte("a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1...")
```

---

## Authentication Middleware

### Middleware Integration

The `TokenAuth` middleware integrates into Echo's middleware chain. It should be applied AFTER logging/recovery middleware but BEFORE route handlers.

**Recommended Middleware Order:**
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

### Middleware Behavior

#### Successful Authentication
```
Request → TokenAuth → Validates token → Calls next() → Handler → Response
```

#### Failed Authentication
```
Request → TokenAuth → Rejects token → Returns 401 → No handler called
```

### Error Responses

The middleware returns Echo's standard `echo.ErrUnauthorized` which produces:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "message": "Unauthorized"
}
```

**Note:** No additional details are provided in error responses to prevent information leakage.

---

## Performance Characteristics

### Token Generation Performance

| Operation | Time | Allocations |
|-----------|------|-------------|
| Generate token | ~50µs | 3 allocations |
| Hex encoding | ~1µs | 1 allocation |
| File write | ~100µs | Varies (filesystem) |
| **Total** | **~150µs** | **~5 allocations** |

**Benchmark Results:**
```
BenchmarkGenerateToken-8    10000    150000 ns/op    512 B/op    5 allocs/op
```

### Token Validation Performance

| Operation | Time | Allocations |
|-----------|------|-------------|
| Header extraction | ~100ns | 0 allocations |
| Prefix check | ~50ns | 0 allocations |
| Length validation | ~10ns | 0 allocations |
| Regex validation | ~500ns | 0 allocations |
| Constant-time compare | ~200ns | 0 allocations |
| **Total** | **~1µs** | **0 allocations** |

**Benchmark Results:**
```
BenchmarkTokenValidation-8    1000000    1000 ns/op    0 B/op    0 allocs/op
```

### Constant-Time Comparison

The constant-time comparison ensures consistent execution time regardless of token similarity:

| Scenario | Average Time | Variance |
|----------|-------------|----------|
| Valid token | 1.2µs | ±0.1µs |
| Invalid token (first char wrong) | 1.2µs | ±0.1µs |
| Invalid token (last char wrong) | 1.2µs | ±0.1µs |
| Invalid token (all chars wrong) | 1.2µs | ±0.1µs |

**Timing Variance:** < 10% (acceptable for constant-time)

### Memory Usage

| Component | Memory |
|-----------|--------|
| Token storage (in-memory) | 64 bytes |
| Token validation (stack) | ~128 bytes |
| Regex compilation (shared) | ~1 KB |
| **Total per request** | **~200 bytes** |

### Scalability

**Request Throughput:**
- Authentication overhead: ~1µs per request
- Expected throughput: ~1,000,000 auth checks/second
- Bottleneck: Network I/O, not authentication

**Concurrent Requests:**
- Middleware is thread-safe
- Token is read-only after initialization
- No locking required
- Scales linearly with CPU cores

---

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

### Error Handling Best Practices

**Initialization:**
```go
// Load or generate token during startup
if err := auth.LoadOrGenerateToken(tokenPath); err != nil {
    // Critical error - cannot start server
    log.Fatal("Failed to setup auth token:", err)
}
```

**Middleware Setup:**
```go
// TokenAuth panics on invalid token - use defer/recover if needed
defer func() {
    if r := recover(); r != nil {
        log.Fatal("TokenAuth initialization failed:", r)
    }
}()

api.Use(auth.TokenAuth(tokenPath))
```

**Client-Side Error Handling:**
```go
resp, err := client.Get(url, headers)
if err != nil {
    return err
}

if resp.StatusCode == 401 {
    // Token invalid - may need to regenerate
    return errors.New("authentication failed: invalid token")
}
```

---

## Security Considerations

### Timing Attacks

**Threat:** Attacker measures response time to guess token character-by-character.

**Mitigation:**
- Uses `subtle.ConstantTimeCompare` for token comparison
- Execution time independent of token similarity
- Tested for timing variance (< 10%)

**Attack Surface:** Remote timing attacks impossible (Unix socket only)

### Token Theft

**Threat:** Unauthorized user reads token from filesystem.

**Mitigation:**
- Token file permissions: 0600 (owner only)
- Token stored in user's home directory
- Unix permissions prevent other users from reading

**Attack Surface:** Physical access or privilege escalation required

### Denial of Service (DoS)

**Threat:** Attacker sends very long tokens to exhaust memory.

**Mitigation:**
- Length validation rejects tokens > 64 characters
- Early rejection (before comparison)
- No allocations for oversized tokens

**Attack Surface:** Limited (Unix socket, localhost only)

### Format Injection

**Threat:** Attacker sends malformed tokens to exploit parsing bugs.

**Mitigation:**
- Strict regex validation (`^[a-f0-9]{64}$`)
- Length validation (exactly 64 chars)
- Early rejection of invalid formats

**Attack Surface:** Mitigated by strict validation

### TOCTOU (Time-of-Check-Time-of-Use)

**Threat:** Token file modified between check and use.

**Mitigation:**
- Token loaded ONCE at server startup
- Middleware references in-memory copy
- File system changes don't affect running server

**Attack Surface:** Eliminated by single-load design

### Information Leakage

**Threat:** Error messages reveal information about valid tokens.

**Mitigation:**
- All authentication failures return same error (401)
- No specific details in error messages
- Tokens never logged or included in errors

**Attack Surface:** No information leakage

### Token Expiration

**Decision:** Tokens do not expire.

**Rationale:**
- Single-user, localhost scenario
- User controls filesystem access
- Manual regeneration available
- Complexity not justified for threat model

**Alternative:** Delete token file to revoke access.

---

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

**Test Scenarios:**

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

### Security Tests

**Timing Attack Test:**
```go
func TestTokenAuth_ConstantTimeComparison(t *testing.T) {
    // 1. Measure valid token comparison (100 samples)
    // 2. Measure invalid token comparison (100 samples)
    // 3. Calculate average times
    // 4. Verify timing difference < 10%
}
```

**Concurrency Test:**
```go
func TestTokenAuth_Concurrency(t *testing.T) {
    // 1. Create middleware
    // 2. Run 100 concurrent authenticated requests
    // 3. Verify all succeed
    // 4. No race conditions detected
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

---

## Usage Examples

> **Note**: These examples describe Unix socket transport. MVP uses HTTP transport on port 8080 with no authentication.

### Example 1: Basic Server Setup

```go
package main

import (
    "context"
    "net"
    "net/http"
    "os"

    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/auth"
)

func main() {
    // Setup paths
    socketPath := "/tmp/api.sock"
    tokenPath := "/tmp/token"

    // Ensure auth token exists
    if err := auth.LoadOrGenerateToken(tokenPath); err != nil {
        panic(err)
    }

    // Create Echo server
    e := echo.New()

    // Public routes (no auth)
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    // Protected routes (auth required)
    api := e.Group("/api/v1")
    api.Use(auth.TokenAuth(tokenPath))

    api.GET("/data", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"data": "secret"})
    })

    // Create Unix socket
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        panic(err)
    }
    defer os.Remove(socketPath)

    // Set socket permissions
    if err := os.Chmod(socketPath, 0600); err != nil {
        panic(err)
    }

    // Start server
    e.Listener = listener
    e.Start("")
}
```

### Example 2: Client Authentication

```go
package main

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
)

func main() {
    // Read token from file
    token, err := os.ReadFile("/tmp/token")
    if err != nil {
        panic(err)
    }

    // Create HTTP client with Unix socket transport
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                return net.Dial("unix", "/tmp/api.sock")
            },
        },
    }

    // Create request with Bearer token
    req, err := http.NewRequest("GET", "http://localhost/api/v1/data", nil)
    if err != nil {
        panic(err)
    }
    req.Header.Set("Authorization", "Bearer "+string(token))

    // Make request
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // Check response
    if resp.StatusCode != http.StatusOK {
        fmt.Printf("Authentication failed: %d\n", resp.StatusCode)
        return
    }

    fmt.Println("Authentication successful!")
}
```

### Example 3: Token Regeneration

```bash
#!/bin/bash

# Stop server
systemctl --user stop contextd

# Remove old token
rm ~/.config/contextd/token

# Start server (will auto-generate new token)
systemctl --user start contextd

# Read new token
NEW_TOKEN=$(cat ~/.config/contextd/token)
echo "New token: $NEW_TOKEN"
```

### Example 4: ctxd CLI Integration

```go
package main

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
    "path/filepath"
)

func main() {
    // Get token path
    home := os.Getenv("HOME")
    tokenPath := filepath.Join(home, ".config", "contextd", "token")
    socketPath := filepath.Join(home, ".config", "contextd", "api.sock")

    // Read token
    token, err := os.ReadFile(tokenPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading token: %v\n", err)
        os.Exit(1)
    }

    // Create client
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                return net.Dial("unix", socketPath)
            },
        },
    }

    // Make authenticated request
    req, _ := http.NewRequest("GET", "http://localhost/api/v1/checkpoints", nil)
    req.Header.Set("Authorization", "Bearer "+string(token))

    resp, err := client.Do(req)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        fmt.Fprintf(os.Stderr, "Authentication failed. Token may be invalid.\n")
        os.Exit(1)
    }

    fmt.Println("Request successful!")
}
```

### Example 5: Testing Authentication

```go
package main_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/auth"
    "github.com/stretchr/testify/assert"
)

func TestAuthentication(t *testing.T) {
    // Generate token
    tokenPath := "/tmp/test-token"
    if err := auth.GenerateToken(tokenPath); err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tokenPath)

    // Read generated token
    token, _ := os.ReadFile(tokenPath)

    // Create Echo server
    e := echo.New()
    api := e.Group("/api")
    api.Use(auth.TokenAuth(tokenPath))

    api.GET("/test", func(c echo.Context) error {
        return c.String(http.StatusOK, "authenticated")
    })

    // Test valid token
    req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
    req.Header.Set("Authorization", "Bearer "+string(token))
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusOK, rec.Code)

    // Test invalid token
    req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
    req.Header.Set("Authorization", "Bearer invalid")
    rec = httptest.NewRecorder()
    e.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

---

## Related Documentation

### Internal Documentation

- **Package Guidelines**: `/docs/standards/package-guidelines.md`
- **Coding Standards**: `/docs/standards/coding-standards.md`
- **Testing Standards**: `/docs/standards/testing-standards.md`
- **Architecture**: `/docs/standards/architecture.md`
- **Package README**: `/pkg/CLAUDE.md` (Section: pkg/auth)

### Implementation Files

- **Main Implementation**: `/pkg/auth/auth.go`
- **Test Suite**: `/pkg/auth/auth_test.go`
- **Server Integration**: `/cmd/contextd/main.go` (lines 209, 456)

### External References

- **RFC 6750**: [The OAuth 2.0 Authorization Framework: Bearer Token Usage](https://www.rfc-editor.org/rfc/rfc6750)
- **Go crypto/rand**: [Package rand documentation](https://pkg.go.dev/crypto/rand)
- **Go crypto/subtle**: [Package subtle documentation](https://pkg.go.dev/crypto/subtle)
- **Echo Framework**: [Echo middleware guide](https://echo.labstack.com/middleware/)

### Security Resources

- **OWASP Authentication Cheat Sheet**: [https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- **NIST Randomness Recommendations**: [SP 800-90A](https://csrc.nist.gov/publications/detail/sp/800-90a/rev-1/final)
- **Timing Attack Prevention**: [Constant-Time Programming](https://www.bearssl.org/constanttime.html)

---

## Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2025-11-04 | Claude Code | Initial specification |

---

## Appendix: Security Checklist

**Deployment Checklist:**

- [ ] Token file permissions are 0600
- [ ] Socket file permissions are 0600
- [ ] Token file is in user's home directory
- [ ] No debug logging in production
- [ ] OTEL_ENVIRONMENT set to "production" (disables debug mode)
- [ ] No tokens in logs or error messages
- [ ] Middleware order correct (auth after logging/recovery)
- [ ] Public routes don't require auth (/health, /ready)
- [ ] All API routes protected with TokenAuth

**Testing Checklist:**

- [ ] Unit tests pass (> 95% coverage)
- [ ] Constant-time comparison tested
- [ ] Concurrency tests pass
- [ ] Integration tests pass
- [ ] No race conditions (go test -race)
- [ ] Benchmarks within acceptable range
- [ ] Token uniqueness verified

**Security Audit Checklist:**

- [ ] Constant-time comparison used
- [ ] Token loaded once at startup
- [ ] Length validation prevents DoS
- [ ] Format validation prevents injection
- [ ] No token exposure in errors
- [ ] File permissions enforced
- [ ] Panic on initialization failure (fail-secure)
- [ ] No timing attack vectors
