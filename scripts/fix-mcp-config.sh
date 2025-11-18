#!/bin/bash
# Quick fix script for MCP connection issue
# Adds contextd to global ~/.claude.json since project .mcp.json may not be loaded

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "  MCP Configuration Fix Script"
echo "======================================"
echo ""

# Check if global config exists
if [[ ! -f ~/.claude.json ]]; then
    echo -e "${RED}Error: ~/.claude.json not found${NC}"
    echo "Please ensure Claude Code is installed and configured."
    exit 1
fi

echo -e "${YELLOW}This script will add contextd to your global ~/.claude.json${NC}"
echo ""
echo "Current contextd config in global file:"
CURRENT=$(jq -r '.mcpServers.contextd // "not found"' ~/.claude.json 2>/dev/null)
echo "$CURRENT"
echo ""

# Backup
BACKUP_FILE=~/.claude.json.backup-$(date +%Y%m%d-%H%M%S)
echo "Creating backup: $BACKUP_FILE"
cp ~/.claude.json "$BACKUP_FILE"
echo -e "${GREEN}✓ Backup created${NC}"
echo ""

# Add or update contextd config
echo "Adding contextd configuration..."
TMP_FILE=$(mktemp)

jq '.mcpServers.contextd = {
  "type": "http",
  "url": "http://localhost:9090/mcp"
}' ~/.claude.json > "$TMP_FILE"

if [[ $? -eq 0 ]]; then
    mv "$TMP_FILE" ~/.claude.json
    echo -e "${GREEN}✓ Configuration updated${NC}"
else
    echo -e "${RED}✗ Failed to update configuration${NC}"
    rm -f "$TMP_FILE"
    exit 1
fi

echo ""
echo "New contextd config:"
jq '.mcpServers.contextd' ~/.claude.json
echo ""

# Verify
echo "Verifying configuration..."
if jq -e '.mcpServers.contextd.type == "http"' ~/.claude.json > /dev/null; then
    echo -e "${GREEN}✓ Transport type: HTTP${NC}"
else
    echo -e "${RED}✗ Transport type incorrect${NC}"
fi

if jq -e '.mcpServers.contextd.url == "http://localhost:9090/mcp"' ~/.claude.json > /dev/null; then
    echo -e "${GREEN}✓ URL: http://localhost:9090/mcp${NC}"
else
    echo -e "${RED}✗ URL incorrect${NC}"
fi

echo ""
echo "======================================"
echo "  Next Steps"
echo "======================================"
echo ""
echo "1. Restart Claude Code completely"
echo "2. Test the /mcp command"
echo "3. If it works, you can remove .mcp.json from the project directory"
echo ""
echo "To restore previous config:"
echo "  cp $BACKUP_FILE ~/.claude.json"
echo ""
