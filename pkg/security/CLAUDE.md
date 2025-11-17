# Package: security

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides comprehensive security utilities for contextd, including sensitive data redaction, path validation, safe type conversions, and sanitization. Automatically redacts API keys, tokens, passwords, and user paths from logs and error messages to prevent credential leakage. Prevents path traversal attacks and integer overflow vulnerabilities.

## Specification

**Full Spec**: Security utilities are documented in [`pkg/CLAUDE.md`](../CLAUDE.md)

**Quick Summary**:
- **Problem**: Logs, error messages, and file operations can accidentally leak credentials or have security vulnerabilities
- **Solution**: Multi-layered security utilities for redaction, validation, and safe operations
- **Key Features**:
  - Pattern-based redaction for 15+ credential types
  - Path traversal attack prevention with symlink resolution
  - Safe integer conversions preventing overflow
  - Sanitization for AI model inputs
  - Zero dependencies, pure Go implementation

## Architecture

**Design Pattern**: Stateless utility functions with regex pattern matching and validation

**Dependencies**:
- `regexp` - Pattern matching for sensitive data
- `path/filepath` - Secure path operations
- `math` - Safe integer bounds checking

**Used By**:
- `pkg/telemetry` - Redact traces and logs
- `pkg/mcp` - Redact error responses and validate paths
- `internal/handlers` - Redact HTTP error responses
- `pkg/backup` - Secure file operations
- `pkg/checkpoint` - Safe type conversions
- `pkg/remediation` - Safe type conversions
- `pkg/installer` - Path validation

## Key Components

### Redaction Functions

```go
// Global redactor instance (singleton pattern)
func Redact(input string) string
func SanitizeForAI(input string, maxLength int) string
func ContainsSensitiveData(input string) bool

// Custom redactor with pattern management
type Redactor struct {
    patterns []*Pattern
}

func NewRedactor() *Redactor
func (r *Redactor) Redact(input string) string
func (r *Redactor) RedactWithDetails(input string) (string, []string)
func (r *Redactor) AddPattern(name, regex, description string) error
```

### Path Validation Functions

```go
// Scoped file operations (prevents path traversal)
func ValidatePath(base, path string) error
func ScopedReadFile(base, path string) ([]byte, error)
func ScopedOpenFile(base, path string, flag int, perm os.FileMode) (*os.File, error)
func ScopedWriteFile(base, path string, data []byte, perm os.FileMode) error
func ScopedMkdirAll(base, path string, perm os.FileMode) error

// CLI path validation (absolute paths only)
func ValidateImportPath(path string) error
func ValidateExportPath(path string) error
```

### Safe Type Conversion Functions

```go
// Safe integer conversions (prevents G115 overflow)
func SafeUint64ToInt64(v uint64) (int64, error)
func SafeInt64ToInt32(v int64) (int32, error)
func SafeInt64ToUint32(v int64) (uint32, error)
func SafeIntToUint64(v int) (uint64, error)
func SafeIntToUint32(v int) (uint32, error)
func SafeIntToUint(v int) (uint, error)

// Sentinel errors for programmatic checking
var (
    ErrUint64Overflow   = errors.New("uint64 exceeds int64 maximum")
    ErrNegativeUnsigned = errors.New("negative value cannot convert to unsigned")
    ErrInt32Overflow    = errors.New("int64 exceeds int32 bounds")
    ErrUint32Overflow   = errors.New("value exceeds uint32 maximum")
)
```

### Sanitization Functions

```go
// Error and string sanitization
func SanitizeError(err error) error
func SanitizeString(s string) string
func ValidateServerName(name string) error
func ValidateAPIKey(key string) error
func ValidateFilePermissions(path string) error
```

## Usage Example

```go
import "github.com/axyzlabs/contextd/pkg/security"

// 1. Redact sensitive data from error message
errorMsg := "failed to connect: OPENAI_API_KEY=sk-abc123def456"
safeMsg := security.Redact(errorMsg)
// Output: "failed to connect: OPENAI_API_KEY=[REDACTED]"

// 2. Sanitize for AI models (redact + truncate + clean control chars)
aiInput := security.SanitizeForAI(errorMsg, 1000)

// 3. Validate paths to prevent traversal attacks
baseDir := "/var/data/contextd"
userPath := "../../etc/passwd" // Attack attempt!
if err := security.ValidatePath(baseDir, userPath); err != nil {
    // Error: "path traversal detected"
    return err
}

// 4. Safe file operations within base directory
data, err := security.ScopedReadFile(baseDir, "safe/path.json")
if err != nil {
    return err
}

// 5. Safe integer conversions
vectorDim := uint64(1536)
dimInt64, err := security.SafeUint64ToInt64(vectorDim)
if err != nil {
    // Handle overflow
    return err
}

// 6. Custom redactor with additional patterns
redactor := security.NewRedactor()
redactor.AddPattern("custom_token", `TOKEN-[0-9]+`, "Custom token format")
sanitized := redactor.Redact(input)

// 7. Detect sensitive data before logging
if security.ContainsSensitiveData(message) {
    log.Warn("Message contains sensitive data, redacting...")
    message = security.Redact(message)
}
log.Info(message)
```

## Testing

**Test Coverage**: 92.9% (Target: ≥80%)

**Key Test Files**:
- `redact_test.go` - Pattern matching tests, edge cases
- `paths_test.go` - Path validation tests
- `path_validation_test.go` - Path traversal attack tests
- `conversions_test.go` - Integer overflow tests
- `errors_test.go` - Error sanitization tests

**Running Tests**:
```bash
go test ./pkg/security/
go test -cover ./pkg/security/
go test -race ./pkg/security/
```

## Configuration

No configuration required. All patterns are hardcoded for security.

**Redacted Patterns** (15+ types):
- **API Keys**: `sk-...`, `AKIA...` (AWS), `key-...`, `API_KEY=...`
- **Tokens**: `Bearer ...`, `token: ...`, `TOKEN=...`, `ghp_...` (GitHub), `xox...` (Slack), JWT tokens
- **Passwords**: `password=...`, `pwd=...`
- **Private Keys**: PEM format, SSH keys
- **Connection Strings**: PostgreSQL, MySQL, MongoDB, Redis with embedded credentials
- **Environment Variables**: `.*_KEY=...`, `.*_SECRET=...`, `export API_KEY=...`
- **Authorization Headers**: `Authorization: ...`
- **Credit Cards**: 16-digit patterns (basic detection)
- **Email/Password Combos**: `email=... password=...`
- **File Paths**: `/home/<username>/...` → `/home/***/...` or `$HOME`

## Security Considerations

**CRITICAL Security Requirements**:

1. **Pattern Coverage**:
   - Patterns MUST cover all common credential formats
   - Add new patterns when new credential types are introduced
   - Test with real credential samples (redacted in tests)
   - Currently covers 15+ credential types

2. **Logging Security**:
   - ALWAYS use `security.Redact()` before logging errors
   - NEVER log raw user input without redaction
   - Apply to all telemetry, metrics, and traces
   - Use `SanitizeForAI()` before sending to AI models

3. **Path Traversal Prevention**:
   - ALWAYS use `ValidatePath()` before file operations with user input
   - Use `Scoped*` functions instead of direct `os.*` calls
   - Validates against: null bytes, absolute paths, `..` components
   - **Symlink Resolution**: Resolves symlinks to prevent symlink escape attacks
   - Rejects paths outside base directory after symlink resolution

4. **Integer Overflow Prevention**:
   - ALWAYS use `Safe*` conversion functions instead of direct casts
   - Prevents G115 gosec violations (integer overflow)
   - Returns descriptive errors for programmatic handling
   - Critical for vector dimensions, counts, and API parameters

5. **Limitations**:
   - ✅ **Protects against**: Accidental credential logging, trace leakage, path traversal, integer overflow
   - ❌ **Does NOT protect against**: Intentional credential theft, memory dumps, timing attacks
   - **Assumption**: Patterns catch common formats, not foolproof
   - **Symlink Security**: Resolves symlinks but may fail for non-existent paths (validates parent instead)

## Performance Notes

- **Redaction time**: ~10μs per call (regex compilation cached)
- **Path validation**: ~50μs per call (includes symlink resolution)
- **Safe conversions**: ~1μs per call (pure arithmetic)
- **Memory**: Negligible (no allocations for cache hits)
- **Overhead**: <0.1% in logging paths

**Pattern Compilation**:
- All regex patterns compiled once at package initialization
- Global redactor instance cached (singleton)
- Zero allocations for pattern matching (cached compiled regexes)

## Related Documentation

- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
- Telemetry: [`pkg/telemetry/CLAUDE.md`](../telemetry/CLAUDE.md)
- MCP: [`pkg/mcp/CLAUDE.md`](../mcp/CLAUDE.md)
- Coding Standards: [`docs/standards/coding-standards.md`](../../docs/standards/coding-standards.md)
