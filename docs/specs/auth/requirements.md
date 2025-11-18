# Authentication Requirements

**Parent**: [../SPEC.md](../SPEC.md)

## Design Philosophy

- **Local-First Security**: Optimized for Unix socket transport with no network exposure
- **Zero External Dependencies**: No external authentication services required
- **Automatic Setup**: Token auto-generation on first run for seamless user experience
- **Timing Attack Prevention**: Constant-time token comparison prevents timing-based attacks
- **File Permission Hardening**: Strict 0600 permissions on token files
- **Fail-Secure**: Panics on initialization errors to prevent insecure startup

## Key Characteristics

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

## Scope

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

## Core Features

### 1. Secure Token Generation

- Cryptographically secure random token generation
- 256 bits of entropy (32 bytes)
- Hex-encoded for safe transmission (64 characters)
- Uses `crypto/rand` from Go standard library

### 2. Automatic Token Setup

- Token auto-generated on first run if missing
- Automatic permission fixing (0600)
- Validation of existing tokens
- Idempotent initialization

### 3. Token Validation

- Length validation (exactly 64 characters)
- Format validation (hexadecimal only)
- Constant-time comparison prevents timing attacks
- DoS protection (rejects oversized tokens)

### 4. Echo Middleware Integration

- Drop-in middleware for Echo framework
- Token loaded once during initialization
- Validates Bearer token format
- Returns standard HTTP 401 on failure

### 5. Security Hardening

- Token loaded once at startup (prevents TOCTOU attacks)
- Strict file permissions (0600)
- Regex validation for token format
- Length limits prevent DoS attacks
- Panic on initialization failure (fail-secure)

## Capabilities Matrix

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
