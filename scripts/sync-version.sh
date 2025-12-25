#!/usr/bin/env bash
# sync-version.sh - Sync version from VERSION file to all version-dependent files
#
# Usage: ./scripts/sync-version.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the repository root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$REPO_ROOT/VERSION"

# Check if VERSION file exists
if [[ ! -f "$VERSION_FILE" ]]; then
    echo -e "${RED}ERROR: VERSION file not found at $VERSION_FILE${NC}"
    exit 1
fi

# Read version from VERSION file
VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')

if [[ -z "$VERSION" ]]; then
    echo -e "${RED}ERROR: VERSION file is empty${NC}"
    exit 1
fi

echo -e "${GREEN}Syncing version: $VERSION${NC}"

# Files to update
FILES_UPDATED=0

# 1. Update plugin.json
PLUGIN_JSON="$REPO_ROOT/.claude-plugin/plugin.json"
if [[ -f "$PLUGIN_JSON" ]]; then
    echo "  Updating $PLUGIN_JSON..."
    # Use jq if available, otherwise sed
    if command -v jq &> /dev/null; then
        TMP_FILE=$(mktemp)
        jq --arg version "$VERSION" '.version = $version' "$PLUGIN_JSON" > "$TMP_FILE"
        mv "$TMP_FILE" "$PLUGIN_JSON"
    else
        # Fallback to sed
        sed -i.bak "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$PLUGIN_JSON"
        rm -f "$PLUGIN_JSON.bak"
    fi
    FILES_UPDATED=$((FILES_UPDATED + 1))
    echo -e "${GREEN}  ✓ Updated plugin.json${NC}"
else
    echo -e "${YELLOW}  ! plugin.json not found, skipping${NC}"
fi

# 2. Check git tags (informational only)
echo ""
echo "Git tag status:"
LATEST_TAG=$(git tag -l "v$VERSION" 2>/dev/null || echo "")
if [[ -n "$LATEST_TAG" ]]; then
    echo -e "${GREEN}  ✓ Git tag v$VERSION exists${NC}"
else
    echo -e "${YELLOW}  ! Git tag v$VERSION does not exist yet${NC}"
    echo "    Run: git tag -a v$VERSION -m \"Release v$VERSION\""
fi

# Summary
echo ""
echo -e "${GREEN}Version sync complete!${NC}"
echo "  Version: $VERSION"
echo "  Files updated: $FILES_UPDATED"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Commit: git add VERSION .claude-plugin/plugin.json && git commit -m \"chore: bump version to $VERSION\""
if [[ -z "$LATEST_TAG" ]]; then
    echo "  3. Tag release: git tag -a v$VERSION -m \"Release v$VERSION\""
    echo "  4. Push: git push && git push --tags"
fi
