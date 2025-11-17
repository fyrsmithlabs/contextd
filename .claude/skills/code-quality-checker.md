# Code Quality Checker Skill

**Type**: Reusable Skill
**Category**: Code Analysis
**Version**: 1.0.0

## Purpose

Runs comprehensive quality checks on code across multiple dimensions: formatting, linting, testing, security, and performance. Language-agnostic with language-specific extensions.

## Capabilities

- Run formatters (language-specific)
- Execute linters and static analyzers
- Verify test coverage thresholds
- Check for security vulnerabilities
- Detect race conditions
- Validate build success

## Input Format

```json
{
  "language": "go|python|typescript|rust|java",
  "scope": "quick|full|coverage|security",
  "paths": ["./pkg/user", "./cmd/server"],
  "coverage_threshold": 80,
  "fail_fast": false
}
```

## Output Format

```json
{
  "passed": true|false,
  "score": 92,
  "checks": {
    "formatting": {"passed": true, "issues": []},
    "linting": {"passed": true, "issues": []},
    "testing": {"passed": true, "coverage": 94.2, "threshold": 80},
    "security": {"passed": true, "vulnerabilities": []},
    "build": {"passed": true, "errors": []}
  },
  "summary": {
    "total_checks": 15,
    "passed_checks": 15,
    "failed_checks": 0,
    "warnings": 2
  },
  "next_steps": ["Fix warning in pkg/user/service.go:45"]
}
```

## Check Definitions

### Quick Scope (< 30s)

**Go**:
- `gofmt -l .` - Formatting check
- `go vet ./...` - Go tool vet
- `go build ./...` - Build verification

**Python**:
- `black --check .` - Formatting check
- `ruff check .` - Fast linting
- `python -m py_compile` - Syntax check

**TypeScript**:
- `prettier --check .` - Formatting check
- `eslint .` - Linting
- `tsc --noEmit` - Type check

### Full Scope (1-3 min)

Includes Quick scope plus:

**Go**:
- `golint ./...` - Style linting
- `staticcheck ./...` - Static analysis
- `go test ./...` - Run all tests
- `go test -race ./...` - Race detection

**Python**:
- `mypy .` - Type checking
- `pylint .` - Full linting
- `pytest` - Run all tests
- `bandit -r .` - Security scan

**TypeScript**:
- `eslint --max-warnings 0` - Strict linting
- `jest --coverage` - Test coverage
- `npm audit` - Security audit

### Coverage Scope (1-2 min)

Focus on test coverage only:

**Go**:
- `go test -coverprofile=coverage.out ./...`
- `go tool cover -func=coverage.out`
- Verify threshold (default: 80%)

**Python**:
- `pytest --cov --cov-report=term`
- Verify threshold

**TypeScript**:
- `jest --coverage --coverageThreshold`
- Verify threshold

### Security Scope (2-5 min)

Focus on security checks:

**Go**:
- `gosec ./...` - Security scanner
- `go list -json -m all | nancy sleuth` - Dependency vulnerabilities

**Python**:
- `bandit -r .` - Security linter
- `safety check` - Dependency vulnerabilities
- `pip-audit` - Package vulnerabilities

**TypeScript**:
- `npm audit` - Package vulnerabilities
- `eslint --plugin security` - Security rules
- `snyk test` - Deep security scan

## Language-Specific Extensions

### Go

```bash
# Standard checks
gofmt -w .
golint ./...
go vet ./...
staticcheck ./...

# Test checks
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...

# Security checks
gosec ./...
go list -json -m all | nancy sleuth

# Build check
go build ./...
```

### Python

```bash
# Standard checks
black .
ruff check --fix .
mypy .
pylint .

# Test checks
pytest --cov --cov-report=term
pytest --cov-fail-under=80

# Security checks
bandit -r .
safety check
pip-audit

# Build check
python -m py_compile **/*.py
```

### TypeScript

```bash
# Standard checks
prettier --write .
eslint --fix .
tsc --noEmit

# Test checks
jest --coverage
jest --coverage --coverageThreshold='{"global":{"branches":80,"functions":80,"lines":80,"statements":80}}'

# Security checks
npm audit fix
snyk test

# Build check
npm run build
```

## Usage Examples

### Example 1: Pre-Commit Quick Check

```
Input:
  language: "go"
  scope: "quick"
  paths: ["./"]
  fail_fast: true

Output:
  passed: false
  score: 75
  checks:
    formatting:
      passed: false
      issues: ["pkg/user/service.go needs formatting"]
    linting:
      passed: true
      issues: []
    build:
      passed: true
      errors: []
  summary:
    total_checks: 3
    passed_checks: 2
    failed_checks: 1
  next_steps: ["Run: gofmt -w pkg/user/service.go"]
```

### Example 2: Pre-PR Full Check

```
Input:
  language: "go"
  scope: "full"
  paths: ["./"]
  coverage_threshold: 80
  fail_fast: false

Output:
  passed: true
  score: 95
  checks:
    formatting: {passed: true, issues: []}
    linting: {passed: true, issues: []}
    testing: {passed: true, coverage: 94.2, threshold: 80}
    security: {passed: true, vulnerabilities: []}
    build: {passed: true, errors: []}
  summary:
    total_checks: 15
    passed_checks: 15
    failed_checks: 0
    warnings: 0
  next_steps: ["Ready for PR submission"]
```

## Integration

### With golang-pro Skill

```
1. golang-pro implements feature with TDD
2. golang-pro uses code-quality-checker (quick scope) after each cycle
3. If checks fail, fix issues before continuing
4. At task completion, run code-quality-checker (full scope)
5. Only mark complete if score ≥ 90
```

### With code-reviewer Agent

```
1. code-reviewer receives PR for review
2. code-reviewer uses code-quality-checker (full scope)
3. If score < 80, request fixes
4. If score ≥ 80, proceed with manual review
5. If score ≥ 95, auto-approve (if configured)
```

### With CI/CD Workflow

```
1. PR created triggers GitHub Actions
2. Workflow uses code-quality-checker (full + security scope)
3. If passed, mark PR checks as passed
4. If failed, comment with issues and block merge
```

## Performance Metrics

| Scope | Time Target | Checks | Languages |
|-------|-------------|--------|-----------|
| Quick | <30s | 3-5 | All |
| Full | 1-3min | 10-15 | All |
| Coverage | 1-2min | 3-5 | All |
| Security | 2-5min | 5-10 | All |

## Error Handling

### Common Errors

**Tool not found**:
```
Error: golint not installed
Solution: Install with: go install golang.org/x/lint/golint@latest
```

**Coverage below threshold**:
```
Error: Coverage 75.2% below threshold 80%
Solution: Add tests to increase coverage
Debug: go tool cover -html=coverage.out
```

**Security vulnerabilities**:
```
Error: 3 high-severity vulnerabilities found
Solution: Update dependencies: go get -u ./...
Debug: gosec -fmt=json ./...
```

## Configuration

### Project-Level Config

Create `.quality-checker.json` in project root:

```json
{
  "language": "go",
  "coverage_threshold": 85,
  "fail_fast": false,
  "skip_checks": [],
  "custom_commands": {
    "lint": "golangci-lint run ./..."
  }
}
```

### CI/CD Integration

```yaml
# .github/workflows/quality-check.yml
- name: Run Quality Checks
  run: |
    # Use code-quality-checker skill
    claude-code --skill code-quality-checker \
      --language go \
      --scope full \
      --coverage-threshold 80
```

## Version History

- **1.0.0** (2025-10-25): Initial version with Go, Python, TypeScript support
