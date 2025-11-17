# Contributing to contextd by Fyrsmith Labs

Thank you for your interest in contributing to contextd! This project aims to provide a robust, context-optimized API service for Claude Code user-level management. We welcome contributions from the community and appreciate your help in making this project better.

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to **maintainers@fyrsmithlabs.com**.

## Security

For security vulnerabilities, please see our [Security Policy](SECURITY.md) and report issues to **security@fyrsmithlabs.com**.

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue on GitHub with the following information:

- **Clear title**: Summarize the issue in one line
- **Description**: Detailed explanation of the problem
- **Steps to reproduce**: Step-by-step instructions to reproduce the issue
- **Expected behavior**: What you expected to happen
- **Actual behavior**: What actually happened
- **Environment**:
  - Go version (`go version`)
  - OS and version
  - contextd version (`contextd --version`)
  - Relevant configuration (redact sensitive data)
- **Logs**: Relevant log output (use code blocks)
- **Additional context**: Screenshots, error messages, etc.

### Suggesting Features

We welcome feature suggestions! Please open an issue with:

- **Clear title**: Feature name or summary
- **Problem statement**: What problem does this solve?
- **Proposed solution**: How should it work?
- **Alternatives considered**: Other approaches you've thought about
- **Additional context**: Examples, mockups, related issues

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Add tests** for any new functionality (≥80% coverage required)
4. **Update documentation** to reflect your changes
5. **Ensure all tests pass** (`go test ./...`)
6. **Submit a pull request** with a clear description

## Development Setup

### Prerequisites

- **Go 1.21 or higher** - [Download](https://golang.org/dl/)
- **Vector Database** - Choose one:
  - Qdrant (recommended) via Docker
  - Hosted vector database cluster
- **TEI (Text Embeddings Inference)** - Optional, for local embeddings
- **OpenTelemetry Collector** (optional) - For observability during development

### Setting Up Pre-commit Hooks

We use [pre-commit](https://pre-commit.com/) to ensure code quality and consistency. Install and set up pre-commit hooks:

```bash
# Install pre-commit
pip install pre-commit  # or brew install pre-commit

# Install the hooks
pre-commit install

# (Optional) Run on all files to check current state
pre-commit run --all-files
```

**What the hooks do**:
- **go-fmt**: Format Go code with standard formatting
- **go-imports**: Organize and format Go imports
- **golangci-lint**: Comprehensive Go linting (same as CI)
- **go-vet**: Static analysis for common Go mistakes
- **go-mod-tidy**: Clean up go.mod and go.sum
- **go-test**: Run fast tests (no race detection for speed)
- **gosec**: Security vulnerability scanning
- **prettier**: Format YAML files
- **markdownlint**: Check markdown formatting
- **shellcheck**: Lint shell scripts
- **General checks**: Trailing whitespace, merge conflicts, large files, etc.

**Skipping hooks**: Use `git commit --no-verify` to skip pre-commit hooks (not recommended).

### Building the Project

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/contextd.git
cd contextd

# Build both binaries
make build-all

# Or build individually
go build -o contextd ./cmd/contextd/
go build -o ctxd ./cmd/ctxd/

# Install as systemd service (Linux)
./ctxd install
systemctl --user start contextd

# Or run directly
./contextd
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/auth

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Style

This project follows standard Go conventions:

- **Formatting**: All code must be formatted with `gofmt`
  ```bash
  gofmt -w .
  ```

- **Linting**: Code should pass `golangci-lint`
  ```bash
  # Install golangci-lint
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

  # Run linter
  golangci-lint run
  ```

- **Naming conventions**:
  - Use `camelCase` for variables and functions
  - Use `PascalCase` for exported types and functions
  - Package names should be lowercase, single word
  - Avoid stuttering (e.g., `auth.AuthToken` → `auth.Token`)

- **Comments**:
  - All exported functions, types, and constants must have doc comments
  - Doc comments should be complete sentences starting with the name
  - Example: `// NewClient creates a new API client with the given configuration.`

## Pull Request Process

### Branch Naming

Use descriptive branch names with a prefix:

- `feature/` - New features (e.g., `feature/checkpoint-search`)
- `fix/` - Bug fixes (e.g., `fix/socket-permissions`)
- `docs/` - Documentation updates (e.g., `docs/api-examples`)
- `refactor/` - Code refactoring (e.g., `refactor/handler-structure`)
- `test/` - Test additions/improvements (e.g., `test/auth-middleware`)
- `security/` - Security-related changes (e.g., `security/fix-filter-injection`)

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `security`: Security-related changes

**Examples**:
```
feat(auth): add token rotation support

Implement automatic token rotation every 90 days with graceful
transition period for existing tokens.

Closes #123
```

```
fix(telemetry): correct OTLP endpoint configuration

The OTLP exporter was using HTTP scheme instead of HTTPS,
causing connection failures in production.
```

```
security(multi-tenant): fix filter injection vulnerability

Remove legacy mode to prevent cross-project data access.
Mandatory multi-tenant architecture now enforced.

BREAKING CHANGE: Legacy mode removed, migration required.

Closes #60
```

### Maintaining the CHANGELOG

We use [Keep a Changelog](https://keepachangelog.com/) format for tracking changes.

**When to update CHANGELOG.md**:
- Add entry for every user-facing change
- Update the `[Unreleased]` section for each PR
- Group changes by type: Added, Changed, Deprecated, Removed, Fixed, Security

**How to add an entry**:

1. **Open CHANGELOG.md** and locate the `[Unreleased]` section
2. **Add your change** under the appropriate category:
   ```markdown
   ## [Unreleased]

   ### Added
   - New feature description with PR reference (#123)

   ### Fixed
   - Bug fix description (#124)

   ### Security
   - Security fix description (#125)
   ```
3. **Use present tense**: "Add feature" not "Added feature"
4. **Be specific**: Describe what changed and why
5. **Reference issues/PRs**: Include `(#123)` at the end

**Example entries**:
```markdown
### Added
- Checkpoint search with semantic similarity matching (#156)
- Support for custom embedding models via environment variables (#157)
- Per-tool rate limiting in MCP server (#58)

### Changed
- Improved Qdrant connection retry logic with exponential backoff (#158)
- Updated OpenTelemetry SDK to v1.21.0 (#159)

### Fixed
- Prevent null pointer dereference in remediation search (#160)
- Correct file permissions on Unix socket creation (#161)

### Security
- Fix filter injection vulnerability by removing legacy mode (#60)
- Add constant-time token comparison (#162)

### BREAKING CHANGES
- Remove legacy mode, multi-tenant architecture now mandatory (#60)
```

**What NOT to include**:
- Internal refactorings that don't affect users
- Test-only changes
- Documentation typo fixes
- Version bumps in go.mod (unless breaking)

**Release process**:
- Maintainers will move `[Unreleased]` items to a versioned section during releases
- GoReleaser automatically generates release notes from commit messages
- CHANGELOG.md provides the detailed, curated change history

**CHANGELOG Rotation**:

To prevent context bloat, we rotate old releases to archive files:

- **Main CHANGELOG.md**: Keep last 5 releases or 6 months (whichever is more)
- **Archives**: Older releases moved to `docs/changelogs/`
  - By year: `docs/changelogs/2024.md`
  - By version: `docs/changelogs/v0.x.md` (pre-1.0)

**When to rotate** (maintainers only):
- CHANGELOG.md exceeds 500 lines
- New major version released (e.g., v2.0.0)
- End of calendar year

**Benefits**:
- Main CHANGELOG stays < 500 lines (optimal for AI context)
- Full history preserved in archives
- Better GitHub PR diffs
- Faster file loading

### Pull Request Requirements

Before submitting a PR, ensure:

- [ ] All tests pass (`go test ./...`)
- [ ] Test coverage ≥80% for new code
- [ ] Code is formatted (`gofmt -w .`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] New functionality includes tests
- [ ] Documentation is updated (README, CLAUDE.md, godocs)
- [ ] CHANGELOG.md is updated
- [ ] Commit messages follow conventional commits
- [ ] PR description clearly explains the changes
- [ ] Related issues are referenced (e.g., "Closes #123")
- [ ] Security implications considered

### Review Process

1. A maintainer will review your PR within 1-2 weeks
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR
4. Your contribution will be included in the next release

## Project Structure

```
contextd/
├── cmd/
│   ├── contextd/           # Main server binary
│   │   └── main.go
│   └── ctxd/              # CLI client binary
│       └── main.go
├── pkg/                    # Public packages (reusable libraries)
│   ├── auth/              # Authentication and token management
│   ├── config/            # Configuration loading
│   ├── telemetry/         # OpenTelemetry setup
│   ├── vectorstore/       # Vector database interface
├── internal/              # Private application code
│   └── service/           # Business logic services
├── scripts/               # Build and deployment scripts
│   ├── install.sh         # Installation script
│   └── stack-manager.sh   # Local stack management
├── monitoring/            # Grafana dashboards and configs
├── docs/                  # Documentation
├── CLAUDE.md              # Project guidance for Claude Code
├── CODE_OF_CONDUCT.md     # Community guidelines
├── SECURITY.md            # Security policy
├── CONTRIBUTING.md        # This file
├── LICENSE                # MIT License
└── README.md              # Project overview
```

**Key principles**:
- `cmd/` contains application entry points
- `pkg/` contains reusable library code (can be imported by other projects)
- `internal/` contains private code (cannot be imported externally)
- Follow standard Go project layout conventions

## Testing Guidelines

### Unit Tests

- Test files should be named `*_test.go`
- Use table-driven tests for multiple scenarios
- Mock external dependencies (filesystem, network, databases)
- Aim for >80% code coverage (enforced in CI)

**Example**:
```go
func TestTokenGeneration(t *testing.T) {
    tests := []struct {
        name    string
        length  int
        wantErr bool
    }{
        {"valid length", 32, false},
        {"zero length", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            token, err := GenerateToken(tt.length)
            if (err != nil) != tt.wantErr {
                t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
            }
            // Add more assertions...
        })
    }
}
```

### Integration Tests

- Tag integration tests with build tags: `//go:build integration`
- Run with: `go test -tags=integration ./...`
- Require external services (Qdrant, etc.)
- Clean up resources after tests

### Testing the API

```bash
# Health check (no auth required)
curl --unix-socket ~/.config/contextd/api.sock http://localhost/health

# Authenticated endpoint
TOKEN=$(cat ~/.config/contextd/token)
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost/api/v1/checkpoints
```

### Testing MCP Tools

```bash
# Use Claude Code slash commands
/checkpoint save "test checkpoint"
/checkpoint search "recent work"
/remediation search "error message"
/troubleshoot "error details"
```

## Documentation Standards

### Code Documentation

- All exported functions, types, and constants must have godoc comments
- Comments should explain *why*, not just *what*
- Include examples for complex functions
- Keep comments up-to-date with code changes

### Project Documentation

- **README.md**: High-level overview, quick start, basic usage
- **CLAUDE.md**: Detailed guidance for Claude Code (architecture, commands, decisions)
- **CODE_OF_CONDUCT.md**: Community guidelines and enforcement
- **SECURITY.md**: Security policy and vulnerability reporting
- **CONTRIBUTING.md**: This file - contribution guidelines
- **docs/**: Additional documentation (architecture, implementation details, guides)
- **Inline comments**: Explain complex logic, edge cases, workarounds

### API Documentation

- Document all endpoints with:
  - HTTP method and path
  - Authentication requirements
  - Request parameters/body
  - Response format
  - Error codes
  - Examples

## Development Workflow

### Research-First Development

**All significant changes require research:**

1. **SDK Research** - Always search for existing SDKs/libraries first
2. **Documentation** - Document findings in `docs/research/`
3. **Review** - Get feedback on approach before implementation
4. **Implementation** - Follow researched approach

See [CLAUDE.md](CLAUDE.md) for detailed workflow.

### Test-Driven Development (TDD)

**TDD is mandatory:**

1. **Red** - Write failing test first
2. **Green** - Implement minimal code to pass
3. **Refactor** - Improve code while keeping tests green
4. **Repeat** - Continue until feature complete

See [docs/TDD-ENFORCEMENT-POLICY.md](docs/TDD-ENFORCEMENT-POLICY.md) for details.

### Bug Tracking

**Every bug requires:**

1. Bug documentation in `tests/regression/bugs/`
2. Reproduction steps
3. Regression test
4. Fix implementation
5. Test verification

See [CLAUDE.md](CLAUDE.md) for bug tracking process.

## Questions?

If you have questions about contributing, feel free to:

- Open a [GitHub Discussion](https://github.com/fyrsmithlabs/contextd/discussions)
- Comment on relevant issues
- Reach out to maintainers at **maintainers@fyrsmithlabs.com**

Thank you for contributing to contextd by Fyrsmith Labs!

---

Built with care by Fyrsmith Labs for Claude Code users who want persistent context and intelligent assistance.
