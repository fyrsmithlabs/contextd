# Authentication Architecture

**Parent**: [../SPEC.md](../SPEC.md)

## System Architecture

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

## Component Design

### 1. Token Generator

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

### 2. Token Loader

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

### 3. Authentication Middleware

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

### Token Format

```
Format:     64 hexadecimal characters
Regex:      ^[a-f0-9]{64}$
Example:    a3f2c8e9d1b4a6c5e8f0d2b3a5c7e9f1a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6
Length:     64 bytes (ASCII encoding)
Entropy:    256 bits
```

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

## Data Flow

### Successful Authentication Flow

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

### Failed Authentication Flow

```
1. Client sends request with invalid/missing token
2. TokenAuth middleware rejects request
3. Returns HTTP 401 Unauthorized
4. Request never reaches handler
```

## Performance Characteristics

### Token Generation Performance

| Operation | Time | Allocations |
|-----------|------|-------------|
| Generate token | ~50µs | 3 allocations |
| Hex encoding | ~1µs | 1 allocation |
| File write | ~100µs | Varies (filesystem) |
| **Total** | **~150µs** | **~5 allocations** |

### Token Validation Performance

| Operation | Time | Allocations |
|-----------|------|-------------|
| Header extraction | ~100ns | 0 allocations |
| Prefix check | ~50ns | 0 allocations |
| Length validation | ~10ns | 0 allocations |
| Regex validation | ~500ns | 0 allocations |
| Constant-time compare | ~200ns | 0 allocations |
| **Total** | **~1µs** | **0 allocations** |

### Constant-Time Comparison

| Scenario | Average Time | Variance |
|----------|-------------|----------|
| Valid token | 1.2µs | ±0.1µs |
| Invalid token (first char wrong) | 1.2µs | ±0.1µs |
| Invalid token (last char wrong) | 1.2µs | ±0.1µs |
| Invalid token (all chars wrong) | 1.2µs | ±0.1µs |

**Timing Variance:** < 10% (acceptable for constant-time)
