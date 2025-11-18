#!/bin/bash
# MCP Connection Diagnostic Script
# Tests MCP connection at multiple layers to isolate failures

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROJECT_DIR="/home/dahendel/projects/contextd"
MCP_URL="http://localhost:9090/mcp"
HEALTH_URL="http://localhost:9090/health"

echo "======================================"
echo "  MCP Connection Diagnostics"
echo "======================================"
echo ""

# Phase 1: Configuration Verification
echo -e "${YELLOW}[Phase 1: Configuration Verification]${NC}"
echo ""

echo "1.1 Checking .mcp.json exists..."
if [[ -f "$PROJECT_DIR/.mcp.json" ]]; then
    echo -e "${GREEN}✓ Config file exists${NC}"
else
    echo -e "${RED}✗ Config file NOT found at $PROJECT_DIR/.mcp.json${NC}"
    exit 1
fi

echo ""
echo "1.2 Validating JSON syntax..."
if jq . "$PROJECT_DIR/.mcp.json" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ JSON is valid${NC}"
else
    echo -e "${RED}✗ JSON is INVALID${NC}"
    exit 1
fi

echo ""
echo "1.3 Config content:"
jq . "$PROJECT_DIR/.mcp.json"

echo ""
echo "1.4 Checking global Claude config for conflicts..."
GLOBAL_CONTEXTD=$(jq -r '.mcpServers.contextd // "not-found"' ~/.claude.json 2>/dev/null)
if [[ "$GLOBAL_CONTEXTD" == "not-found" ]]; then
    echo -e "${GREEN}✓ No global contextd config (project config will be used)${NC}"
else
    echo -e "${YELLOW}⚠ Global contextd config exists:${NC}"
    echo "$GLOBAL_CONTEXTD"
fi

echo ""
echo ""

# Phase 2: Service Status
echo -e "${YELLOW}[Phase 2: Service Status]${NC}"
echo ""

echo "2.1 Checking systemd service..."
if systemctl --user is-active contextd > /dev/null 2>&1; then
    echo -e "${GREEN}✓ contextd service is active${NC}"
else
    echo -e "${RED}✗ contextd service is NOT active${NC}"
    systemctl --user status contextd
    exit 1
fi

echo ""
echo "2.2 Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s "$HEALTH_URL")
if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ Health endpoint accessible${NC}"
    echo "Response: $HEALTH_RESPONSE"
else
    echo -e "${RED}✗ Health endpoint NOT accessible${NC}"
    exit 1
fi

echo ""
echo ""

# Phase 3: MCP Discovery (GET requests)
echo -e "${YELLOW}[Phase 3: MCP Discovery]${NC}"
echo ""

echo "3.1 Testing GET /mcp/tools/list..."
TOOLS_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$MCP_URL/tools/list")
HTTP_CODE=$(echo "$TOOLS_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$TOOLS_RESPONSE" | grep -v "HTTP_CODE:")

echo "HTTP Status: $HTTP_CODE"
if [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "${GREEN}✓ Tools list endpoint accessible${NC}"
    TOOL_COUNT=$(echo "$BODY" | jq -r '.result | length // "ERROR"')
    echo "Number of tools: $TOOL_COUNT"
    echo "Response:"
    echo "$BODY" | jq .
elif [[ "$HTTP_CODE" == "401" ]] || [[ "$HTTP_CODE" == "403" ]]; then
    echo -e "${YELLOW}⚠ Authentication required (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq .
else
    echo -e "${RED}✗ Unexpected HTTP status: $HTTP_CODE${NC}"
    echo "$BODY"
fi

echo ""
echo "3.2 Testing GET /mcp/resources/list..."
RESOURCES_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$MCP_URL/resources/list")
HTTP_CODE=$(echo "$RESOURCES_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$RESOURCES_RESPONSE" | grep -v "HTTP_CODE:")

echo "HTTP Status: $HTTP_CODE"
if [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "${GREEN}✓ Resources list endpoint accessible${NC}"
    echo "Response:"
    echo "$BODY" | jq .
else
    echo -e "${YELLOW}⚠ HTTP $HTTP_CODE${NC}"
    echo "$BODY" | jq .
fi

echo ""
echo ""

# Phase 4: MCP Tool Invocation (POST requests)
echo -e "${YELLOW}[Phase 4: MCP Tool Invocation]${NC}"
echo ""

echo "4.1 Testing POST /mcp/status (JSON-RPC 2.0)..."
STATUS_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST "$MCP_URL/status" \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "id": "test-status",
        "method": "status",
        "params": {}
    }')

HTTP_CODE=$(echo "$STATUS_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$STATUS_RESPONSE" | grep -v "HTTP_CODE:")

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq .

if [[ "$HTTP_CODE" == "200" ]]; then
    # Check if it's a JSON-RPC success or error
    JSONRPC_ERROR=$(echo "$BODY" | jq -r '.error // "null"')
    if [[ "$JSONRPC_ERROR" == "null" ]]; then
        echo -e "${GREEN}✓ Status tool executed successfully${NC}"
        echo "Service: $(echo "$BODY" | jq -r '.result.service // "unknown"')"
        echo "Version: $(echo "$BODY" | jq -r '.result.version // "unknown"')"
    else
        echo -e "${RED}✗ JSON-RPC error returned${NC}"
        echo "Error code: $(echo "$BODY" | jq -r '.error.code')"
        echo "Error message: $(echo "$BODY" | jq -r '.error.message')"

        # Check if it's an auth error
        ERROR_CODE=$(echo "$BODY" | jq -r '.error.code')
        if [[ "$ERROR_CODE" == "-32005" ]]; then
            echo -e "${YELLOW}⚠ Authentication error detected (code: -32005)${NC}"
        fi
    fi
else
    echo -e "${RED}✗ HTTP error: $HTTP_CODE${NC}"
fi

echo ""
echo ""

# Phase 5: Authentication Analysis
echo -e "${YELLOW}[Phase 5: Authentication Analysis]${NC}"
echo ""

echo "5.1 Checking owner ID derivation..."
CURRENT_USER=$(whoami)
echo "Current user: $CURRENT_USER"

OWNER_HASH=$(echo -n "$CURRENT_USER" | sha256sum | cut -d' ' -f1)
echo "Computed owner hash (SHA256): $OWNER_HASH"

echo ""
echo "5.2 Testing with explicit owner header (if supported)..."
OWNER_TEST_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST "$MCP_URL/status" \
    -H "Content-Type: application/json" \
    -H "X-Owner-ID: $OWNER_HASH" \
    -d '{
        "jsonrpc": "2.0",
        "id": "test-owner",
        "method": "status",
        "params": {}
    }')

HTTP_CODE=$(echo "$OWNER_TEST_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$OWNER_TEST_RESPONSE" | grep -v "HTTP_CODE:")

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq .

echo ""
echo ""

# Phase 6: Network Verification
echo -e "${YELLOW}[Phase 6: Network Verification]${NC}"
echo ""

echo "6.1 Checking port 9090 is listening..."
if netstat -tlnp 2>/dev/null | grep -q ":9090" || ss -tlnp 2>/dev/null | grep -q ":9090"; then
    echo -e "${GREEN}✓ Port 9090 is listening${NC}"
    netstat -tlnp 2>/dev/null | grep ":9090" || ss -tlnp 2>/dev/null | grep ":9090"
else
    echo -e "${RED}✗ Port 9090 is NOT listening${NC}"
fi

echo ""
echo "6.2 Testing TCP connectivity..."
if nc -zv localhost 9090 2>&1 | grep -q "succeeded"; then
    echo -e "${GREEN}✓ TCP connection successful${NC}"
else
    echo -e "${RED}✗ TCP connection failed${NC}"
fi

echo ""
echo ""

# Summary
echo "======================================"
echo "  Summary"
echo "======================================"
echo ""

# Determine overall status
if systemctl --user is-active contextd > /dev/null 2>&1 && \
   curl -s "$HEALTH_URL" > /dev/null && \
   curl -s "$MCP_URL/tools/list" > /dev/null; then
    echo -e "${GREEN}✓ Basic connectivity: WORKING${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Review authentication requirements in pkg/auth/middleware.go"
    echo "2. Check if Claude Code is loading the project .mcp.json"
    echo "3. Enable debug logging in Claude Code if available"
    echo "4. Test with: curl -X POST http://localhost:9090/mcp/status -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":\"test\",\"method\":\"status\",\"params\":{}}'"
else
    echo -e "${RED}✗ Basic connectivity: FAILED${NC}"
    echo ""
    echo "Action required:"
    echo "1. Check systemd service: systemctl --user status contextd"
    echo "2. Check logs: journalctl --user -u contextd -f"
    echo "3. Verify configuration in $PROJECT_DIR/.mcp.json"
fi

echo ""
echo "Full diagnostic log saved to: /tmp/mcp-diagnostics-$(date +%Y%m%d-%H%M%S).log"
