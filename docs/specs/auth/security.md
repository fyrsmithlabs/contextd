# Authentication Security Model

**Parent**: [../SPEC.md](../SPEC.md)

## Threat Model

### Assumptions

**Trusted:**
- The user's filesystem
- The user's local processes
- The operating system kernel

**Untrusted:**
- Network traffic (N/A - no network)
- Other users on the system
- Processes running as other users

### Threats Addressed

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Timing attack | Medium | High | Constant-time comparison |
| Token theft | Low | High | File permissions (0600) |
| DoS via large tokens | Low | Medium | Length validation |
| Format injection | Low | Medium | Hex regex validation |
| Token exposure in logs | Low | High | Never log tokens |
| TOCTOU attack | Low | High | Load token once at startup |

### Threats Not Addressed

| Threat | Rationale |
|--------|-----------|
| Token expiration | Single-user, localhost - rotation not required |
| Multi-user isolation | Designed for single-user scenarios |
| Network interception | No network exposure (Unix socket) |
| Key rotation | Manual regeneration sufficient |

## Security Properties

### Confidentiality

- **Token Storage**: Protected by filesystem permissions (0600)
- **Token Transmission**: Over Unix domain socket (no network)
- **Token Exposure**: Never logged or included in error messages

### Integrity

- **Token Generation**: Cryptographically secure random
- **Token Validation**: Format and length validation
- **Token Comparison**: Constant-time prevents timing attacks

### Availability

- **DoS Protection**: Length validation rejects oversized tokens
- **Fail-Secure**: Panic on initialization errors
- **Graceful Degradation**: Returns 401 on invalid tokens

## Constant-Time Comparison

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

## Defense in Depth

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

## Deployment Security Checklist

- [ ] Token file permissions are 0600
- [ ] Socket file permissions are 0600
- [ ] Token file is in user's home directory
- [ ] No debug logging in production
- [ ] OTEL_ENVIRONMENT set to "production" (disables debug mode)
- [ ] No tokens in logs or error messages
- [ ] Middleware order correct (auth after logging/recovery)
- [ ] Public routes don't require auth (/health, /ready)
- [ ] All API routes protected with TokenAuth

## Security Audit Checklist

- [ ] Constant-time comparison used
- [ ] Token loaded once at startup
- [ ] Length validation prevents DoS
- [ ] Format validation prevents injection
- [ ] No token exposure in errors
- [ ] File permissions enforced
- [ ] Panic on initialization failure (fail-secure)
- [ ] No timing attack vectors
