# MCP Connection Test Strategy

**Date**: 2025-11-17
**Issue**: Claude Code shows "Failed to reconnect to contextd" after configuration change
**Fix Applied**: Changed `.mcp.json` from SSE to HTTP transport type

## Problem Analysis

### Known Working Components
- Contextd service: RUNNING (port 9090, systemd active)
- Qdrant: RUNNING (port 6333)
- Health endpoint: WORKING (`curl http://localhost:9090/health` succeeds)
- MCP discovery: WORKING (`curl http://localhost:9090/mcp/tools/list` returns 1 tool)
- Configuration file: UPDATED (`.mcp.json` now uses `"type": "http"`)

### Unknown/Unverified
- Whether Claude Code loads `.mcp.json` (project-level config)
- Whether MCP protocol handshake works (JSON-RPC 2.0 over HTTP)
- Whether authentication passes (owner-based auth middleware)
- Whether tool invocation works end-to-end

### Key Architectural Details

From code analysis:
1. **Authentication**: ALL `/mcp/*` endpoints require `OwnerAuthMiddleware()`
2. **Protocol**: JSON-RPC 2.0 over HTTP POST
3. **Endpoints**: 12 tools exposed at `/mcp/<tool>/<action>` paths
4. **Owner ID**: Derived from system username (SHA256 hash)
5. **Transport**: HTTP (not SSE for initial connection)

## Test Strategy

### Phase 1: Configuration Verification

#### Test 1.1: Verify Config File Exists and Valid
```bash
# Check file exists
test -f /home/dahendel/projects/contextd/.mcp.json && echo "Config exists" || echo "Config missing"

# Validate JSON syntax
jq . /home/dahendel/projects/contextd/.mcp.json

# Check config content
cat /home/dahendel/projects/contextd/.mcp.json
```

**Expected Output**:
```json
{
  "mcpServers": {
    "contextd": {
      "type": "http",
      "url": "http://localhost:9090/mcp"
    }
  }
}
```

#### Test 1.2: Check Claude Code Config Loading
```bash
# Claude Code may merge project .mcp.json with ~/.claude.json
# Check if there's a contextd entry in global config that might override
jq '.mcpServers.contextd // "not found"' ~/.claude.json 2>/dev/null
```

**Expected**: Should show "not found" or match project config

**Action if Different**: Claude Code may be using global config instead of project config. Need to verify which takes precedence.

### Phase 2: HTTP Transport Verification

#### Test 2.1: Manual MCP Discovery Request
```bash
# Test GET /mcp/tools/list (unauthenticated - should fail or return limited info)
curl -v http://localhost:9090/mcp/tools/list

# Expected: May fail with auth error OR return tool list (depending on auth middleware on GET)
```

**Critical Check**: Look for:
- HTTP 200 response
- JSON-RPC 2.0 response format
- Tool list in response body

#### Test 2.2: Manual Tool Invocation (Status Endpoint)
```bash
# Test POST /mcp/status with minimal JSON-RPC request
curl -v -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "test-001",
    "method": "status",
    "params": {}
  }'
```

**Expected Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "test-001",
  "result": {
    "status": "healthy",
    "service": "contextd",
    "version": "0.9.0-rc-1"
  }
}
```

**Critical Checks**:
- HTTP status code (200 for success, 401/403 for auth failure)
- JSON-RPC response format
- Error details if authentication fails

#### Test 2.3: Authentication Flow Test
```bash
# The middleware expects owner ID from authenticated context
# Test without auth header (should fail)
curl -v -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"test-002","method":"status","params":{}}' \
  2>&1 | grep -E "HTTP/|401|403|owner"

# Check what authentication is required
# From code: auth.OwnerAuthMiddleware() is applied to all /mcp/* routes
# Need to check pkg/auth/middleware.go to see what headers/tokens are expected
```

**Action**: Examine authentication middleware to understand required headers/tokens

### Phase 3: MCP Protocol Testing

#### Test 3.1: MCP Handshake Simulation
```bash
# Create a test script to simulate MCP client handshake
cat > /tmp/test_mcp_handshake.sh << 'EOF'
#!/bin/bash

# MCP Client typically:
# 1. Calls tools/list to discover available tools
# 2. Calls tools/call to invoke a specific tool

# Step 1: Discovery
echo "=== MCP Discovery ==="
curl -s -X GET http://localhost:9090/mcp/tools/list | jq .

# Step 2: Tool Invocation (status tool)
echo -e "\n=== MCP Tool Invocation ==="
curl -s -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "test-handshake",
    "method": "status",
    "params": {}
  }' | jq .
EOF

chmod +x /tmp/test_mcp_handshake.sh
/tmp/test_mcp_handshake.sh
```

**Expected**: Both requests should succeed with JSON-RPC responses

#### Test 3.2: Test All Tool Endpoints
```bash
# Test each tool endpoint to verify they're accessible
cat > /tmp/test_all_tools.sh << 'EOF'
#!/bin/bash

TOOLS=(
  "status"
  "tools/list"
  "resources/list"
)

for tool in "${TOOLS[@]}"; do
  echo "=== Testing: $tool ==="
  if [[ "$tool" == "tools/list" ]] || [[ "$tool" == "resources/list" ]]; then
    # GET endpoints
    curl -s -X GET "http://localhost:9090/mcp/$tool" | jq -r '.result // .error // "NO RESPONSE"'
  else
    # POST endpoints
    curl -s -X POST "http://localhost:9090/mcp/$tool" \
      -H "Content-Type: application/json" \
      -d "{\"jsonrpc\":\"2.0\",\"id\":\"test-$tool\",\"method\":\"$tool\",\"params\":{}}" | jq -r '.result // .error // "NO RESPONSE"'
  fi
  echo -e "\n"
done
EOF

chmod +x /tmp/test_all_tools.sh
/tmp/test_all_tools.sh
```

### Phase 4: Authentication Deep Dive

#### Test 4.1: Examine Authentication Middleware
```bash
# Read the auth middleware code to understand requirements
cat /home/dahendel/projects/contextd/pkg/auth/middleware.go

# Look for:
# - Required headers (Authorization, X-Owner-ID, etc.)
# - Token validation
# - How owner ID is derived
```

**Critical Questions**:
1. Does it require a Bearer token?
2. Does it require X-Owner-ID header?
3. How is the owner ID validated?
4. Does it integrate with system username?

#### Test 4.2: Test with Proper Auth Headers
Once we know what headers are required, test with them:
```bash
# Example (adjust based on actual requirements):
curl -v -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token-if-required>" \
  -H "X-Owner-ID: <owner-hash-if-required>" \
  -d '{"jsonrpc":"2.0","id":"test-auth","method":"status","params":{}}'
```

### Phase 5: Claude Code Integration Testing

#### Test 5.1: Enable MCP Debug Logging
If Claude Code supports debug mode:
```bash
# Check for Claude Code debug flags or environment variables
# (This depends on Claude Code's capabilities - may need to check documentation)

# Potential options:
# - Enable verbose logging
# - Enable MCP protocol tracing
# - Check Claude Code's developer console
```

#### Test 5.2: Test MCP Connection from Claude Code CLI (if available)
```bash
# Some MCP clients provide CLI tools for testing
# Check if Claude Code has a test command:

# Hypothetical command (adjust based on actual CLI):
# claude mcp test contextd
# claude mcp list
# claude mcp call contextd status
```

#### Test 5.3: Background Session Testing
Based on user's question about headless testing:
```bash
# Start Claude Code in background with logging
# (Exact command depends on Claude Code installation)

# Example approach:
# 1. Create a test script that Claude Code can execute
# 2. Run Claude Code in headless mode with MCP enabled
# 3. Capture logs to diagnose connection issues

# Placeholder (adjust based on actual Claude Code capabilities):
# claude --headless --log-level=debug --mcp-config=/home/dahendel/projects/contextd/.mcp.json
```

### Phase 6: Network and Process Verification

#### Test 6.1: Verify Port Accessibility
```bash
# Check if port 9090 is accessible
netstat -tlnp | grep 9090

# Test TCP connection
nc -zv localhost 9090

# Test HTTP connectivity
curl -I http://localhost:9090/health
```

#### Test 6.2: Check for Conflicting Services
```bash
# Check if anything else is using port 9090
sudo lsof -i :9090

# Check systemd service status
systemctl --user status contextd

# Check service logs for errors
journalctl --user -u contextd --since "5 minutes ago"
```

#### Test 6.3: Firewall and SELinux
```bash
# Check if firewall is blocking localhost connections
sudo iptables -L -n | grep 9090

# Check SELinux status (if enabled)
getenforce
```

## Diagnostic Commands Summary

### Quick Diagnostic Script
```bash
#!/bin/bash
# MCP Connection Diagnostics

echo "=== Configuration ==="
echo "Project .mcp.json:"
jq . /home/dahendel/projects/contextd/.mcp.json 2>/dev/null || echo "Not found or invalid JSON"

echo -e "\n=== Service Status ==="
systemctl --user is-active contextd
curl -s http://localhost:9090/health | jq .

echo -e "\n=== MCP Discovery ==="
curl -s http://localhost:9090/mcp/tools/list | jq '.result | length // "ERROR"'

echo -e "\n=== MCP Status Tool ==="
curl -s -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"diag","method":"status","params":{}}' | jq .

echo -e "\n=== Authentication Test ==="
curl -s -X POST http://localhost:9090/mcp/status \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"auth-test","method":"status","params":{}}' \
  2>&1 | grep -E "401|403|Unauthorized|owner" || echo "No auth errors detected"

echo -e "\n=== Port Status ==="
netstat -tlnp 2>/dev/null | grep 9090 || ss -tlnp | grep 9090
```

## Expected Outcomes by Test

| Test | Success Indicator | Failure Indicator | Next Action |
|------|-------------------|-------------------|-------------|
| Config Verification | Valid JSON, correct type/URL | Missing file, invalid JSON | Fix config file |
| HTTP Connectivity | HTTP 200, valid JSON response | Connection refused, timeout | Check service status |
| MCP Discovery | Tool list returned | Empty or error response | Check MCP server implementation |
| Tool Invocation | JSON-RPC success response | JSON-RPC error | Analyze error code/message |
| Authentication | 200 OK with result | 401/403 or auth error | Add required headers |
| Protocol Handshake | Both discovery and call work | Either step fails | Check MCP protocol compliance |

## Isolation Strategy

To identify the exact failure point, test in this order:

1. **Configuration Layer**: Is .mcp.json loaded?
2. **Network Layer**: Can we reach the HTTP endpoint?
3. **Protocol Layer**: Does JSON-RPC work?
4. **Authentication Layer**: Do requests pass auth middleware?
5. **Application Layer**: Do tools execute correctly?
6. **Integration Layer**: Does Claude Code connect?

## Next Steps Based on Findings

### If Configuration is the Issue
- Check if Claude Code loads project-level .mcp.json
- Verify global ~/.claude.json doesn't override
- Check config merge behavior

### If Authentication Fails
- Read pkg/auth/middleware.go to understand requirements
- Generate required tokens
- Add required headers to test requests
- Check owner ID derivation logic

### If Protocol Fails
- Compare our JSON-RPC format with MCP spec
- Check for protocol version mismatches
- Verify content-type headers

### If Integration Fails (Claude Code specific)
- Check Claude Code documentation for MCP setup
- Look for Claude Code debug logs
- Test with alternative MCP client (if available)
- Check Claude Code GitHub issues for similar problems

## Critical Files to Review

1. `/home/dahendel/projects/contextd/.mcp.json` - Project MCP config
2. `/home/dahendel/projects/contextd/pkg/auth/middleware.go` - Auth requirements
3. `/home/dahendel/projects/contextd/pkg/mcp/server.go` - MCP server implementation
4. `/home/dahendel/projects/contextd/pkg/mcp/types.go` - JSON-RPC types
5. `~/.claude.json` - Global Claude config (may override)

## Test Execution Order

1. Run Quick Diagnostic Script (baseline)
2. Execute Phase 1: Config Verification
3. Execute Phase 4.1: Read auth middleware code
4. Execute Phase 2: HTTP Transport Tests (with auth if required)
5. Execute Phase 3: MCP Protocol Tests
6. Execute Phase 6: Network/Process Verification
7. Execute Phase 5: Claude Code Integration (if previous phases pass)

## Success Criteria

The MCP connection is working when:
- [ ] `.mcp.json` loads correctly
- [ ] HTTP endpoints are accessible
- [ ] JSON-RPC requests return valid responses
- [ ] Authentication passes (if required)
- [ ] At least one tool (status) executes successfully
- [ ] Claude Code can discover and call tools

## Debug Logging Recommendations

Enable verbose logging for:
- Contextd service (journalctl)
- Echo HTTP server (if configurable)
- MCP protocol exchanges (request/response logging)
- Claude Code (if debug mode available)

## References

- MCP Specification: (check for official spec URL)
- JSON-RPC 2.0: https://www.jsonrpc.org/specification
- Contextd MCP Implementation: `/home/dahendel/projects/contextd/pkg/mcp/`
- Authentication: `/home/dahendel/projects/contextd/pkg/auth/`
