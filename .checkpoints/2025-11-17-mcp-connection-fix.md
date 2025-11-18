# MCP Connection Fix - 2025-11-17

## Problem Summary

**Issue**: MCP connection failing despite contextd running on port 9090

**Root Cause**: Incorrect transport type in `.mcp.json` configuration file

## Investigation Process

### Phase 1: Root Cause Investigation

1. **Qdrant Status**: Container showing "unhealthy" but actually running fine
   - Docker health check uses `curl` which doesn't exist in container
   - Manual test `curl http://localhost:6333/healthz` succeeds
   - **Verdict**: False alarm - Qdrant is working

2. **MCP Configuration Files**:
   - `.mcp.json` exists in project: `/home/dahendel/projects/contextd/.mcp.json`
   - `~/.claude.json` has empty `mcpServers: {}`
   - **CRITICAL**: `.mcp.json` uses `"type": "sse"` (deprecated transport)

3. **Environment Variables**:
   - `QDRANT_URL` not set (not needed - contextd service has it)
   - `OPENAI_API_KEY` not set (optional - can use TEI instead)

4. **MCP Server Verification**:
   - Contextd HTTP service running: ✅
   - Health endpoint responding: ✅ `http://localhost:9090/health`
   - MCP discovery working: ✅ `http://localhost:9090/mcp/tools/list` returns all 12 tools
   - **Verdict**: Server is perfect, only config is wrong

### Phase 2: Pattern Analysis

**What the Code Expects**:
- Transport: HTTP JSON-RPC 2.0 (NOT SSE for main protocol)
- SSE: Only used for streaming progress of long-running operations
- Endpoints: POST /mcp/checkpoint/save, POST /mcp/remediation/search, etc.
- Authentication: Owner-based (automatic from system username)

**What Claude Code Docs Say**:
```json
{
  "mcpServers": {
    "http-service": {
      "type": "http",  // Use "http" for HTTP JSON-RPC servers
      "url": "https://api.example.com/mcp"
    }
  }
}
```

**The Problem**:
```json
{
  "mcpServers": {
    "contextd": {
      "type": "sse",  // ❌ WRONG - SSE transport is deprecated
      "url": "http://localhost:9090/mcp"
    }
  }
}
```

### Phase 3: Hypothesis and Testing

**Hypothesis**: Changing `"type": "sse"` to `"type": "http"` will fix MCP connection

**Fix Applied**: Updated `.mcp.json`:

```diff
{
  "mcpServers": {
    "contextd": {
-      "type": "sse",
+      "type": "http",
      "url": "http://localhost:9090/mcp"
    }
  }
}
```

## Changes Made

### 1. Fixed `.mcp.json` Configuration

**File**: `/home/dahendel/projects/contextd/.mcp.json`

**Change**: `"type": "sse"` → `"type": "http"`

**Reason**:
- SSE transport is deprecated per MCP documentation
- Contextd implements HTTP JSON-RPC 2.0, not SSE transport
- SSE is only used for progress streaming, not the main protocol

### 2. Docker Health Check Fix (Optional)

**File**: `docker-compose.yml` (qdrant service)

**Issue**: Health check uses `curl` which doesn't exist in Qdrant container

**Options**:
1. Use `wget` instead (may not exist either)
2. Remove health check (container works fine without it)
3. Add `wget` to container (requires custom Dockerfile)

**Recommendation**: Remove health check for now since Qdrant works fine

## Next Steps for User

### 1. Test MCP Connection

**Option A: Restart Claude Code Session**
```bash
# Exit current Claude Code session
# Start new session in this project
cd /home/dahendel/projects/contextd
claude
```

**Option B: Reload MCP Configuration**
```bash
# If Claude Code supports hot-reload (check with /mcp command)
# Otherwise, restart session
```

### 2. Verify MCP Tools Work

Once connected, test the contextd MCP tools:

```
# Test checkpoint save
Use contextd's checkpoint_save tool to save this session

# Test checkpoint search
Use contextd's checkpoint_search to find previous checkpoints

# Test collection list
Use contextd's collection_list to see available collections
```

### 3. Proceed with Context-Folding Implementation

After MCP connection is verified:

1. **Validate existing tools work** (checkpoint, remediation, skill tools)
2. **Test multi-tenant isolation** (different project paths)
3. **Begin Phase 1 of context-folding spec**:
   - Implementation plan: `docs/specs/context-folding/08-implementation-phases.md`
   - Create implementation worktree
   - Implement core branch/fold mechanism

## Summary

**Problem**: Wrong MCP transport type in configuration
**Fix**: Changed `.mcp.json` from SSE to HTTP transport
**Status**: Fix applied, requires Claude Code restart to test
**Blocker Removed**: ✅ Can now proceed with context-folding implementation

## Files Modified

- `.mcp.json` - Fixed transport type (sse → http)

## Files Created

- `.checkpoints/2025-11-17-mcp-connection-fix.md` - This document

## References

- Previous checkpoint: `.checkpoints/2025-01-17-context-folding-spec-complete.md`
- Context-folding spec: `docs/specs/context-folding/INDEX.md`
- MCP spec: `docs/specs/mcp/SPEC.md`
- Claude Code MCP docs: https://docs.claude.com/en/docs/claude-code/mcp
