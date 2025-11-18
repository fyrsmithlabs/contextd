# MCP Architecture

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the architectural design of contextd's MCP integration.

---

## Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Claude Code (Client)                  │
└─────────────────┬───────────────────────────────────────┘
                  │ HTTP/SSE (JSON-RPC 2.0)
                  │ POST/GET /mcp
┌─────────────────▼───────────────────────────────────────┐
│                    MCP Server (pkg/mcp)                  │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Tool Registry (16 tools)                          │ │
│  ├────────────────────────────────────────────────────┤ │
│  │  Input Validation & Type Conversion                │ │
│  ├────────────────────────────────────────────────────┤ │
│  │  Rate Limiting (per-connection, per-tool)          │ │
│  ├────────────────────────────────────────────────────┤ │
│  │  OpenTelemetry (traces + metrics)                  │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────┬───────────────────────────────────────┘
                  │
    ┌─────────────┼─────────────┬─────────────┬──────────┐
    │             │             │             │          │
┌───▼────┐  ┌────▼────┐  ┌─────▼────┐  ┌────▼────┐  ┌─▼──┐
│Checkpoint│ │Remediation│ │Trouble-  │ │Skills   │ │Analytics│
│Service   │ │Service    │ │shooting  │ │Service  │ │Service│
└───┬────┘  └────┬────┘  └─────┬────┘  └────┬────┘  └─┬──┘
    │             │             │             │          │
    └─────────────┴─────────────┴─────────────┴──────────┘
                              │
                    ┌─────────▼──────────┐
                    │  Vector Store      │
                    └────────────────────┘
```

---

## Transport Layer

### HTTP/SSE Transport

**MCP Streamable HTTP** (specification version 2025-03-26):

- **Protocol**: HTTP/1.1 with Server-Sent Events (SSE)
- **Endpoint**: POST/GET `/mcp` (single endpoint for all MCP operations)
- **Format**: JSON-RPC 2.0
- **Port**: 8080 (configurable via CONTEXTD_HTTP_PORT)
- **Host**: 0.0.0.0 (accepts remote connections)
- **Session Management**: `Mcp-Session-Id` header for multi-client support

**Request Format**:
```json
{
  "jsonrpc": "2.0",
  "id": "unique-request-id",
  "method": "tools/call",
  "params": {
    "name": "checkpoint_save",
    "arguments": {
      "summary": "Completed feature X",
      "project_path": "/path/to/project"
    }
  }
}
```

**Response Format**:
```json
{
  "jsonrpc": "2.0",
  "id": "unique-request-id",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"id\":\"cp-123\",\"summary\":\"Completed feature X\",\"created_at\":\"2025-11-04T12:00:00Z\",\"token_count\":42}"
      }
    ]
  }
}
```

### Connection Management

- **Multiple Connections**: HTTP server handles multiple concurrent connections (multi-session support)
- **Session Identification**: `Mcp-Session-Id` header tracks individual client sessions
- **Rate Limiting**: Per-connection, per-tool rate limiting
- **Context Propagation**: Request context flows through all operations
- **Timeout Handling**: Operation-specific timeouts enforced
- **Remote Access**: Supports remote connections from distributed teams

---

## MCP Server Design

### Tool Registry

**16 Tools Registered**:
- Session Management: `checkpoint_save`, `checkpoint_search`, `checkpoint_list`
- Error Resolution: `remediation_save`, `remediation_search`
- AI Diagnosis: `troubleshoot`, `list_patterns`
- Repository Indexing: `index_repository`
- Skills Management: `skill_create`, `skill_search`, `skill_list`, `skill_update`, `skill_delete`, `skill_apply`
- System Operations: `status`, `analytics_get`

**Tool Registration**:
```go
mcpServer, err := mcp.NewServer(mcpServices)
if err != nil {
    return fmt.Errorf("failed to create MCP server: %w", err)
}
```

### Input Validation & Type Conversion

**Type Conversion Functions**:
```go
// Context metadata: map[string]interface{} → map[string]string
func contextToStringMap(ctx map[string]interface{}) map[string]string

// Context metadata: map[string]string → map[string]interface{}
func contextToInterfaceMap(ctx map[string]string) map[string]interface{}

// Safe timestamp conversion with validation
func safeTimestamp(ts int64) time.Time
```

**Validation Flow**:
1. Parse Input: Unmarshal JSON to input struct
2. Validate Fields: Check required fields and constraints
3. Create Request: Convert to service request struct
4. Call Service: Execute service operation with context
5. Convert Output: Transform service response to output struct
6. Record Metrics: Log telemetry data

### Rate Limiting

**Per-Connection, Per-Tool Isolation**:
```go
func (rl *MCPRateLimiter) Allow(connectionID, toolName string) bool {
    key := connectionID + ":" + toolName
    limiterI, _ := rl.limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(rps), burst))
    limiter := limiterI.(*rate.Limiter)
    return limiter.Allow()
}
```

**Default Limits**:
- Default RPS: 10 requests per second per tool
- Default Burst: 20 requests
- Algorithm: Token bucket

**Tool-Specific Overrides**:
```go
"troubleshoot": {
    RPS:   2,   // Slower for expensive AI operations
    Burst: 5,
}
"skill_create": {
    RPS:   5,   // Moderate for large embeddings
    Burst: 10,
}
```

### OpenTelemetry Instrumentation

**Tracing**:
```go
func (s *Server) handleToolName(ctx context.Context, req *mcpsdk.CallToolRequest, input ToolInput) (*mcpsdk.CallToolResult, ToolOutput, error) {
    startTime := time.Now()
    ctx, span := startToolSpan(ctx, "tool_name")
    defer span.End()

    // Set attributes
    span.SetAttributes(
        attribute.String("tool.name", "tool_name"),
        attribute.String("project.path", input.ProjectPath),
    )

    // Record error or success
    if err != nil {
        span.RecordError(err)
        recordToolError(ctx, span, "tool_name", err, time.Since(startTime))
    } else {
        recordToolSuccess(ctx, span, "tool_name", time.Since(startTime))
    }
}
```

**Metrics**:
- HTTP request duration (histogram)
- HTTP request count (counter)
- Active connections (gauge)
- HTTP status codes (labels)
- MCP tool call performance
- Vector store operation timing
- Embedding generation duration

---

## Service Layer Integration

### Services Struct

```go
type Services struct {
    Checkpoint      *checkpoint.Service
    Remediation     *remediation.Service
    Troubleshooting *troubleshoot.Service
    Skills          *skills.Service
    Analytics       *analytics.Service
}
```

### Tool Handler Pattern

**Signature**:
```go
func (s *Server) handleToolName(
    ctx context.Context,
    req *mcpsdk.CallToolRequest,
    input ToolInput,
) (*mcpsdk.CallToolResult, ToolOutput, error)
```

**Example Implementation**:
```go
func (s *Server) handleCheckpointSave(ctx context.Context, req *mcpsdk.CallToolRequest, input CheckpointSaveInput) (*mcpsdk.CallToolResult, CheckpointSaveOutput, error) {
    startTime := time.Now()
    ctx, span := startToolSpan(ctx, "checkpoint_save")
    defer span.End()

    // Set timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Validate inputs
    if err := validateSummary(input.Summary); err != nil {
        mcpErr := NewValidationError("invalid summary", map[string]interface{}{
            "field": "summary",
            "error": err.Error(),
        })
        recordToolError(ctx, span, "checkpoint_save", mcpErr, time.Since(startTime))
        return nil, CheckpointSaveOutput{}, mcpErr
    }

    // Call service
    result, err := s.services.Checkpoint.Create(ctx, &validation.CreateCheckpointRequest{
        Summary:     input.Summary,
        Description: input.Description,
        ProjectPath: input.ProjectPath,
        Context:     contextToStringMap(input.Context),
        Tags:        input.Tags,
    })
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            mcpErr := NewTimeoutError("checkpoint creation timed out", err)
            recordToolError(ctx, span, "checkpoint_save", mcpErr, time.Since(startTime))
            return nil, CheckpointSaveOutput{}, mcpErr
        }
        mcpErr := NewInternalError("failed to create checkpoint", err)
        recordToolError(ctx, span, "checkpoint_save", mcpErr, time.Since(startTime))
        return nil, CheckpointSaveOutput{}, mcpErr
    }

    // Return output
    output := CheckpointSaveOutput{
        ID:         result.ID,
        Summary:    result.Summary,
        CreatedAt:  result.CreatedAt,
        TokenCount: result.TokenCount,
    }

    recordToolSuccess(ctx, span, "checkpoint_save", time.Since(startTime))
    return nil, output, nil
}
```

---

## Design Patterns

### Adapter Pattern

**MCP server adapts contextd services to MCP protocol**:
- Converts MCP JSON-RPC requests to service calls
- Transforms service responses to MCP JSON-RPC responses
- Handles protocol-specific concerns (timeouts, errors, telemetry)

### Strategy Pattern

**Different validation strategies per tool**:
- Checkpoint validation: Summary, description, project path
- Remediation validation: Error message, error type, solution
- Skill validation: Name, content, version, category
- Troubleshoot validation: Error message, stack trace, context

### Observer Pattern

**OpenTelemetry observes all tool invocations**:
- Traces tool execution with spans
- Records metrics for performance monitoring
- Logs errors with structured context

---

## Error Handling

### MCPError Structure

```go
type MCPError struct {
    Category ErrorCategory          // Error type
    Message  string                 // Human-readable message
    Details  map[string]interface{} // Additional context
    Cause    error                  // Underlying error (not serialized)
}
```

### Error Categories

| Category | Description | HTTP Equivalent |
|----------|-------------|-----------------|
| `validation` | Invalid input provided | 400 Bad Request |
| `not_found` | Resource not found | 404 Not Found |
| `internal` | Internal server error | 500 Internal Server Error |
| `timeout` | Operation timed out | 504 Gateway Timeout |
| `unauthorized` | Unauthorized access | 401 Unauthorized |

### Error Propagation

```go
// Service error
result, err := s.services.Checkpoint.Create(ctx, req)
if err != nil {
    // Check for timeout
    if errors.Is(err, context.DeadlineExceeded) {
        mcpErr := NewTimeoutError("checkpoint creation timed out", err)
        recordToolError(ctx, span, "checkpoint_save", mcpErr, time.Since(startTime))
        return nil, CheckpointSaveOutput{}, mcpErr
    }

    // Generic internal error
    mcpErr := NewInternalError("failed to create checkpoint", err)
    recordToolError(ctx, span, "checkpoint_save", mcpErr, time.Since(startTime))
    return nil, CheckpointSaveOutput{}, mcpErr
}
```

---

## Data Models

### Input/Output Types

**Example Type Definition**:
```go
type CheckpointSaveInput struct {
    Summary     string                 `json:"summary" jsonschema:"required,Brief summary of checkpoint (max 500 chars)"`
    Description string                 `json:"description,omitempty" jsonschema:"Detailed description (optional)"`
    ProjectPath string                 `json:"project_path" jsonschema:"required,Absolute path to project directory"`
    Context     map[string]interface{} `json:"context,omitempty" jsonschema:"Additional context metadata"`
    Tags        []string               `json:"tags,omitempty" jsonschema:"Tags for categorization"`
}
```

### Validation Constraints

**Constants** (defined in `pkg/mcp/constants.go`):
```go
const (
    MaxSummaryLength      = 500
    MaxDescriptionLength  = 5000
    MaxErrorMessageLength = 10000
    MaxStackTraceLength   = 50000
    MaxTags               = 20
    MaxTagLength          = 50
    MaxQueryLength        = 1000
    MaxContextFields      = 50
    MaxContextValueLength = 1000
)
```

---

## Implementation Files

| File | Purpose |
|------|---------|
| `pkg/mcp/server.go` | MCP server implementation and tool registration |
| `pkg/mcp/tools.go` | Checkpoint, remediation, and troubleshooting tool handlers |
| `pkg/mcp/skills_tools.go` | Skills management tool handlers |
| `pkg/mcp/analytics_tool.go` | Analytics tool handler |
| `pkg/mcp/types.go` | Input/output type definitions with JSON schemas |
| `pkg/mcp/errors.go` | Error types and constructors |
| `pkg/mcp/validation.go` | Input validation functions |
| `pkg/mcp/constants.go` | Timeouts and validation constants |
| `pkg/mcp/telemetry.go` | OpenTelemetry instrumentation |
| `cmd/contextd/main.go` | Server initialization and lifecycle |

---

## Summary

**Key Architectural Decisions**:
- ✅ HTTP/SSE transport for remote access and multi-session support
- ✅ Single `/mcp` endpoint for all JSON-RPC operations (MCP spec 2025-03-26)
- ✅ Adapter pattern to bridge contextd services and MCP protocol
- ✅ Per-connection, per-tool rate limiting for abuse prevention
- ✅ OpenTelemetry for comprehensive observability
- ✅ Type safety at multiple levels (compile-time, schema, runtime)
- ✅ Structured error responses with categories
- ✅ Official MCP SDK for protocol compliance
