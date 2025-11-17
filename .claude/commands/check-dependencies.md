# Check Dependencies

**Command**: `/check-dependencies [--update] [--security] [--outdated]`

**Description**: Vulnerability scanning and dependency management for Go projects.

**Usage**:
```
/check-dependencies                  # Check vulnerabilities (default)
/check-dependencies --security       # Deep security scan with CVE links
/check-dependencies --outdated       # Show outdated packages
/check-dependencies --update         # Update dependencies safely
```

## Purpose

Critical security feature that:
- Scans for known vulnerabilities using govulncheck
- Identifies outdated dependencies
- Safely updates dependencies with automated testing
- Links to CVE/GHSA vulnerability databases
- Should run in CI/CD pipeline

## Agent Workflow

When this command is invoked, execute the dependency check script:

```bash
# Run the dependency check script
.scripts/check-dependencies.sh [flags]
```

## Check Modes

### Default (Vulnerability Check)
**Purpose**: Quick vulnerability scan
**Actions**:
- Run `govulncheck ./...`
- List all dependencies with `go list -m all`
- Report vulnerabilities with severity
- **Time**: ~30-60 seconds

**When to use**: Daily checks, pre-release verification

### Security Mode (`--security`)
**Purpose**: Deep security analysis
**Actions**:
- Run govulncheck with detailed output
- Check for indirect vulnerabilities
- Link to CVE/GHSA databases
- Report affected symbols and call stacks
- Suggest update paths
- **Time**: ~1-2 minutes

**When to use**: Security audits, compliance reviews, investigating alerts

### Outdated Mode (`--outdated`)
**Purpose**: Identify outdated dependencies
**Actions**:
- List all dependencies
- Check for available updates
- Show current vs latest versions
- Categorize by major/minor/patch
- **Time**: ~30 seconds

**When to use**: Weekly maintenance, before planning updates

### Update Mode (`--update`)
**Purpose**: Safely update dependencies
**Actions**:
- Check current vulnerabilities
- Update dependencies to latest compatible versions
- Run full test suite after updates
- Create git commit if tests pass
- Rollback if tests fail
- **Time**: ~3-5 minutes

**When to use**: Addressing vulnerabilities, regular maintenance

## Implementation Script

Location: `.scripts/check-dependencies.sh`

```bash
#!/bin/bash
# Script: check-dependencies.sh
# Purpose: Vulnerability scanning and dependency management
# Usage: ./check-dependencies.sh [--security|--outdated|--update]

set -e

MODE="default"

# Parse flags
while [[ $# -gt 0 ]]; do
    case $1 in
        --security)
            MODE="security"
            shift
            ;;
        --outdated)
            MODE="outdated"
            shift
            ;;
        --update)
            MODE="update"
            shift
            ;;
        *)
            echo "❌ Unknown flag: $1"
            exit 1
            ;;
    esac
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔐 Dependency Security Check - Mode: $MODE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Ensure govulncheck is installed
if ! command -v govulncheck &> /dev/null; then
    echo "Installing govulncheck..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
fi

# Default mode: Quick vulnerability check
if [[ "$MODE" == "default" ]]; then
    echo "🔍 Scanning for vulnerabilities..."
    if govulncheck ./...; then
        echo "✅ No known vulnerabilities found"
    else
        echo ""
        echo "⚠️  Vulnerabilities detected!"
        echo "Run with --security flag for detailed analysis"
        echo "Run with --update flag to attempt automatic fixes"
        exit 1
    fi

    echo ""
    echo "📦 Current dependencies:"
    go list -m all

    echo ""
    echo "✅ Dependency check complete"
    exit 0
fi

# Security mode: Deep analysis
if [[ "$MODE" == "security" ]]; then
    echo "🔍 Running deep security scan..."
    echo ""

    # Run with verbose output
    if govulncheck -show verbose ./...; then
        echo ""
        echo "✅ No vulnerabilities found"
    else
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "⚠️  VULNERABILITIES DETECTED"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "Review the output above for:"
        echo "  - CVE/GHSA identifiers"
        echo "  - Affected functions and call stacks"
        echo "  - Recommended fixes and versions"
        echo ""
        echo "To fix vulnerabilities, run:"
        echo "  /check-dependencies --update"
        exit 1
    fi

    echo ""
    echo "📊 Dependency analysis:"
    go list -m all
    echo ""
    echo "Total dependencies: $(go list -m all | wc -l)"

    echo ""
    echo "✅ Security scan complete"
    exit 0
fi

# Outdated mode: Check for updates
if [[ "$MODE" == "outdated" ]]; then
    echo "📊 Checking for outdated dependencies..."
    echo ""

    echo "Current dependencies:"
    go list -m all

    echo ""
    echo "Checking for available updates..."

    # Use go list to check for updates
    go list -u -m all 2>/dev/null | grep -v "^go " | while read -r line; do
        if echo "$line" | grep -q '\['; then
            echo "📦 Update available: $line"
        fi
    done || echo "ℹ️  All dependencies are up to date"

    echo ""
    echo "To update dependencies, run:"
    echo "  /check-dependencies --update"

    echo ""
    echo "✅ Outdated check complete"
    exit 0
fi

# Update mode: Safely update dependencies
if [[ "$MODE" == "update" ]]; then
    echo "🔄 Updating dependencies..."
    echo ""

    # Check for vulnerabilities first
    echo "1. Checking current vulnerabilities..."
    if govulncheck ./...; then
        echo "✅ No vulnerabilities in current dependencies"
    else
        echo "⚠️  Found vulnerabilities - will attempt to fix"
    fi

    echo ""
    echo "2. Updating dependencies..."

    # Update go.mod and go.sum
    go get -u ./...

    # Tidy up
    go mod tidy

    echo ""
    echo "3. Running tests..."

    # Run full test suite
    if go test ./...; then
        echo "✅ All tests pass with updated dependencies"
    else
        echo "❌ Tests failed after update - rolling back"
        git checkout go.mod go.sum
        go mod download
        echo "Rolled back to previous dependency versions"
        exit 1
    fi

    echo ""
    echo "4. Checking for race conditions..."
    if go test -race ./...; then
        echo "✅ No race conditions detected"
    else
        echo "❌ Race conditions detected - rolling back"
        git checkout go.mod go.sum
        go mod download
        exit 1
    fi

    echo ""
    echo "5. Final vulnerability check..."
    if govulncheck ./...; then
        echo "✅ No vulnerabilities after update"
    else
        echo "⚠️  Some vulnerabilities remain (may require major version updates)"
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Dependencies Updated Successfully"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Updated dependencies:"
    git diff go.mod

    echo ""
    echo "Next steps:"
    echo "  1. Review changes: git diff go.mod go.sum"
    echo "  2. Commit changes: git add go.mod go.sum && git commit -m 'deps: update dependencies'"
    echo "  3. Push changes: git push"

    exit 0
fi
```

## Integration with CI/CD

Add to GitHub Actions workflow:

```yaml
- name: Check Dependencies
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    .scripts/check-dependencies.sh --security
```

## Vulnerability Database

Govulncheck uses the Go vulnerability database:
- **Database**: https://vuln.go.dev
- **Format**: CVE and GHSA identifiers
- **Coverage**: Go standard library and known packages
- **Updates**: Continuously updated by Go security team

## Example Workflows

### Daily Security Check
```bash
# Quick vulnerability scan
/check-dependencies

# If vulnerabilities found:
/check-dependencies --security  # Detailed analysis
/check-dependencies --update     # Attempt fix
```

### Weekly Maintenance
```bash
# Check for outdated packages
/check-dependencies --outdated

# Review and plan updates
# Update dependencies
/check-dependencies --update
```

### Before Release
```bash
# Comprehensive security audit
/check-dependencies --security

# Update if needed
/check-dependencies --update

# Verify fix
/check-dependencies --security
```

## Error Handling

### Vulnerabilities Found
1. Review detailed output for CVE/GHSA links
2. Check if fix is available via update
3. If no fix available, consider:
   - Finding alternative package
   - Waiting for upstream fix
   - Implementing workaround

### Update Failures
1. Check test output for failures
2. Review breaking changes in updated packages
3. Fix compatibility issues
4. Re-run update

### No Fix Available
1. Document vulnerability in security advisory
2. Implement mitigation if possible
3. Monitor for upstream fixes
4. Consider alternative packages

## Success Criteria

- ✅ All modes execute without errors
- ✅ Vulnerabilities detected and reported with CVE links
- ✅ Updates applied safely with test verification
- ✅ Rollback works if tests fail
- ✅ Integration with CI/CD pipeline

## Notes

- **Security Critical**: Run before every release
- **Automation**: Add to CI/CD pipeline
- **Weekly Check**: Schedule regular dependency reviews
- **Update Strategy**: Test thoroughly before merging
- **Documentation**: Keep security advisories updated
