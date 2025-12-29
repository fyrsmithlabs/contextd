#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Read VERSION file
VERSION_FILE="$PROJECT_ROOT/VERSION"
if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}ERROR: VERSION file not found at $VERSION_FILE${NC}"
    exit 1
fi

VERSION=$(cat "$VERSION_FILE" | tr -d '\n' | tr -d ' ')

# Validate version format (semantic versioning)
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$'; then
    echo -e "${RED}ERROR: Invalid version format: $VERSION${NC}"
    echo -e "${YELLOW}Expected: semantic version (e.g., 1.2.3, 1.2.3-rc.1, 1.2.3+build.123)${NC}"
    exit 1
fi

echo -e "${GREEN}Syncing version $VERSION across project files...${NC}"

# Update .claude-plugin/plugin.json
PLUGIN_JSON="$PROJECT_ROOT/.claude-plugin/plugin.json"
if [ -f "$PLUGIN_JSON" ]; then
    # Use sed to update the version field in plugin.json
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS requires -i '' for in-place edit
        sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$PLUGIN_JSON"
    else
        # Linux
        sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$PLUGIN_JSON"
    fi
    echo -e "${GREEN}✓ Updated .claude-plugin/plugin.json${NC}"
else
    echo -e "${YELLOW}⚠ .claude-plugin/plugin.json not found, skipping${NC}"
fi

echo -e "${GREEN}Version sync complete: $VERSION${NC}"
echo -e "${YELLOW}Don't forget to commit these changes:${NC}"
echo -e "  git add VERSION .claude-plugin/plugin.json"
echo -e "  git commit -m \"chore: bump version to $VERSION\""
