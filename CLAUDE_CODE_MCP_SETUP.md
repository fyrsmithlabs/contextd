# Claude Code MCP Client Setup

**Date**: 2025-11-18
**Status**: READY FOR TESTING
**Server**: contextd running on port 9090

---

## Configuration

### Claude Desktop Config

**File**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "contextd": {
      "url": "http://localhost:9090/mcp",
      "transport": "http"
    }
  }
}
```

### Server Status

```bash
# Check if contextd is running
$ pgrep -f contextd
# Should show process ID

# Check server health
$ curl http://localhost:9090/health
{"status":"healthy"}

# Check server logs
$ tail -f /tmp/contextd-new.log
```

---

## Testing Connection

### Step 1: Restart Claude Desktop

If you're using Claude Desktop app:
1. Quit Claude Desktop completely
2. Restart Claude Desktop
3. Open a new conversation

If you're using Claude Code (CLI):
1. Configuration is automatically loaded
2. Start a new conversation

### Step 2: Verify MCP Connection

In a Claude conversation, check if contextd tools are available:

**Available Tools** (12 total):
- `checkpoint_save` - Save session checkpoints
- `checkpoint_search` - Search checkpoints semantically
- `checkpoint_list` - List recent checkpoints
- `remediation_save` - Save error solutions
- `remediation_search` - Search error remediations
- `skill_save` - Save reusable skills
- `skill_search` - Search skills
- `index_repository` - Index code repositories
- `status` - Get operation status
- `collection_create` - Create vector collections
- `collection_delete` - Delete vector collections
- `collection_list` - List collections

### Step 3: Test Tool Call

Try using a tool in your conversation:

```
User: "Use the checkpoint_save tool to save a test checkpoint"
```

Claude should be able to call the tool and return results.

---

## Protocol Details

### MCP Streamable HTTP Protocol

**Spec Version**: 2025-03-26
**Protocol Version**: 2024-11-05

**Connection Flow**:
1. Client sends `initialize` request to `/mcp`
2. Server returns session ID in `Mcp-Session-Id` header
3. Client includes session ID in subsequent requests
4. Server validates session before processing tools/list, tools/call

**Headers Required**:
- `Accept: application/json, text/event-stream`
- `Content-Type: application/json`
- `Mcp-Session-Id: <uuid>` (after initialize)

### Example Initialize Request

```bash
curl -X POST http://localhost:9090/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "claude-code",
        "version": "2.0.44"
      }
    }
  }'
```

**Response**:
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

**Headers**:
- `Mcp-Session-Id: <uuid>`
- `Mcp-Protocol-Version: 2024-11-05`

---

## Troubleshooting

### Connection Refused

**Problem**: Claude Code cannot connect to contextd

**Solutions**:
1. Check contextd is running: `pgrep -f contextd`
2. Verify port is correct: `netstat -tlnp | grep 9090` or `ss -tlnp | grep 9090`
3. Check server logs: `tail -f /tmp/contextd-new.log`
4. Restart contextd: `pkill -f contextd && SERVER_PORT=9090 ./contextd &`

### Session Validation Failed

**Problem**: "Valid session ID required" error

**Cause**: Session expired or not created

**Solution**:
- Restart Claude Desktop to force new initialize handshake
- Check server logs for session creation

### Tools Not Available

**Problem**: Claude Code doesn't show contextd tools

**Solutions**:
1. Verify config file syntax: `cat ~/.config/Claude/claude_desktop_config.json | jq`
2. Check config location is correct
3. Restart Claude Desktop completely
4. Check server logs for initialize requests

### Accept Header Validation Failed

**Problem**: "Not Acceptable: Client must accept both application/json and text/event-stream"

**Cause**: Client not sending correct Accept header

**Solution**:
- This should not happen with Claude Code client
- If it does, it's a client bug - file issue with Anthropic

---

## Monitoring Connection

### Watch Server Logs

```bash
tail -f /tmp/contextd-new.log | grep -E "initialize|tools/list|tools/call"
```

### Check Active Sessions

Currently sessions are in-memory only. To see session activity:

```bash
# Look for session creation in logs
tail -100 /tmp/contextd-new.log | grep "Mcp-Session-Id"
```

### Test Tools Manually

```bash
# Get session ID from initialize
SESSION_ID="<uuid-from-initialize>"

# List tools
curl -X POST http://localhost:9090/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}'

# Call status tool
curl -X POST http://localhost:9090/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":"2","method":"tools/call","params":{"name":"status","arguments":{}}}'
```

---

## Next Steps

1. **Restart Claude Desktop** to load new configuration
2. **Start new conversation** to test MCP connection
3. **Try using a tool** to verify end-to-end functionality
4. **Monitor server logs** during tool calls
5. **Report any issues** with connection or tool execution

---

## Success Criteria

✅ Claude Desktop loads contextd MCP server
✅ Initialize handshake succeeds
✅ Session ID is created and stored
✅ Tools/list returns all 12 tools
✅ Tools/call executes successfully
✅ Results are returned to Claude Code

---

## References

- **MCP Spec**: MCP Streamable HTTP spec 2025-03-26
- **Implementation**: `pkg/mcp/protocol.go`
- **Tests**: `pkg/mcp/protocol_test.go`
- **E2E Results**: `MCP_PROTOCOL_IMPLEMENTATION_STATUS.md`
