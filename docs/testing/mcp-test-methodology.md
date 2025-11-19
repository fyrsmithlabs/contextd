# MCP Test Methodology

**Parent**: [MCP_E2E_TEST_RESULTS.md](../../MCP_E2E_TEST_RESULTS.md)

## Test Script

**Location**: `/tmp/test_all_mcp_tools.sh`

**Protocol**: MCP Streamable HTTP (2024-11-05)

**Transport**: HTTP POST to `http://localhost:9090/mcp`

## Session Initialization

**Step 1**: Initialize MCP session

```bash
curl -i -X POST http://localhost:9090/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "test",
        "version": "1.0"
      }
    }
  }'
```

**Step 2**: Extract session ID from response header

```bash
SESSION_ID=$(echo "$RESPONSE" | grep -i "^Mcp-Session-Id:" | cut -d' ' -f2 | tr -d '\r\n')
```

**Result**: Session ID in `Mcp-Session-Id` header (e.g., `sess_abc123`)

## Tool Testing Pattern

**Request Format**:
```bash
curl -X POST http://localhost:9090/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": <unique-id>,
    "method": "tools/call",
    "params": {
      "name": "<tool-name>",
      "arguments": {
        <tool-specific-args>
      }
    }
  }'
```

**Response Format** (Success):
```json
{
  "jsonrpc": "2.0",
  "id": <matching-id>,
  "result": {
    <tool-specific-result>
  }
}
```

**Response Format** (Error):
```json
{
  "jsonrpc": "2.0",
  "id": <matching-id>,
  "error": {
    "code": <error-code>,
    "message": "<error-message>"
  }
}
```

## Test Cases

### 1. checkpoint_save
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1001,
  "method": "tools/call",
  "params": {
    "name": "checkpoint_save",
    "arguments": {
      "project_path": "/home/user/project",
      "summary": "Test checkpoint",
      "description": "Test description",
      "context": {"key": "value"},
      "tags": ["test"]
    }
  }
}'
```

### 2. checkpoint_search
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1002,
  "method": "tools/call",
  "params": {
    "name": "checkpoint_search",
    "arguments": {
      "query": "test",
      "project_path": "/home/user/project",
      "limit": 5
    }
  }
}'
```

### 3. checkpoint_list
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1003,
  "method": "tools/call",
  "params": {
    "name": "checkpoint_list",
    "arguments": {
      "project_path": "/home/user/project",
      "limit": 10
    }
  }
}'
```

### 4. remediation_save
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1004,
  "method": "tools/call",
  "params": {
    "name": "remediation_save",
    "arguments": {
      "error_msg": "test error",
      "solution": "test solution",
      "project_path": "/home/user/project"
    }
  }
}'
```

### 5. remediation_search
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1005,
  "method": "tools/call",
  "params": {
    "name": "remediation_search",
    "arguments": {
      "error_msg": "test error",
      "limit": 5,
      "project_path": "/home/user/project"
    }
  }
}'
```

### 6. skill_save
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1006,
  "method": "tools/call",
  "params": {
    "name": "skill_save",
    "arguments": {
      "name": "test-skill",
      "description": "Test skill",
      "content": "Test content"
    }
  }
}'
```

### 7. skill_search
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1007,
  "method": "tools/call",
  "params": {
    "name": "skill_search",
    "arguments": {
      "query": "test",
      "limit": 5
    }
  }
}'
```

### 8. index_repository
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1008,
  "method": "tools/call",
  "params": {
    "name": "index_repository",
    "arguments": {
      "path": "/home/user/project"
    }
  }
}'
```

### 9. collection_create
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1009,
  "method": "tools/call",
  "params": {
    "name": "collection_create",
    "arguments": {
      "name": "test-collection",
      "vector_size": 384
    }
  }
}'
```

### 10. collection_list
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1010,
  "method": "tools/call",
  "params": {
    "name": "collection_list",
    "arguments": {}
  }
}'
```

### 11. collection_delete
```bash
curl ... -d '{
  "jsonrpc": "2.0",
  "id": 1011,
  "method": "tools/call",
  "params": {
    "name": "collection_delete",
    "arguments": {
      "name": "test-collection"
    }
  }
}'
```

## Running Tests

```bash
# Make script executable
chmod +x /tmp/test_all_mcp_tools.sh

# Run tests
/tmp/test_all_mcp_tools.sh

# View results
cat /tmp/mcp_test_results.md
```

## Success Criteria

**Test passes if**:
- Response contains `"result"` field (not `"error"`)
- Status code is 200
- Result contains expected data structure

**Test fails if**:
- Response contains `"error"` field
- Status code is non-200
- Response is malformed JSON
