#!/usr/bin/env bash
# check-version-sync.sh - Validate version consistency across VERSION, CHANGELOG.md, and plugin.json
#
# Usage: ./scripts/check-version-sync.sh [--strict]
#
# Options:
#   --strict    Require explicit CHANGELOG entry for VERSION (not Unreleased)
#
# Exit codes:
#   0 - All versions in sync
#   1 - Version mismatch or validation error

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse arguments
STRICT_MODE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --strict)
            STRICT_MODE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--strict]"
            echo ""
            echo "Validates version consistency across:"
            echo "  - VERSION file (source of truth)"
            echo "  - .claude-plugin/plugin.json"
            echo "  - CHANGELOG.md"
            echo ""
            echo "Options:"
            echo "  --strict    Require explicit CHANGELOG entry (not Unreleased)"
            echo "  -h, --help  Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Get the repository root
if REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    :
else
    REPO_ROOT="$(pwd)"
fi

VERSION_FILE="$REPO_ROOT/VERSION"
PLUGIN_JSON="$REPO_ROOT/.claude-plugin/plugin.json"
CHANGELOG="$REPO_ROOT/CHANGELOG.md"

ERRORS=0

echo "=== Version Sync Validation ==="
echo ""

# 1. Check VERSION file exists and is valid
echo "Checking VERSION file..."
if [[ ! -f "$VERSION_FILE" ]]; then
    echo -e "  ${RED}ERROR: VERSION file not found at $VERSION_FILE${NC}"
    exit 1
fi

VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')

if [[ -z "$VERSION" ]]; then
    echo -e "  ${RED}ERROR: VERSION file is empty${NC}"
    exit 1
fi

# Validate semantic versioning format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$'; then
    echo -e "  ${RED}ERROR: Invalid version format: $VERSION${NC}"
    echo -e "  ${YELLOW}Expected: semantic version (e.g., 1.2.3, 1.2.3-rc.1)${NC}"
    exit 1
fi

echo -e "  ${GREEN}VERSION: $VERSION${NC}"

# 2. Check plugin.json version
echo ""
echo "Checking plugin.json..."
if [[ -f "$PLUGIN_JSON" ]]; then
    if command -v jq &> /dev/null; then
        PLUGIN_VERSION=$(jq -r '.version' "$PLUGIN_JSON" 2>/dev/null || echo "")
    else
        # Fallback to grep/sed
        PLUGIN_VERSION=$(grep -o '"version": "[^"]*"' "$PLUGIN_JSON" | cut -d'"' -f4)
    fi

    if [[ "$PLUGIN_VERSION" == "$VERSION" ]]; then
        echo -e "  ${GREEN}plugin.json version: $PLUGIN_VERSION (matches)${NC}"
    else
        echo -e "  ${RED}plugin.json version: $PLUGIN_VERSION (MISMATCH!)${NC}"
        echo -e "  ${YELLOW}  Expected: $VERSION${NC}"
        echo -e "  ${YELLOW}  Run: ./scripts/sync-version.sh${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "  ${YELLOW}plugin.json not found (skipping)${NC}"
fi

# 3. Check CHANGELOG.md
echo ""
echo "Checking CHANGELOG.md..."
if [[ -f "$CHANGELOG" ]]; then
    # Check for exact version entry (## [X.Y.Z])
    if grep -qE "^## \[$VERSION\]" "$CHANGELOG"; then
        echo -e "  ${GREEN}CHANGELOG has entry for [$VERSION]${NC}"
    elif grep -qE "^## \[Unreleased\]" "$CHANGELOG"; then
        if [[ "$STRICT_MODE" == "true" ]]; then
            echo -e "  ${RED}CHANGELOG has [Unreleased] but no [$VERSION] entry${NC}"
            echo -e "  ${YELLOW}  Strict mode requires explicit version entry${NC}"
            ERRORS=$((ERRORS + 1))
        else
            echo -e "  ${YELLOW}CHANGELOG has [Unreleased] (acceptable for pre-release)${NC}"
            echo -e "  ${YELLOW}  Use --strict to require explicit [$VERSION] entry${NC}"
        fi
    else
        echo -e "  ${RED}CHANGELOG missing entry for [$VERSION] or [Unreleased]${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "  ${YELLOW}CHANGELOG.md not found (skipping)${NC}"
fi

# 4. Summary
echo ""
echo "=== Summary ==="
if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}All version checks passed!${NC}"
    echo ""
    echo "VERSION file is the single source of truth."
    echo "All dependent files are in sync."
    exit 0
else
    echo -e "${RED}Found $ERRORS version sync error(s)${NC}"
    echo ""
    echo "To fix:"
    echo "  1. Ensure VERSION file contains the correct version"
    echo "  2. Run: ./scripts/sync-version.sh"
    echo "  3. Update CHANGELOG.md with version entry"
    exit 1
fi
