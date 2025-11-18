#!/bin/bash
# Configure contextd MCP server in ~/.claude.json

set -e

CLAUDE_CONFIG="$HOME/.claude.json"
PROJECT_PATH="/home/dahendel/projects/contextd"
CONTEXTD_URL="http://localhost:9090/mcp"

echo "🔧 Configuring contextd MCP server..."

# Check if contextd is running
if ! curl -s http://localhost:9090/health > /dev/null 2>&1; then
    echo "❌ contextd is not running on port 9090"
    echo "   Start it with: ./start-contextd.sh"
    exit 1
fi

echo "✅ contextd is running"

# Check if ~/.claude.json exists
if [ ! -f "$CLAUDE_CONFIG" ]; then
    echo "❌ ~/.claude.json not found"
    echo "   Claude Code hasn't been run yet"
    exit 1
fi

echo "✅ Found ~/.claude.json"

# Use Python to safely update JSON
python3 << 'PYTHON_EOF'
import json
import sys
from pathlib import Path

claude_json = Path.home() / '.claude.json'

# Read config
with open(claude_json, 'r') as f:
    config = json.load(f)

project_path = '/home/dahendel/projects/contextd'

# Ensure project entry exists
if project_path not in config:
    config[project_path] = {
        "mcpServers": {},
        "enabledMcpjsonServers": [],
        "disabledMcpjsonServers": [],
        "hasTrustDialogAccepted": True
    }

# Ensure mcpServers key exists
if 'mcpServers' not in config[project_path]:
    config[project_path]['mcpServers'] = {}

# Add contextd MCP server
config[project_path]['mcpServers']['contextd'] = {
    'url': 'http://localhost:9090/mcp',
    'transport': {
        'type': 'http'
    }
}

# Write back
with open(claude_json, 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Added contextd MCP server to ~/.claude.json")
print(f"   Project: {project_path}")
print(f"   URL: http://localhost:9090/mcp")

PYTHON_EOF

echo ""
echo "✅ Configuration complete!"
echo ""
echo "Next steps:"
echo "  1. Restart Claude Code completely (quit and reopen)"
echo "  2. Open the contextd project in Claude Code"
echo "  3. Test with: 'list available mcp tools'"
echo ""
echo "The MCP server will authenticate automatically using your system username."
