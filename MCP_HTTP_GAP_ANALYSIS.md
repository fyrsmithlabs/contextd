# MCP HTTP Transport Gap Analysis

## Executive Summary

**Root Cause**: contextd implements a **custom MCP-like API** instead of the **official MCP Streamable HTTP Transport protocol**. Claude Code expects the standard protocol but finds incompatible custom endpoints.

## Critical Findings

### 1. Missing Core Protocol Endpoint

**Expected** (MCP Specification):
- `POST /mcp` - Handle all JSON-RPC 2.0 requests (initialize, tool calls)
- `GET /mcp` - SSE streaming for async responses
- `DELETE /mcp` - Session termination

**Actual** (contextd):
```
POST /mcp/checkpoint/save
POST /mcp/checkpoint/search
POST /mcp/checkpoint/list
POST /mcp/remediation/save
POST /mcp/remediation/search
POST /mcp/skill/save
POST /mcp/skill/search
POST /mcp/collection/create
POST /mcp/collection/delete
POST /mcp/collection/list
POST /mcp/index/repository
POST /mcp/status
GET  /mcp/sse/:operation_id
GET  /mcp/tools/list
GET  /mcp/resources/list
POST /mcp/resources/read
```

**Problem**: Claude Code sends `POST /mcp` with initialize request → Gets 404 Not Found

### 2. Wrong Transport Pattern

#### MCP Streamable HTTP (Official Spec)

```typescript
// Single endpoint handles all methods via JSON-RPC 2.0
app.post('/mcp', async (req, res) => {
  const message = req.body; // JSON-RPC request

  if (message.method === 'initialize') {
    // Initialize session, generate session ID
    const sessionId = randomUUID();
    res.setHeader('mcp-session-id', sessionId);
    // Return initialize response
  }

  if (message.method === 'tools/call') {
    const { name, arguments } = message.params;
    // Route to tool handler based on name
    if (name === 'checkpoint_save') {
      // Call checkpoint save logic
    }
  }
});

// SSE streaming for async responses
app.get('/mcp', async (req, res) => {
  const sessionId = req.headers['mcp-session-id'];
  // Start SSE stream for session
});
```

#### contextd (Current Implementation)

```go
// Each tool has its own endpoint (REST-like)
mcp.POST("/checkpoint/save", s.handleCheckpointSave)
mcp.POST("/checkpoint/search", s.handleCheckpointSearch)
mcp.POST("/remediation/save", s.handleRemediationSave)
// ... etc
```

### 3. Protocol Compliance Issues

| Requirement | MCP Spec | contextd | Status |
|-------------|----------|----------|--------|
| Single `/mcp` endpoint | ✅ Required | ❌ Missing | **FAIL** |
| JSON-RPC 2.0 routing | ✅ Required | ❌ Custom endpoints | **FAIL** |
| `initialize` method | ✅ Required | ❌ Not implemented | **FAIL** |
| Session management | ✅ `mcp-session-id` header | ❌ Uses auth tokens | **FAIL** |
| SSE streaming | ✅ `GET /mcp` | ⚠️  `GET /mcp/sse/:id` | **PARTIAL** |
| Accept headers | ✅ Must check | ❌ Not validated | **FAIL** |
| Protocol version | ✅ `mcp-protocol-version` | ❌ Not sent | **FAIL** |

### 4. Evidence from Logs

```
Nov 17 10:05:59 contextd[66755]:
  method=GET uri=/mcp/sse user_agent=claude-code/2.0.29 status=404
```

Claude Code is trying to:
1. Connect to `/mcp/sse` (not `/mcp/sse/:operation_id`)
2. Using the `claude-code/2.0.29` user agent
3. Getting 404 because that exact route doesn't exist

### 5. Port Configuration Mismatch

**Additional Issue**:
- Config says port 9090: `~/.claude.json` → `http://localhost:9090/mcp`
- Server runs on port 8081: `systemd` logs show "port":8081
- Claude Code connects to 8081 (discovers via some mechanism?)

## MCP Streamable HTTP Specification Requirements

Based on analysis of the official TypeScript SDK (`/tmp/typescript-sdk/src/server/streamableHttp.ts`):

### Required Headers

**POST Requests**:
```
Content-Type: application/json
Accept: application/json, text/event-stream
```

**GET Requests** (SSE):
```
Accept: text/event-stream
Mcp-Session-Id: <session-id>  (after initialization)
```

**All Requests** (after init):
```
Mcp-Session-Id: <session-id>
Mcp-Protocol-Version: 2024-11-05 (or negotiated version)
```

### Required Response Headers

**Initialization Response**:
```
Mcp-Session-Id: <generated-uuid>
Mcp-Protocol-Version: <negotiated-version>
```

**SSE Responses**:
```
Content-Type: text/event-stream
Cache-Control: no-cache, no-transform
Connection: keep-alive
```

### Protocol Flow

1. **Client → Server**: `POST /mcp` with `initialize` method
   - No session ID yet
   - Server generates session ID
   - Server returns session ID in header

2. **Client → Server**: `GET /mcp` with session ID
   - Establishes SSE stream
   - Server sends notifications via SSE

3. **Client → Server**: `POST /mcp` with `tools/call` method
   - Includes session ID
   - Server executes tool
   - Returns result via POST response OR SSE stream

4. **Client → Server**: `DELETE /mcp` with session ID
   - Terminates session
   - Cleans up resources

### JSON-RPC 2.0 Message Format

**Initialize Request**:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "claude-code",
      "version": "2.0.29"
    }
  }
}
```

**Initialize Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {},
      "resources": {}
    },
    "serverInfo": {
      "name": "contextd",
      "version": "0.9.0-rc-1"
    }
  }
}
```

**Tool Call Request**:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/call",
  "params": {
    "name": "checkpoint_save",
    "arguments": {
      "content": "...",
      "project_path": "...",
      "metadata": {}
    }
  }
}
```

### Session Management

**Stateful Mode** (recommended):
- Server generates session ID on initialization
- Client includes `mcp-session-id` header on all subsequent requests
- Server validates session ID
- Invalid session → 404 Not Found
- Missing session (non-init) → 400 Bad Request

**Stateless Mode**:
- No session ID generated
- Each request is independent
- Simpler but no resumability

## Root Cause Analysis

contextd was designed as a **REST-like API** with tool-specific endpoints, not as an **MCP protocol server**. The implementation:

1. ✅ Uses JSON-RPC 2.0 for responses
2. ✅ Provides tool discovery (`/mcp/tools/list`)
3. ❌ Doesn't implement the `/mcp` protocol endpoint
4. ❌ Doesn't handle `initialize` lifecycle
5. ❌ Doesn't use session management headers
6. ❌ Doesn't validate Accept headers
7. ❌ SSE endpoint has wrong signature (`/sse/:id` vs `/mcp`)

## Recommended Fix

### Option 1: Full MCP Protocol Compliance (Recommended)

Implement a proper MCP Streamable HTTP transport:

```go
// pkg/mcp/protocol.go
func (s *Server) RegisterMCPProtocol() {
    // Main protocol endpoint
    s.echo.POST("/mcp", s.handleMCPRequest)
    s.echo.GET("/mcp", s.handleMCPStream)
    s.echo.DELETE("/mcp", s.handleMCPDelete)
}

func (s *Server) handleMCPRequest(c echo.Context) error {
    // Validate Accept header
    accept := c.Request().Header.Get("Accept")
    if !strings.Contains(accept, "application/json") ||
       !strings.Contains(accept, "text/event-stream") {
        return c.JSON(406, map[string]interface{}{
            "jsonrpc": "2.0",
            "error": map[string]interface{}{
                "code": -32000,
                "message": "Not Acceptable: Client must accept both application/json and text/event-stream",
            },
            "id": nil,
        })
    }

    // Parse JSON-RPC request
    var req JSONRPCRequest
    if err := c.Bind(&req); err != nil {
        return JSONRPCErrorWithContext(c, "", ParseError, err)
    }

    // Route based on method
    switch req.Method {
    case "initialize":
        return s.handleInitialize(c, req)
    case "tools/list":
        return s.handleToolsList(c)
    case "tools/call":
        return s.handleToolCall(c, req)
    case "resources/list":
        return s.handleResourcesList(c)
    case "resources/read":
        return s.handleResourceRead(c, req)
    default:
        return JSONRPCErrorWithContext(c, req.ID, MethodNotFound,
            fmt.Errorf("unknown method: %s", req.Method))
    }
}

func (s *Server) handleInitialize(c echo.Context, req JSONRPCRequest) error {
    // Generate session ID
    sessionID := uuid.New().String()
    s.sessions.Store(sessionID, &Session{
        ID: sessionID,
        CreatedAt: time.Now(),
    })

    // Set response headers
    c.Response().Header().Set("Mcp-Session-Id", sessionID)
    c.Response().Header().Set("Mcp-Protocol-Version", "2024-11-05")

    return JSONRPCSuccess(c, req.ID, map[string]interface{}{
        "protocolVersion": "2024-11-05",
        "capabilities": map[string]interface{}{
            "tools": map[string]interface{}{},
            "resources": map[string]interface{}{},
        },
        "serverInfo": map[string]interface{}{
            "name": "contextd",
            "version": "0.9.0-rc-1",
        },
    })
}

func (s *Server) handleToolCall(c echo.Context, req JSONRPCRequest) error {
    // Validate session
    sessionID := c.Request().Header.Get("Mcp-Session-Id")
    if sessionID == "" {
        return c.JSON(400, map[string]interface{}{
            "jsonrpc": "2.0",
            "error": map[string]interface{}{
                "code": -32000,
                "message": "Bad Request: No session ID provided",
            },
            "id": req.ID,
        })
    }

    // Parse params
    var params struct {
        Name      string                 `json:"name"`
        Arguments map[string]interface{} `json:"arguments"`
    }
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
    }

    // Route to tool handler
    switch params.Name {
    case "checkpoint_save":
        return s.handleCheckpointSaveTool(c, req.ID, params.Arguments)
    case "checkpoint_search":
        return s.handleCheckpointSearchTool(c, req.ID, params.Arguments)
    // ... etc
    default:
        return JSONRPCErrorWithContext(c, req.ID, InvalidParams,
            fmt.Errorf("unknown tool: %s", params.Name))
    }
}
```

### Option 2: Hybrid Approach (Keep Both)

Keep existing REST-like endpoints for backward compatibility, add MCP protocol endpoint:

```go
func (s *Server) RegisterRoutes() {
    // NEW: MCP protocol endpoints
    s.echo.POST("/mcp", s.handleMCPRequest)
    s.echo.GET("/mcp", s.handleMCPStream)
    s.echo.DELETE("/mcp", s.handleMCPDelete)

    // EXISTING: Tool-specific endpoints (backward compat)
    mcp := s.echo.Group("/mcp", auth.OwnerAuthMiddleware())
    mcp.POST("/checkpoint/save", s.handleCheckpointSave)
    // ... etc
}
```

## Configuration Fix

Update user config to point to correct port:

```bash
# Fix port mismatch
claude mcp remove contextd
claude mcp add -s user -t http contextd http://localhost:8081/mcp

# Or update SERVER_PORT env var
echo 'SERVER_PORT=9090' >> /etc/systemd/user/contextd.service.d/override.conf
systemctl --user daemon-reload
systemctl --user restart contextd
```

## Testing Plan

1. **Test initialize**:
   ```bash
   curl -X POST http://localhost:8081/mcp \
     -H "Content-Type: application/json" \
     -H "Accept: application/json, text/event-stream" \
     -d '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
   ```

2. **Test tool call**:
   ```bash
   curl -X POST http://localhost:8081/mcp \
     -H "Content-Type: application/json" \
     -H "Accept: application/json, text/event-stream" \
     -H "Mcp-Session-Id: <session-id-from-init>" \
     -d '{"jsonrpc":"2.0","id":"2","method":"tools/call","params":{"name":"checkpoint_save","arguments":{"content":"test","project_path":"/tmp"}}}'
   ```

3. **Test SSE stream**:
   ```bash
   curl -N -H "Accept: text/event-stream" \
     -H "Mcp-Session-Id: <session-id>" \
     http://localhost:8081/mcp
   ```

## References

- **MCP TypeScript SDK**: `/tmp/typescript-sdk/src/server/streamableHttp.ts`
- **Example Server**: `/tmp/servers/src/everything/streamableHttp.ts`
- **Blog Post**: https://blog.fka.dev/blog/2025-06-06-why-mcp-deprecated-sse-and-go-with-streamable-http/
- **MCP Specification**: https://spec.modelcontextprotocol.io/

## Next Steps

1. ✅ **Immediate**: Fix port configuration mismatch (8081 vs 9090)
2. 🔧 **Short-term**: Implement `/mcp` protocol endpoint (Option 1 or 2)
3. 🧪 **Testing**: Verify Claude Code can connect and call tools
4. 📝 **Documentation**: Update MCP setup guide with correct protocol

## Impact Assessment

**Breaking Changes**:
- None if using Option 2 (hybrid approach)
- Tool-specific endpoints remain backward compatible

**Work Estimate**:
- Option 1 (full rewrite): 8-16 hours
- Option 2 (hybrid): 4-8 hours
- Configuration fix: 5 minutes

**Risk Level**: Medium
- Protocol implementation is well-documented
- TypeScript SDK provides clear reference
- Can test incrementally

## Conclusion

contextd needs to implement the official MCP Streamable HTTP transport protocol, not a custom REST-like API. The core issue is the missing `/mcp` endpoint that handles JSON-RPC 2.0 method routing. Once implemented, Claude Code will be able to:

1. Initialize a session via `POST /mcp` with `initialize` method
2. Discover tools via `POST /mcp` with `tools/list` method
3. Call tools via `POST /mcp` with `tools/call` method
4. Stream async responses via `GET /mcp` SSE endpoint

The fix is well-scoped and has clear implementation guidance from the official SDK.
