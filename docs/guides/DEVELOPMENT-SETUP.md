# Development Setup Guide

This guide covers setting up your development environment for contextd.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Development Workflow](#development-workflow)
- [Testing](#testing)

## Prerequisites

### Required

- **Go 1.21+**: [Install Go](https://go.dev/doc/install)
- **Git**: Version control
- **Docker**: For running Qdrant and monitoring stack

### Recommended

- **pre-commit**: Git hook framework (automated setup available)
- **golangci-lint**: Go linting tool
- **gosec**: Go security scanner
- **yamllint**: YAML linting

## Initial Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/axyzlabs/contextd.git
   cd contextd
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Build the project**:
   ```bash
   make build-all
   ```

4. **Set up pre-commit hooks** (recommended):
   ```bash
   ./scripts/setup-pre-commit.sh
   ```

## Pre-commit Hooks

Pre-commit hooks automatically check your code before each commit, catching issues early.

### Automatic Setup

Run the setup script to install and configure pre-commit hooks:

```bash
./scripts/setup-pre-commit.sh
```

This will:
- Install pre-commit (if not already installed)
- Configure git hooks
- Enable all quality checks

### Manual Setup

If you prefer manual setup:

```bash
# Install pre-commit
pip install pre-commit
# OR
brew install pre-commit

# Install git hooks
pre-commit install --hook-type pre-commit
pre-commit install --hook-type commit-msg
```

### What Gets Checked

Pre-commit hooks run the following checks:

#### Go Code Quality
- **gofmt**: Code formatting
- **goimports**: Import organization
- **go vet**: Go compiler checks
- **golangci-lint**: Comprehensive linting (see `.golangci.yml`)
- **go mod tidy**: Dependency management

#### Security
- **TruffleHog**: Secret detection and scanning
  - Detects hardcoded credentials, API keys, tokens
  - Scans only verified secrets (reduces false positives)
  - Runs on both pre-commit and pre-push stages
  - Use `trufflehog:ignore` comment to exclude false positives
- **gosec**: Security vulnerability scanning
  - Excludes `cmd/ctxd` directory
  - Scans for common security issues
  - Same configuration as CI pipeline

#### File Quality
- **trailing-whitespace**: Remove trailing whitespace
- **end-of-file-fixer**: Ensure files end with newline
- **check-yaml**: YAML syntax validation
- **check-added-large-files**: Prevent large file commits (>1MB)
- **detect-private-key**: Prevent accidental credential commits
- **yamllint**: YAML linting
- **markdownlint**: Markdown linting

#### Commit Quality
- **conventional-commits**: Enforce conventional commit message format

### Running Hooks Manually

```bash
# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run gosec --all-files
pre-commit run golangci-lint --all-files

# Run on staged files only (default behavior)
pre-commit run
```

### Skipping Hooks (Not Recommended)

In rare cases where you need to skip hooks:

```bash
# Skip all hooks
git commit --no-verify

# Skip specific checks (use with caution)
SKIP=gosec git commit
```

**Warning**: Skipping hooks may cause CI failures. Only skip when absolutely necessary.

### Updating Hooks

To update hook versions:

```bash
pre-commit autoupdate
```

### Troubleshooting

**Hooks are slow**:
- First run installs hook dependencies (one-time cost)
- Subsequent runs are cached and much faster
- Consider running specific hooks: `pre-commit run golangci-lint`

**Hooks fail on existing code**:
- Run `pre-commit run --all-files` to check all files
- Fix issues or update `.pre-commit-config.yaml` to exclude legacy code

**TruffleHog not found**:
- Install TruffleHog: `brew install trufflehog` (macOS)
- OR download from: https://github.com/trufflesecurity/trufflehog/releases
- OR use Docker: See `.pre-commit-config.yaml` for Docker configuration

**TruffleHog false positives**:
- Add `trufflehog:ignore` comment on the line with the false positive
- Example: `password := "test123" // trufflehog:ignore`
- Create `.trufflehog.yaml` config file to exclude patterns

**Python/pre-commit not found**:
- Install Python 3.x
- Install pre-commit: `pip3 install pre-commit`
- OR use Homebrew: `brew install pre-commit`

## Development Workflow

### Standard Workflow

1. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes**:
   - Write tests first (TDD)
   - Implement feature
   - Run tests: `go test ./...`

3. **Commit changes**:
   ```bash
   git add .
   git commit -m "feat: Add your feature description"
   # Pre-commit hooks run automatically
   ```

4. **Push and create PR**:
   ```bash
   git push origin feature/your-feature-name
   gh pr create
   ```

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
- `test`: Test additions/changes
- `chore`: Build process or auxiliary tool changes
- `perf`: Performance improvements
- `ci`: CI/CD changes

**Examples**:
```bash
git commit -m "feat(checkpoint): Add semantic search with Qdrant"
git commit -m "fix(auth): Prevent token timing attacks"
git commit -m "docs: Update API documentation"
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With race detection
go test -race ./...

# Specific package
go test ./pkg/checkpoint/...

# Verbose output
go test -v ./...
```

### Test Requirements

- Minimum coverage: 80% overall
- Core packages: 100% coverage
- All new features must include tests
- Bug fixes must include regression tests

See [TDD-ENFORCEMENT-POLICY.md](../TDD-ENFORCEMENT-POLICY.md) for details.

### Quality Checks

Before submitting a PR, run:

```bash
# Format code
gofmt -w .

# Lint
golangci-lint run

# Security scan
gosec -exclude-dir=cmd/ctxd ./...

# All checks (via pre-commit)
pre-commit run --all-files
```

## IDE Setup

### VS Code

Recommended extensions:
- Go (golang.go)
- golangci-lint (golangci.golangci-lint)
- YAML (redhat.vscode-yaml)
- Conventional Commits (vivaxy.vscode-conventional-commits)

Settings (`.vscode/settings.json`):
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "go.testOnSave": true
}
```

### GoLand / IntelliJ IDEA

1. Enable golangci-lint:
   - Settings → Tools → File Watchers
   - Add golangci-lint watcher

2. Configure goimports:
   - Settings → Tools → File Watchers
   - Add goimports watcher

3. Enable conventional commits plugin

## Additional Resources

- [TDD Enforcement Policy](../TDD-ENFORCEMENT-POLICY.md)
- [Code Review Guidelines](../CLAUDE.md#code-review-checklist)
- [Architecture Documentation](../architecture/)
- [Testing Standards](../standards/testing-standards.md)
- [Contributing Guidelines](../../CONTRIBUTING.md)

## Getting Help

- **Documentation**: Check `docs/` directory
- **Issues**: [GitHub Issues](https://github.com/axyzlabs/contextd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/axyzlabs/contextd/discussions)
