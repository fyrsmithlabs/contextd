# Contributing to ContextD

Thank you for your interest in contributing to ContextD! This guide will help you get started.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

---

## Code of Conduct

Be respectful, inclusive, and constructive. We're all here to build something useful together.

---

## Getting Started

### Prerequisites

- **Go 1.25+** - Required for building
- **Docker** - For running the full stack
- **Qdrant** - Vector database (included in Docker image)

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd

# Install dependencies
go mod download

# Run tests
go test ./... -v

# Build the binary
go build -o contextd ./cmd/contextd
```

---

## Development Setup

### Running Locally

ContextD requires Qdrant for vector storage. The easiest approach is to use Docker:

```bash
# Start Qdrant
docker run -d --name qdrant -p 6333:6333 -p 6334:6334 qdrant/qdrant:v1.12.1

# Run ContextD
go run ./cmd/contextd
```

### Environment Variables

See [docs/configuration.md](docs/configuration.md) for all options. Key development settings:

```bash
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
export EMBEDDINGS_PROVIDER=fastembed
export OTEL_ENABLE=false  # Disable telemetry for local dev
```

### IDE Setup

**VS Code**: Install the Go extension and ensure `gopls` is configured.

**GoLand**: Works out of the box with the Go module.

---

## Project Structure

```
contextd/
├── cmd/
│   └── contextd/           # Main entry point
├── internal/               # Private packages
│   ├── mcp/                # MCP server and handlers
│   │   └── handlers/       # Tool handler implementations
│   ├── reasoningbank/      # Cross-session memory
│   ├── checkpoint/         # Context snapshots
│   ├── remediation/        # Error pattern tracking
│   ├── vectorstore/        # Qdrant interface
│   ├── embeddings/         # Embedding providers
│   ├── secrets/            # Secret scrubbing (gitleaks)
│   ├── compression/        # Context compression
│   ├── hooks/              # Lifecycle hooks
│   ├── config/             # Configuration (Koanf)
│   ├── logging/            # Structured logging (Zap)
│   ├── telemetry/          # OpenTelemetry
│   ├── tenant/             # Multi-tenancy
│   ├── project/            # Project management
│   ├── repository/         # Code indexing
│   ├── troubleshoot/       # Diagnostics
│   ├── http/               # HTTP server
│   ├── services/           # Service registry
│   └── qdrant/             # Qdrant gRPC client
├── deploy/                 # Deployment files
│   ├── entrypoint.sh       # Docker entrypoint
│   └── supervisord.conf    # Process management
├── docs/                   # Documentation
│   ├── api/                # API reference
│   └── spec/               # Specifications
└── Dockerfile              # Container build
```

### Package Guidelines

| Directory | Purpose | Visibility |
|-----------|---------|------------|
| `cmd/` | Entry points | Public |
| `internal/` | Implementation | Private |
| `pkg/` | Reusable libraries | Public (future) |
| `docs/` | Documentation | Public |

---

## Coding Standards

### Go Style

Follow standard Go conventions:

- Run `gofmt` before committing
- Use meaningful variable names
- Keep functions focused and small
- Document exported types and functions

### Error Handling

Wrap errors with context:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to connect to qdrant: %w", err)
}

// Bad
if err != nil {
    return err
}
```

### Logging

Use structured logging with Zap:

```go
logger.Info("operation completed",
    zap.String("session_id", sessionID),
    zap.Int("count", count),
)
```

**Log Levels:**
- `Debug`: Detailed troubleshooting info
- `Info`: Normal operations
- `Warn`: Unexpected but recoverable
- `Error`: Failures requiring attention

### Configuration

Add new config options to `internal/config/config.go`:

```go
type Config struct {
    // ... existing fields
    NewFeature NewFeatureConfig `koanf:"new_feature"`
}

type NewFeatureConfig struct {
    Enabled bool   `koanf:"enabled"`
    Limit   int    `koanf:"limit"`
}
```

Document in `docs/configuration.md`.

---

## Testing

### Running Tests

```bash
# All tests
go test ./... -v

# With coverage
go test ./... -cover

# Specific package
go test ./internal/secrets/... -v

# With race detection
go test ./... -race
```

### Test Patterns

We use table-driven tests with `testify`:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "result",
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| `secrets` | 90%+ | 97% |
| `project` | 90%+ | 97% |
| `reasoningbank` | 80%+ | 82% |
| `remediation` | 80%+ | 82% |
| Others | 70%+ | Varies |

### Test Categories

- **Unit tests**: `*_test.go` in the same package
- **Integration tests**: `*_integration_test.go` (may require external services)

Run integration tests with:

```bash
go test ./... -tags=integration
```

---

## Submitting Changes

### Branch Naming

```
feature/short-description
fix/issue-number-description
docs/what-changed
refactor/component-name
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `chore`: Maintenance

**Examples:**

```
feat(memory): add confidence decay over time

Memories now lose confidence if not accessed within 30 days.
This prevents stale strategies from dominating search results.

Closes #123
```

```
fix(checkpoint): handle empty session gracefully

Previously, checkpoint_save would panic with nil session.
Now returns a clear error message.
```

### Pull Request Process

1. **Create a branch** from `main`
2. **Make your changes** with tests
3. **Run tests locally**: `go test ./... -race`
4. **Update documentation** if needed
5. **Push and create PR**
6. **Address review feedback**
7. **Squash and merge** when approved

### PR Template

```markdown
## Summary

Brief description of changes.

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation
- [ ] Refactoring

## Testing

How was this tested?

## Checklist

- [ ] Tests pass locally
- [ ] Documentation updated
- [ ] No new warnings
```

---

## Release Process

Releases are automated via GitHub Actions when tags are pushed.

### Versioning

We use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes

### Creating a Release

```bash
# Tag the release
git tag -a v1.2.3 -m "Release v1.2.3"

# Push the tag
git push origin v1.2.3
```

GitHub Actions will:
1. Build the Docker image
2. Push to `ghcr.io/fyrsmithlabs/contextd`
3. Tag with version number and `latest`

---

## Architecture Decisions

Major changes should be discussed before implementation:

1. **Open an issue** describing the proposal
2. **Discuss** alternatives and tradeoffs
3. **Document** the decision in the code or docs
4. **Implement** with tests

### Key Design Principles

- **Simplicity**: Prefer simple solutions
- **Security**: Scrub all output, assume hostile input
- **Performance**: Optimize for sub-100ms tool responses
- **Testability**: Design for easy testing

---

## Getting Help

- **Issues**: https://github.com/fyrsmithlabs/contextd/issues
- **Discussions**: Use GitHub Issues for now

---

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (TBD).
