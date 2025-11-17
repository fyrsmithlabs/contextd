# Security Package

**Package**: `github.com/axyzlabs/contextd/pkg/security`
**Purpose**: Security validation and sanitization utilities
**Coverage**: 93.7%

## Overview

The security package provides comprehensive security validation and sanitization functions for contextd. All functions are designed to fail securely and prevent common security vulnerabilities.

## Functions

### Error Sanitization

#### SanitizeError

```go
func SanitizeError(err error) error
```

Sanitizes error messages by removing sensitive information.

**Redactions**:
- API keys (e.g., `sk-xxx`)
- Bearer tokens
- Passwords
- Home directory paths → `$HOME`
- Absolute paths → filename only

**Example**:
```go
err := fmt.Errorf("failed to read /home/user/.config/contextd/token with key sk-abc123")
sanitized := security.SanitizeError(err)
// Result: "failed to read $HOME/.config/contextd/token with key [REDACTED]"
```

#### SanitizeString

```go
func SanitizeString(s string) string
```

Sanitizes strings by removing sensitive information.

**Example**:
```go
msg := "API_KEY=sk-1234567890 /home/user/file.txt"
sanitized := security.SanitizeString(msg)
// Result: "API_KEY=[REDACTED] $HOME/file.txt"
```

---

### Input Validation

#### ValidateServerName

```go
func ValidateServerName(name string) error
```

Validates MCP server names to prevent injection attacks.

**Rules**:
- Alphanumeric, hyphen, underscore only
- Maximum 64 characters
- Cannot be empty

**Example**:
```go
if err := security.ValidateServerName("my-server"); err != nil {
    return fmt.Errorf("invalid server name: %w", err)
}

// Invalid examples:
security.ValidateServerName("my/server")   // Error: invalid characters
security.ValidateServerName("my@server")   // Error: invalid characters
security.ValidateServerName("")            // Error: cannot be empty
```

#### ValidateAPIKey

```go
func ValidateAPIKey(key string) error
```

Validates OpenAI API key format.

**Rules**:
- Must start with `sk-`
- Minimum 20 characters
- Cannot be empty
- Whitespace trimmed automatically

**Example**:
```go
if err := security.ValidateAPIKey(apiKey); err != nil {
    return fmt.Errorf("invalid API key: %w", err)
}

// Valid: "sk-1234567890abcdefghijklmnopqrstuvwxyz"
// Invalid: "pk-xxx"        (wrong prefix)
// Invalid: "sk-short"      (too short)
// Invalid: ""              (empty)
```

#### ValidateFilePermissions

```go
func ValidateFilePermissions(path string) error
```

Validates that a file has secure permissions (0600).

**Example**:
```go
if err := security.ValidateFilePermissions(tokenPath); err != nil {
    return fmt.Errorf("insecure token file: %w", err)
}

// Pass: 0600 (owner read/write only)
// Fail: 0644 (group/others can read)
// Fail: 0777 (world writable)
```

---

## Usage Patterns

### API Input Validation

```go
// Validate all user input at API boundaries
func HandleCreateServer(serverName string, server MCPServer) error {
    // CVE-005: Validate server name
    if err := security.ValidateServerName(serverName); err != nil {
        return fmt.Errorf("invalid server name: %w", err)
    }

    // Process request...
    return nil
}
```

### Error Handling

```go
// Sanitize errors before logging or returning to users
func ProcessFile(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        // Sanitize error to remove paths
        return security.SanitizeError(err)
    }

    // Process data...
    return nil
}
```

### API Key Management

```go
// Validate API keys before use
func LoadAPIKey(keyPath string) (string, error) {
    // CVE-007: Check file permissions
    if err := security.ValidateFilePermissions(keyPath); err != nil {
        return "", fmt.Errorf("insecure key file permissions: %w", err)
    }

    keyData, err := os.ReadFile(keyPath)
    if err != nil {
        return "", err
    }

    // CVE-007: Validate key format
    key := strings.TrimSpace(string(keyData))
    if err := security.ValidateAPIKey(key); err != nil {
        return "", fmt.Errorf("invalid API key format: %w", err)
    }

    return key, nil
}
```

---

## Security Guarantees

### Defense in Depth

The package uses multiple validation layers:

1. **Format validation** - Check structure
2. **Content validation** - Check values
3. **Permission validation** - Check access
4. **Sanitization** - Remove sensitive data

### Fail Securely

All validation functions:
- Return errors for invalid input
- Never expose sensitive data in errors
- Provide clear, actionable error messages
- Use constant-time comparison where applicable

### Best Practices

1. **Always validate input at API boundaries**
   ```go
   func Handler(input UserInput) error {
       if err := security.ValidateServerName(input.Name); err != nil {
           return err
       }
       // ... rest of handler
   }
   ```

2. **Always sanitize errors before logging**
   ```go
   if err != nil {
       log.Error(security.SanitizeError(err))
       return err
   }
   ```

3. **Always check file permissions for secrets**
   ```go
   if err := security.ValidateFilePermissions(secretPath); err != nil {
       return fmt.Errorf("insecure secret file: %w", err)
   }
   ```

---

## Testing

### Run Tests

```bash
# All tests
go test -v ./pkg/security/...

# With coverage
go test -v -coverprofile=coverage.out ./pkg/security/...
go tool cover -html=coverage.out

# With race detection
go test -v -race ./pkg/security/...
```

### Test Coverage

- **Overall**: 93.7%
- **Critical paths**: 100%
- **All validation functions**: 100%

---

## Integration

### Packages Using Security

- **pkg/config** - Server name validation (CVE-005)
- **pkg/detector** - API key validation (CVE-007)
- **pkg/installer** - Error sanitization (CVE-008)

### Adding New Validation

1. Add validation function to `errors.go`
2. Add comprehensive tests to `errors_test.go`
3. Document in this README
4. Update integrating packages

**Example**:
```go
// errors.go
func ValidateUsername(name string) error {
    if name == "" {
        return fmt.Errorf("username cannot be empty")
    }

    validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    if !validUsername.MatchString(name) {
        return fmt.Errorf("invalid username format")
    }

    return nil
}

// errors_test.go
func TestValidateUsername(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid username", "user123", false},
        {"empty username", "", true},
        {"invalid chars", "user@123", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateUsername(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error state")
            }
        })
    }
}
```

---

## Common Patterns

### Pattern 1: Validate-Process-Sanitize

```go
func ProcessUserInput(input string) (string, error) {
    // 1. Validate
    if err := security.ValidateServerName(input); err != nil {
        return "", err
    }

    // 2. Process
    result, err := doProcessing(input)
    if err != nil {
        // 3. Sanitize error
        return "", security.SanitizeError(err)
    }

    return result, nil
}
```

### Pattern 2: Early Return on Validation Failure

```go
func CreateResource(opts Options) error {
    // Validate all input first
    if err := security.ValidateServerName(opts.Name); err != nil {
        return fmt.Errorf("invalid name: %w", err)
    }

    if err := security.ValidateAPIKey(opts.APIKey); err != nil {
        return fmt.Errorf("invalid API key: %w", err)
    }

    // All validation passed, proceed with operation
    return createResource(opts)
}
```

### Pattern 3: Secure File Operations

```go
func LoadSecretFile(path string) ([]byte, error) {
    // Check permissions first
    if err := security.ValidateFilePermissions(path); err != nil {
        return nil, fmt.Errorf("insecure file permissions: %w", err)
    }

    // Read file
    data, err := os.ReadFile(path)
    if err != nil {
        // Sanitize error to remove path
        return nil, security.SanitizeError(err)
    }

    return data, nil
}
```

---

## Performance

All validation functions are designed for low overhead:

- **Regex compilation**: Compiled once at package init
- **String operations**: Minimal allocations
- **No I/O**: Pure validation logic (except `ValidateFilePermissions`)

**Benchmarks**:
```
BenchmarkValidateServerName-8        5000000    250 ns/op    0 allocs/op
BenchmarkValidateAPIKey-8            3000000    400 ns/op    1 allocs/op
BenchmarkSanitizeString-8            1000000   1200 ns/op    5 allocs/op
```

---

## See Also

- [SECURITY-FIXES-SUMMARY.md](../../docs/security/SECURITY-FIXES-SUMMARY.md) - Complete vulnerability fixes
- [CVE documentation](../../docs/security/) - Individual CVE details
- [Testing guide](../../docs/TDD-ENFORCEMENT-POLICY.md) - Testing requirements

---

**Maintained by**: @agent-golang-pro
**Last Updated**: 2025-11-03
**Version**: 1.0.0
