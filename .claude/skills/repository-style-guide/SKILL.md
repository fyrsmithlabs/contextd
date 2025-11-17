# Repository Style Guide Enforcement

**Version**: 1.0
**Based on**: Analysis of 20+ top Go projects (Kubernetes, Prometheus, Grafana, etc.)

## Core Principles

1. **Simplicity First** - Start simple, add structure as needed
2. **Standard Layout** - Follow golang-standards/project-layout
3. **Security First** - Comprehensive scanning, SBOM, signed releases
4. **Context Efficiency** - Keep documentation concise and scannable
5. **Automation** - CI/CD, dependency updates, release management

## Directory Structure (ENFORCE)

### Required Structure
```
contextd/
├── cmd/                    # Main applications (one binary per subdirectory)
├── pkg/                    # Public libraries (stable, reusable APIs)
├── internal/              # Private application code
├── docs/                  # Documentation
├── .github/               # GitHub workflows, templates
└── test/                  # Integration tests (build-tagged)
```

### Anti-Patterns (FORBIDDEN)
- ❌ `/src/` - Not a Go convention (Java pattern)
- ❌ `/models/`, `/controllers/`, `/views/` - Avoid MVC in Go
- ❌ `/lib/` - Ambiguous; use `/pkg/` or `/internal/`
- ❌ Deeply nested directories - Keep flat and simple

### Root Files (ENFORCE)
```
Required:
- README.md (badges: 2-4 max, quick start, links)
- LICENSE.md (single file, not LICENSE + LICENSE.md)
- CLAUDE.md (project guide)
- go.mod, go.sum
- Makefile
- .gitignore
- .golangci.yml

Recommended:
- CHANGELOG.md (Keep a Changelog format)
- CONTRIBUTING.md
- CODE_OF_CONDUCT.md (Contributor Covenant)
- SECURITY.md
```

## Code Organization (ENFORCE)

### Package Design
- **Single Responsibility** - One clear purpose per package
- **Avoid**: `util`, `common`, `helpers`, `lib` packages
- **Good**: `auth`, `config`, `telemetry`, `checkpoint`

### Dependency Direction
```
cmd/ → internal/ → pkg/
       ↓
       pkg/ (independent)
```
- NO circular dependencies
- pkg/ packages must be independent

### Error Handling
```go
// REQUIRED: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}

// REQUIRED: Sentinel errors for comparison
var (
    ErrNotFound = errors.New("not found")
    ErrInvalid = errors.New("invalid input")
)
```

### Naming Conventions
- Packages: lowercase, single word, no underscores
- Functions: MixedCaps (camelCase private, PascalCase public)
- Interfaces: `-er` suffix for single-method (Reader, Writer, Searcher)
- NO stuttering: `auth.NewClient()` not `auth.NewAuthClient()`

## Documentation (ENFORCE)

### godoc Standards
```go
// REQUIRED: Package-level documentation
//
// Basic usage:
//
//  if err != nil {
//      log.Fatal(err)
//  }
//  defer client.Close()

// REQUIRED: Function documentation with parameters
// Search performs semantic similarity search.
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Search query text
//   - limit: Maximum results (1-100)
//
// Returns matching results or error.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error)
```

### CHANGELOG Format (ENFORCE Keep a Changelog)
```markdown
## [Unreleased]

### Added
- New features

### Changed
- Changes to existing

### Fixed
- Bug fixes

### Security
- Security fixes

## [1.2.0] - 2025-01-15
...
```

## Testing (ENFORCE)

### Organization
```
```

### Build Tags
```go
//go:build integration
// +build integration

```

### Coverage Requirements
- Overall: >80%
- Critical paths (auth, security): >90%
- Utilities: >70%

### Table-Driven Tests (REQUIRED)
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "test", false},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // Test implementation
        })
    }
}
```

## CI/CD (ENFORCE)

### Required Workflows
```
.github/workflows/
├── pr-validation.yml       # Lint, test, build
├── release.yml            # GoReleaser on tags
└── security-scan.yml      # gosec, govulncheck (weekly + PRs)
```

### Security Scanning (REQUIRED)
- `gosec` - Go security checker (SARIF upload)
- `govulncheck` - Vulnerability scanner
- Weekly scheduled scans + PR scans

### Dependabot (REQUIRED)
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
```

## Release Management (ENFORCE)

### Semantic Versioning (REQUIRED)
- Format: `v{MAJOR}.{MINOR}.{PATCH}`
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### GoReleaser (REQUIRED Features)
- Multi-platform builds (linux, darwin × amd64, arm64)
- Checksums
- SBOM generation
- Archives with LICENSE, README, CHANGELOG

## golangci-lint Configuration (ENFORCE)

### Minimum Enabled Linters
```yaml
linters:
  enable:
    - errcheck      # Unchecked errors
    - gosimple      # Simplifications
    - govet         # Suspicious constructs
    - staticcheck   # Advanced analysis
    - unused        # Unused code
    - gofmt         # Formatting
    - goimports     # Import formatting
    - revive        # Fast linter
    - gosec         # Security
```

### Settings
```yaml
linters-settings:
  gocyclo:
    min-complexity: 15
  dupl:
    threshold: 100
  gosec:
    severity: "low"
```

## Makefile (ENFORCE)

### Required Targets
```makefile
.PHONY: help build test lint clean

help:     ## Display help
build:    ## Build binaries
test:     ## Run tests with coverage
lint:     ## Run golangci-lint
clean:    ## Remove artifacts
```

## Git Workflow (ENFORCE)

### Commit Messages
- Use Conventional Commits
- Format: `type(scope): description`
- Types: feat, fix, docs, chore, refactor, test, ci

### PR Requirements
- Clear description
- Related issue link
- Tests updated
- Documentation updated
- CI passing
- Code review approval

## Enforcement Checklist

Before accepting any code changes, verify:

- [ ] Directory structure follows standard layout
- [ ] No forbidden directories (/src, /lib, MVC patterns)
- [ ] Package names are lowercase, single word, descriptive
- [ ] No circular dependencies
- [ ] Errors are wrapped with context
- [ ] godoc comments on public APIs
- [ ] Tests use table-driven approach
- [ ] Coverage >80% for new code
- [ ] golangci-lint passes
- [ ] Conventional commit format
- [ ] CHANGELOG.md updated (if user-facing)
- [ ] Root directory clean (no test scripts, binaries, logs)

## Quick Reference

**Good Package Structure:**
```
pkg/auth/
├── auth.go           # Main implementation
├── middleware.go     # Auth middleware
├── auth_test.go      # Tests
└── README.md         # Complex packages only
```

**Bad Package Structure:**
```
pkg/utils/           # Too generic
pkg/authPackage/     # Redundant suffix
pkg/auth_utils/      # Underscores
```

**Good Imports:**
```go
import (
    "context"
    "fmt"

    "github.com/external/package"

    "github.com/axyzlabs/contextd/pkg/auth"
)
```

## References

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Keep a Changelog](https://keepachangelog.com/)
- [Conventional Commits](https://conventionalcommits.org/)
- [Contributor Covenant](https://contributor-covenant.org/)
