# Run Quality Gates

**Command**: `/run-quality-gates [scope]`

**Description**: Execute comprehensive quality checks with configurable scope, including vulnerability scanning.

**Usage**:
```
/run-quality-gates              # Default: full checks
/run-quality-gates quick        # Build, tests, format only
/run-quality-gates full         # All checks + race, linters, coverage
/run-quality-gates coverage     # Deep coverage analysis
/run-quality-gates security     # govulncheck + dependency audit
```

## Purpose

Provides fastest feedback loop during development by running appropriate quality checks based on the current development phase. Integrates vulnerability scanning using `govulncheck` to catch security issues early.

## Agent Workflow

When this command is invoked, execute the quality gates script with the specified scope:

```bash
# Run the quality gates script
.scripts/run-quality-gates.sh [scope]
```

## Scope Definitions

### Quick Scope
**Purpose**: Fast feedback during active development
**Checks**:
- `go build ./...` - Verify code compiles
- `go test ./...` - Run all tests
- `gofmt -w .` - Format code
- **Time**: ~30 seconds

**When to use**: After making code changes, before committing

### Full Scope (Default)
**Purpose**: Comprehensive quality verification
**Checks**:
- All "quick" checks
- `go test -race ./...` - Race condition detection
- `golint ./...` - Linting
- `go vet ./...` - Static analysis
- `staticcheck ./...` - Advanced static analysis
- Coverage report (≥80% required)
- **Time**: ~2-3 minutes

**When to use**: Before creating PR, during PR updates

### Coverage Scope
**Purpose**: Deep coverage analysis
**Checks**:
- Generate coverage profile
- Coverage report by package
- Identify uncovered critical paths
- Suggest test cases for gaps
- **Time**: ~1-2 minutes

**When to use**: When improving test coverage, responding to coverage issues

### Security Scope
**Purpose**: Security vulnerability scanning
**Checks**:
- `govulncheck ./...` - Go vulnerability database check
- `go list -m all` - List all dependencies
- Check for known CVEs
- Report vulnerable dependencies with links
- **Time**: ~1 minute

**When to use**: Before releases, weekly security checks, responding to security alerts

## Implementation Script

Location: `.scripts/run-quality-gates.sh`

```bash
#!/bin/bash
# Script: run-quality-gates.sh
# Purpose: Run quality gates with configurable scope

set -e

SCOPE=${1:-full}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Running Quality Gates - Scope: $SCOPE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Quick checks (always run)
if [[ "$SCOPE" == "quick" || "$SCOPE" == "full" || "$SCOPE" == "coverage" ]]; then
    echo "📦 Building..."
    go build ./... || { echo "❌ Build failed"; exit 1; }

    echo "🧪 Running tests..."
    go test ./... || { echo "❌ Tests failed"; exit 1; }

    echo "💅 Formatting code..."
    gofmt -w .

    if [[ "$SCOPE" == "quick" ]]; then
        echo ""
        echo "✅ Quick quality gates passed!"
        exit 0
    fi
fi

# Full checks
if [[ "$SCOPE" == "full" ]]; then
    echo "🏃 Checking for race conditions..."
    go test -race ./... || { echo "❌ Race conditions detected"; exit 1; }

    echo "🔍 Running linters..."
    golint ./... || { echo "⚠️  Linting warnings found"; }
    go vet ./... || { echo "❌ go vet failed"; exit 1; }
    staticcheck ./... || { echo "❌ staticcheck failed"; exit 1; }

    echo "📊 Checking coverage..."
    go test ./... -coverprofile=coverage.out -covermode=atomic
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
        echo "❌ Coverage $COVERAGE% is below 80% minimum"
        exit 1
    fi

    echo "✅ Coverage: $COVERAGE%"
    echo ""
    echo "✅ All quality gates passed!"
    exit 0
fi

# Coverage analysis
if [[ "$SCOPE" == "coverage" ]]; then
    echo "📊 Generating detailed coverage report..."
    go test ./... -coverprofile=coverage.out -covermode=atomic

    echo ""
    echo "Coverage by package:"
    go tool cover -func=coverage.out

    echo ""
    echo "Generating HTML report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "✅ HTML report: coverage.html"

    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo ""
    echo "Overall coverage: $COVERAGE%"

    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
        echo "⚠️  Coverage below 80% minimum"
        exit 1
    fi

    echo "✅ Coverage analysis complete!"
    exit 0
fi

# Security checks
if [[ "$SCOPE" == "security" ]]; then
    echo "🔐 Running security checks..."

    # Check if govulncheck is installed
    if ! command -v govulncheck &> /dev/null; then
        echo "Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi

    echo "Scanning for vulnerabilities..."
    govulncheck ./... || { echo "⚠️  Vulnerabilities found"; }

    echo ""
    echo "Checking dependencies..."
    go list -m all

    echo ""
    echo "✅ Security scan complete!"
    exit 0
fi

echo "❌ Unknown scope: $SCOPE"
echo "Valid scopes: quick, full, coverage, security"
exit 1
```

## Integration with Existing Workflows

This command enhances the existing `.scripts/pre-pr.sh` script by providing:
- **Granular control**: Choose appropriate scope for current task
- **Faster feedback**: Quick scope for rapid iteration
- **Security focus**: Dedicated security scanning
- **Better diagnostics**: Detailed coverage analysis

## Success Criteria

- ✅ All scopes execute without errors
- ✅ Quick scope completes in <30 seconds
- ✅ Full scope catches all quality issues
- ✅ Coverage scope generates actionable reports
- ✅ Security scope identifies vulnerabilities with CVE links

## Example Workflows

### During Development
```bash
# Make changes
vim pkg/mypackage/feature.go

# Quick check
/run-quality-gates quick

# Continue development...
```

### Before Creating PR
```bash
# Comprehensive check
/run-quality-gates full

# If coverage issues found
/run-quality-gates coverage

# Fix gaps, then recheck
/run-quality-gates full
```

### Weekly Security Review
```bash
# Security scan
/run-quality-gates security

# If vulnerabilities found, update dependencies
go get -u ./...

# Recheck
/run-quality-gates security
```

## Error Handling

If any check fails:
1. **Build failures**: Review compiler errors in output
2. **Test failures**: Check specific test output
3. **Race conditions**: Review race detector output
4. **Linting failures**: Fix issues or justify exceptions
5. **Coverage gaps**: Use coverage report to identify untested code
6. **Vulnerabilities**: Review govulncheck output for CVE details and update paths

## Notes

- **Performance**: Quick scope provides fastest feedback
- **Pre-commit hook**: Consider running quick scope automatically
- **CI/CD**: Full scope should run in CI/CD pipeline
- **Security**: Run security scope weekly or before releases
- **Coverage**: Use coverage scope when improving test quality
