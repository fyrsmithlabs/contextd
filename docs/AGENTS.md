# AGENTS.md - ContextD Development Guidelines

## Build/Lint/Test Commands

### Core Commands
- **Build**: `make build` or `go build -o contextd ./cmd/contextd`
- **Test all**: `make test` or `go test ./... -v`
- **Test single**: `go test -run TestName ./internal/package/...`
- **Test with race**: `make test-race` or `go test -race ./...`
- **Coverage**: `make coverage` or `go test -coverprofile=coverage.out ./...`
- **Lint**: `make lint` or `golangci-lint run --timeout=5m`
- **Format**: `make fmt` or `go fmt ./... && goimports -w -local github.com/fyrsmithlabs/contextd .`
- **Vet**: `make vet` or `go vet ./...`
- **Audit**: `make audit` (comprehensive: lint + vet + test + security)

### Development Workflow
- **Live reload**: `make dev-mcp` (Air live reload for MCP mode)
- **Watch tests**: `make test-watch` (continuous testing)
- **Setup dev env**: `make setup-dev` (installs all tools)

## Code Style Guidelines

### Go Standards
- **Go version**: 1.25+ (see go.mod)
- **Formatting**: `gofmt` + `goimports` with local prefix `github.com/fyrsmithlabs/contextd`
- **Naming**: Meaningful names, follow Go conventions (camelCase, PascalCase for exported)
- **Functions**: Keep focused and small, single responsibility
- **Documentation**: Document all exported types/functions with meaningful comments

### Import Ordering
```go
import (
    // Standard library
    "context"
    "fmt"
    "time"

    // Third-party
    "github.com/google/uuid"
    "go.uber.org/zap"

    // Local packages (alphabetical)
    "github.com/fyrsmithlabs/contextd/internal/checkpoint"
    "github.com/fyrsmithlabs/contextd/internal/qdrant"
)
```

### Error Handling
- **Wrap errors** with context: `return fmt.Errorf("failed to connect: %w", err)`
- **Never ignore errors** unless explicitly documented
- **Use sentinel errors** for expected failures

### Logging
- **Use structured logging** with Zap: `logger.Info("operation", zap.String("key", value))`
- **Log levels**: Debug (troubleshooting), Info (normal), Warn (recoverable), Error (attention needed)
- **Include context**: session_id, user_id, operation details

### Testing
- **Table-driven tests** with testify: `assert.Equal(t, expected, actual)`
- **Test naming**: `TestFunctionName` or `TestFunctionName_Scenario`
- **Coverage targets**: secrets/project (90%+), reasoningbank/remediation (80%+), others (70%+)
- **Integration tests**: Use `*_integration_test.go` with `-tags=integration`

### Configuration
- **Use Koanf** for config management
- **Environment variables** override file config
- **Document all options** in `docs/configuration.md`

### Architecture Patterns
- **Vectorstore abstraction** in `internal/vectorstore/` (chromem default, Qdrant optional)
- **Service interfaces** in `internal/services/`
- **Dependency injection** via service registry
- **Context cancellation** support throughout
- **OpenTelemetry** for observability
- **Secret scrubbing** on contextd MCP tool responses (gitleaks)

### Commit Messages
Follow Conventional Commits: `type(scope): description`
- **feat**: New features
- **fix**: Bug fixes
- **docs**: Documentation
- **test**: Tests
- **refactor**: Code changes
- **chore**: Maintenance

### Package Structure
- **cmd/**: Entry points (public)
- **internal/**: Implementation (private)
- **pkg/**: Reusable libraries (public, future)
- **docs/**: Documentation (public)

## No Cursor/Copilot Rules Found

No `.cursor/rules/` or `.github/copilot-instructions.md` files exist in this repository.