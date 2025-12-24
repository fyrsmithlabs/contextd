# MCP Tool Handlers

This package provides handlers that bridge the MCP (Model Context Protocol) server to contextd's internal services.

## Overview

Each handler wraps a service and provides methods that:
1. Accept JSON input from the MCP server
2. Validate and convert input to service request types
3. Call the appropriate service method
4. Convert service responses to MCP-compatible output

## Handler Files

### checkpoint.go
Handles checkpoint management tools:
- `checkpoint_save` - Save a new checkpoint
- `checkpoint_list` - List checkpoints for a session/project
- `checkpoint_resume` - Resume from a saved checkpoint

**Service Integration**: `checkpoint.Service`

**Example Usage**:
```json
{
  "tool": "checkpoint_save",
  "input": {
    "session_id": "sess_123",
    "tenant_id": "tenant_456",
    "project_path": "/path/to/project",
    "summary": "Implemented authentication",
    "token_count": 15000
  }
}
```

### remediation.go
Handles error remediation tools:
- `remediation_search` - Search for similar error fixes
- `remediation_record` - Record a new error remediation

**Service Integration**: `remediation.Service`

**Example Usage**:
```json
{
  "tool": "remediation_search",
  "input": {
    "query": "nil pointer dereference in auth handler",
    "tenant_id": "tenant_456",
    "limit": 5,
    "min_confidence": 0.7
  }
}
```

### repository.go
Handles repository indexing tools:
- `repository_index` - Index repository files for search

**Service Integration**: `repository.Service`

**Example Usage**:
```json
{
  "tool": "repository_index",
  "input": {
    "path": "/path/to/repository",
    "include_patterns": ["*.go", "*.md"],
    "exclude_patterns": ["vendor/**", "*.test.go"],
    "max_file_size": 1048576
  }
}
```

### troubleshoot.go
Handles AI-powered error diagnosis tools:
- `troubleshoot` - Diagnose an error with AI
- `troubleshoot_pattern` - Save a new error pattern
- `troubleshoot_patterns` - Get all known error patterns

**Service Integration**: `troubleshoot.Service`

**Example Usage**:
```json
{
  "tool": "troubleshoot",
  "input": {
    "error_message": "panic: runtime error: invalid memory address",
    "error_context": "occurred in auth.ValidateToken when token was nil"
  }
}
```

## Registry

The `registry.go` file provides a central registry for all tool handlers:

```go
registry := NewRegistry(
    checkpointSvc,
    remediationSvc,
    repositorySvc,
    troubleshootSvc,
)

// Get handler
handler, err := registry.GetHandler("checkpoint_save")

// Call tool directly
result, err := registry.Call(ctx, "checkpoint_save", inputJSON)

// List all tools
tools := registry.ListTools()
```

## Integration with MCP Server

To integrate these handlers with an MCP server:

1. **Initialize services**:
```go
checkpointSvc, _ := checkpoint.NewService(cfg, qdrantClient, logger)
remediationSvc, _ := remediation.NewService(cfg, qdrantClient, embedder, logger)
repositorySvc := repository.NewService(checkpointSvc)
troubleshootSvc := troubleshoot.NewService(vectorStore, logger, aiClient)
```

2. **Create registry**:
```go
registry := handlers.NewRegistry(
    checkpointSvc,
    remediationSvc,
    repositorySvc,
    troubleshootSvc,
)
```

3. **Wire to MCP server**:
```go
// Example MCP server integration
mcpServer.RegisterToolsHandler(func(ctx context.Context) []ToolDefinition {
    tools := registry.ListTools()
    // Convert to MCP tool definitions
    return convertToMCPDefs(tools)
})

mcpServer.RegisterCallHandler(func(ctx context.Context, toolName string, input json.RawMessage) (interface{}, error) {
    return registry.Call(ctx, toolName, input)
})
```

## Error Handling

All handlers follow consistent error handling:
- Input validation errors return `fmt.Errorf("invalid input: %w", err)`
- Service errors return `fmt.Errorf("failed to <action>: %w", err)`
- Errors preserve original context for debugging

## Response Format

Handlers return `interface{}` containing `map[string]interface{}` with:
- Core response fields (varies by tool)
- Consistent naming (snake_case for JSON compatibility)
- Timestamps in RFC3339 format when applicable

## Testing

Each handler should have corresponding tests in `*_test.go` files:
- Mock service implementations
- Valid input scenarios
- Invalid input handling
- Error propagation

## Security Considerations

- All handlers require `tenant_id` for multi-tenant isolation
- Input is validated before service calls
- Service layer enforces authorization
- No direct file system access (goes through services)

## Future Enhancements

- [ ] Add metrics/observability to handlers
- [ ] Implement handler middleware (auth, logging, rate limiting)
- [ ] Add input schema validation using JSON Schema
- [ ] Generate OpenAPI/tool definitions from handler metadata
