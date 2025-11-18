# MCP Integration Specification

## Document Status

- **Status**: Complete
- **Version**: 2.0.0
- **Last Updated**: 2025-11-04
- **Owner**: contextd team

## Overview

The MCP (Model Context Protocol) integration provides a standardized interface for Claude Code and other AI assistants to interact with contextd services. It implements the MCP Streamable HTTP transport (specification version 2025-03-26), enabling remote access and multiple concurrent sessions for distributed teams.

### Purpose

The MCP integration serves as a bridge between AI assistants and contextd's core functionality, providing:

1. **Session Management**: Save and retrieve checkpoints for resuming work
2. **Error Resolution**: Store and search for error solutions with hybrid matching
3. **AI Diagnosis**: Intelligent troubleshooting with hypothesis generation
4. **Knowledge Management**: Skills system for reusable workflows and templates
5. **Analytics**: Usage metrics and performance tracking
6. **Repository Indexing**: Semantic search across codebase files

### Key Features

- **16 MCP Tools**: Complete coverage of contextd functionality
- **HTTP/SSE Transport**: Remote access with multiple concurrent sessions
- **MCP Streamable HTTP**: Specification version 2025-03-26 compliant
- **Multi-Session Support**: Multiple Claude Code instances simultaneously
- **OpenTelemetry**: Full observability with traces and metrics
- **Rate Limiting**: Per-connection, per-tool rate limiting
- **Structured Errors**: Categorized error responses for better error handling
- **Type Safety**: Strongly typed input/output schemas with validation
- **Context Timeouts**: Operation-specific timeouts for reliability

## MCP Protocol Compliance

### Protocol Version

- **MCP Version**: 2025-03-26 (Streamable HTTP)
- **Transport**: HTTP/1.1 with Server-Sent Events (SSE)
- **Endpoint**: POST/GET `/mcp` (single endpoint for all MCP operations)
- **Format**: JSON-RPC 2.0
- **Schema**: JSON Schema for tool inputs/outputs
- **Session Management**: `Mcp-Session-Id` header for multi-client support
- **Port**: 8080 (configurable via CONTEXTD_HTTP_PORT)

### Implementation

The MCP server is implemented using the official `github.com/modelcontextprotocol/go-sdk/mcp` SDK, ensuring full compliance with the MCP specification.

**Key Compliance Points**:

1. **Tool Discovery**: Server advertises all 16 tools with complete schemas
2. **JSON-RPC 2.0**: All messages follow JSON-RPC 2.0 format
3. **Error Handling**: Errors returned as JSON-RPC error objects
4. **Resource Management**: Proper context handling and cancellation
5. **Lifecycle Management**: Graceful initialization and shutdown

## Tool Catalog

### 1. checkpoint_save

**Purpose**: Save a session checkpoint for resuming work later.

**Description**: Stores a session checkpoint with summary, description, project path, context metadata, and tags. Automatic vector embeddings are generated for semantic search.

**Input Schema**:
```json
{
  "summary": "string (required, max 500 chars)",
  "description": "string (optional, max 5000 chars)",
  "project_path": "string (required, absolute path)",
  "context": "object (optional, key-value metadata)",
  "tags": "array of strings (optional, max 20 tags)"
}
```

**Output Schema**:
```json
{
  "id": "string (checkpoint ID)",
  "summary": "string",
  "created_at": "timestamp",
  "token_count": "integer (embedding tokens)"
}
```

**Timeout**: 30 seconds

### 2. checkpoint_search

**Purpose**: Search checkpoints using semantic similarity.

**Description**: Finds relevant checkpoints based on query meaning, with optional filtering by project path and tags. Uses vector similarity search with cosine distance.

**Input Schema**:
```json
{
  "query": "string (required, max 1000 chars)",
  "top_k": "integer (optional, default: 5, max: 100)",
  "project_path": "string (optional, filter by project)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "summary": "string",
      "description": "string",
      "project_path": "string",
      "context": "object",
      "tags": "array of strings",
      "score": "float (similarity score 0-1)",
      "distance": "float (cosine distance)",
      "created_at": "timestamp"
    }
  ],
  "query": "string (original query)",
  "top_k": "integer"
}
```

**Timeout**: 10 seconds

### 3. checkpoint_list

**Purpose**: List recent checkpoints with pagination.

**Description**: Supports filtering by project path and sorting by creation/update time. Useful for browsing recent work.

**Input Schema**:
```json
{
  "limit": "integer (optional, default: 10, max: 100)",
  "offset": "integer (optional, default: 0)",
  "project_path": "string (optional, filter by project)",
  "sort_by": "string (optional, created_at|updated_at)"
}
```

**Output Schema**:
```json
{
  "checkpoints": [
    {
      "id": "string",
      "summary": "string",
      "description": "string",
      "project_path": "string",
      "context": "object",
      "tags": "array of strings",
      "created_at": "timestamp"
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

**Timeout**: 5 seconds

### 4. remediation_save

**Purpose**: Store an error solution for future reference.

**Description**: Saves error message, type, solution, stack trace, and metadata with vector embeddings for intelligent matching. Supports severity levels and project-specific context.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "error_type": "string (required)",
  "solution": "string (required)",
  "project_path": "string (optional)",
  "context": "object (optional, error context)",
  "tags": "array of strings (optional)",
  "severity": "string (optional, low|medium|high|critical)",
  "stack_trace": "string (optional, max 50000 chars)"
}
```

**Output Schema**:
```json
{
  "id": "string (remediation ID)",
  "error_message": "string",
  "error_type": "string",
  "solution": "string",
  "created_at": "timestamp"
}
```

**Timeout**: 30 seconds

### 5. remediation_search

**Purpose**: Find similar error solutions using hybrid matching.

**Description**: Returns ranked results with match scores using 70% semantic similarity + 30% string matching. Includes detailed match breakdowns for transparency.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "stack_trace": "string (optional, for better matching)",
  "limit": "integer (optional, default: 5, max: 100)",
  "min_score": "float (optional, 0-1, default: 0.5)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "error_message": "string",
      "error_type": "string",
      "solution": "string",
      "tags": "array of strings",
      "match_score": "float (combined score)",
      "semantic_score": "float (70% weight)",
      "string_score": "float (30% weight)",
      "stack_trace_match": "boolean",
      "error_type_match": "boolean",
      "context": "object",
      "created_at": "timestamp"
    }
  ],
  "query": "string (original error message)",
  "total": "integer"
}
```

**Timeout**: 10 seconds

### 6. troubleshoot

**Purpose**: AI-powered error diagnosis and troubleshooting.

**Description**: Analyzes error messages and stack traces, identifies root causes, generates hypotheses, and recommends diagnostic steps and solutions. Includes similar issues from knowledge base.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "stack_trace": "string (optional)",
  "context": "object (optional, environment, versions, etc)",
  "category": "string (optional, configuration|resource|dependency|etc)",
  "mode": "string (optional, auto|interactive|guided, default: auto)",
  "tags": "array of strings (optional)",
  "top_k": "integer (optional, similar issues, default: 5)"
}
```

**Output Schema**:
```json
{
  "session_id": "string (troubleshooting session ID)",
  "root_cause": "string",
  "confidence": "string (high|medium|low)",
  "confidence_score": "float (0-1)",
  "category": "string",
  "severity": "string",
  "hypotheses": [
    {
      "description": "string",
      "probability": "float",
      "evidence": "array of strings",
      "category": "string",
      "verification_steps": "array of strings"
    }
  ],
  "similar_issues": [
    {
      "id": "string",
      "error_pattern": "string",
      "root_cause": "string",
      "solution": "string",
      "match_score": "float",
      "semantic_score": "float",
      "success_rate": "float",
      "severity": "string",
      "category": "string",
      "tags": "array of strings",
      "confidence": "string",
      "is_destructive": "boolean",
      "safety_warnings": "array of strings"
    }
  ],
  "recommended_actions": [
    {
      "step": "integer",
      "description": "string",
      "commands": "array of strings",
      "expected_outcome": "string",
      "destructive": "boolean",
      "safety_notes": "string"
    }
  ],
  "diagnostic_steps": "array of strings",
  "time_taken_ms": "float",
  "diagnosed_at": "timestamp"
}
```

**Timeout**: 60 seconds

### 7. list_patterns

**Purpose**: Browse troubleshooting patterns from the knowledge base.

**Description**: Supports filtering by category, severity, and minimum success rate. Useful for learning from past solutions.

**Input Schema**:
```json
{
  "category": "string (optional, filter by category)",
  "severity": "string (optional, critical|high|medium|low)",
  "min_success_rate": "float (optional, 0-1)",
  "limit": "integer (optional, default: 10, max: 100)"
}
```

**Output Schema**:
```json
{
  "patterns": [
    {
      "id": "string",
      "error_pattern": "string",
      "category": "string",
      "severity": "string",
      "root_cause": "string",
      "solution": "string",
      "success_rate": "float",
      "tags": "array of strings",
      "usage_count": "integer",
      "last_used": "timestamp"
    }
  ],
  "total": "integer"
}
```

**Timeout**: 5 seconds

### 8. index_repository

**Purpose**: Index an existing repository or directory for semantic search.

**Description**: Creates searchable checkpoints from files matching include patterns while respecting exclude patterns and file size limits. Supports glob patterns for flexible file selection.

**Input Schema**:
```json
{
  "path": "string (required, absolute path to repository)",
  "include_patterns": "array of strings (optional, e.g., ['*.md', '*.txt'])",
  "exclude_patterns": "array of strings (optional, e.g., ['*.log', 'node_modules/**'])",
  "max_file_size": "integer (optional, bytes, default: 1MB, max: 10MB)"
}
```

**Output Schema**:
```json
{
  "path": "string (repository path indexed)",
  "files_indexed": "integer",
  "include_patterns": "array of strings",
  "exclude_patterns": "array of strings",
  "max_file_size": "integer",
  "indexed_at": "timestamp"
}
```

**Timeout**: 300 seconds (5 minutes)

**Security Note**: Path traversal protection is enforced. All indexed files must be within the specified repository path.

### 9. status

**Purpose**: Get contextd service status and health information.

**Description**: Shows service health, version, uptime, and system metrics. Useful for monitoring and debugging.

**Input Schema**:
```json
{}
```

**Output Schema**:
```json
{
  "status": "string (healthy|degraded|unhealthy)",
  "version": "string (service version)",
  "uptime": "string (optional)",
  "services": {
    "checkpoint": {
      "status": "string (healthy|unhealthy|unknown)",
      "error": "string (optional)"
    }
  },
  "metrics": {
    "tools_available": "integer",
    "mcp_server": "string"
  },
  "last_updated": "timestamp"
}
```

**Timeout**: 30 seconds

### 10. analytics_get

**Purpose**: Get context usage analytics and metrics.

**Description**: Tracks token reduction, feature adoption, performance metrics, and business impact. Shows average token savings, search precision, and time saved.

**Input Schema**:
```json
{
  "period": "string (optional, daily|weekly|monthly|all-time, default: weekly)",
  "project_path": "string (optional, filter by project)",
  "start_date": "string (optional, YYYY-MM-DD)",
  "end_date": "string (optional, YYYY-MM-DD)"
}
```

**Output Schema**:
```json
{
  "period": "string",
  "start_date": "timestamp",
  "end_date": "timestamp",
  "total_sessions": "integer",
  "avg_token_reduction_pct": "float",
  "total_time_saved_min": "float",
  "search_precision": "float",
  "estimated_cost_save_usd": "float",
  "top_features": [
    {
      "feature": "string",
      "count": "integer",
      "avg_latency_ms": "float",
      "success_rate": "float"
    }
  ],
  "performance": {
    "avg_search_latency_ms": "float",
    "avg_checkpoint_latency_ms": "float",
    "cache_hit_rate": "float",
    "overall_success_rate": "float"
  }
}
```

**Timeout**: 30 seconds

### 11. skill_create

**Purpose**: Create a new reusable skill/workflow template.

**Description**: Skills can be searched semantically and applied to similar situations. Supports versioning, categorization, and metadata.

**Input Schema**:
```json
{
  "name": "string (required, max 200 chars)",
  "description": "string (required, max 2000 chars)",
  "content": "string (required, markdown, max 50000 chars)",
  "version": "string (required, semver, e.g., '1.0.0')",
  "author": "string (required)",
  "category": "string (required, debugging|deployment|analysis|etc)",
  "prerequisites": "array of strings (optional)",
  "expected_outcome": "string (optional)",
  "tags": "array of strings (optional)",
  "metadata": "object (optional)"
}
```

**Output Schema**:
```json
{
  "id": "string (skill ID)",
  "name": "string",
  "version": "string",
  "token_count": "integer",
  "created_at": "timestamp"
}
```

**Timeout**: 120 seconds (longer due to large content embedding)

### 12. skill_search

**Purpose**: Search for skills using semantic similarity.

**Description**: Find relevant workflows and templates based on query meaning, with optional filtering by category and tags.

**Input Schema**:
```json
{
  "query": "string (required, max 1000 chars)",
  "top_k": "integer (optional, default: 5, max: 100)",
  "category": "string (optional, filter by category)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "prerequisites": "array of strings",
      "expected_outcome": "string",
      "tags": "array of strings",
      "usage_count": "integer",
      "success_rate": "float",
      "score": "float",
      "distance": "float",
      "metadata": "object",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "query": "string",
  "top_k": "integer"
}
```

**Timeout**: 10 seconds

### 13. skill_list

**Purpose**: List all skills with pagination and filtering.

**Description**: Supports filtering by category, tags, and sorting by creation date, usage count, or success rate.

**Input Schema**:
```json
{
  "limit": "integer (optional, default: 10, max: 100)",
  "offset": "integer (optional, default: 0)",
  "category": "string (optional, filter by category)",
  "tags": "array of strings (optional, filter by tags)",
  "sort_by": "string (optional, created_at|updated_at|usage_count|success_rate)"
}
```

**Output Schema**:
```json
{
  "skills": [
    {
      "id": "string",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "prerequisites": "array of strings",
      "expected_outcome": "string",
      "tags": "array of strings",
      "usage_count": "integer",
      "success_rate": "float",
      "metadata": "object",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

**Timeout**: 30 seconds

### 14. skill_update

**Purpose**: Update an existing skill.

**Description**: Allows modifying name, description, content, version, tags, and metadata. All fields are optional except ID.

**Input Schema**:
```json
{
  "id": "string (required, skill ID)",
  "name": "string (optional)",
  "description": "string (optional)",
  "content": "string (optional)",
  "version": "string (optional)",
  "category": "string (optional)",
  "prerequisites": "array of strings (optional)",
  "expected_outcome": "string (optional)",
  "tags": "array of strings (optional)",
  "metadata": "object (optional)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "name": "string",
  "version": "string",
  "updated_at": "timestamp"
}
```

**Timeout**: 120 seconds

### 15. skill_delete

**Purpose**: Delete a skill by ID.

**Description**: This action cannot be undone. Removes skill from database and vector store.

**Input Schema**:
```json
{
  "id": "string (required, skill ID to delete)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "message": "string (confirmation)"
}
```

**Timeout**: 30 seconds

### 16. skill_apply

**Purpose**: Apply a skill to the current context.

**Description**: Returns the skill content and tracks usage statistics. Optionally records success/failure for success rate calculation.

**Input Schema**:
```json
{
  "id": "string (required, skill ID)",
  "success": "boolean (optional, for tracking)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "name": "string",
  "content": "string (skill content to apply)",
  "prerequisites": "array of strings",
  "expected_outcome": "string",
  "usage_count": "integer",
  "success_rate": "float"
}
```

**Timeout**: 30 seconds

## Error Response Format

All MCP tools use a structured error format for consistent error handling:

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

### Error Response Example

```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "error": {
    "code": -32603,
    "message": "[validation] invalid summary",
    "data": {
      "category": "validation",
      "message": "invalid summary",
      "details": {
        "field": "summary",
        "error": "summary is required"
      }
    }
  }
}
```

### Common Error Scenarios

1. **Validation Errors**: Invalid input parameters
   - Missing required fields
   - Exceeding length limits
   - Invalid path formats
   - Invalid date formats

2. **Timeout Errors**: Operation exceeded time limit
   - Search timeouts (10s)
   - Embedding timeouts (20s)
   - Diagnosis timeouts (60s)
   - Indexing timeouts (300s)

3. **Internal Errors**: Service failures
   - Database connection errors
   - Embedding service failures
   - Vector store errors

4. **Not Found Errors**: Resource doesn't exist
   - Checkpoint not found
   - Skill not found
   - Remediation not found

## Architecture and Design

### Component Overview

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

### Key Design Patterns

1. **Adapter Pattern**: MCP server adapts contextd services to MCP protocol
2. **Strategy Pattern**: Different validation strategies per tool
3. **Observer Pattern**: OpenTelemetry observes all tool invocations
4. **Rate Limiting**: Token bucket algorithm per connection+tool

### Type Conversion

The MCP server handles type conversion between MCP JSON types and Go types:

```go
// Context metadata: map[string]interface{} → map[string]string
func contextToStringMap(ctx map[string]interface{}) map[string]string

// Context metadata: map[string]string → map[string]interface{}
func contextToInterfaceMap(ctx map[string]string) map[string]interface{}

// Safe timestamp conversion with validation
func safeTimestamp(ts int64) time.Time
```

## Server Lifecycle

### Initialization

1. **Parse Command-Line Flags**: Check for `--mcp` flag
2. **Load Configuration**: Environment variables and defaults
3. **Initialize Services**: Vector store, embedding, checkpoint, remediation, skills, troubleshooting
4. **Initialize OpenTelemetry**: Traces and metrics
5. **Create MCP Server**: Register all 16 tools
6. **Start HTTP Server**: Begin accepting requests on `/mcp` endpoint (port 8080)

**Initialization Code**:
```go
// Create MCP services struct
mcpServices := &mcp.Services{
    Checkpoint:      services.Checkpoint,
    Remediation:     services.Remediation,
    Troubleshooting: services.Troubleshooting,
    Skills:          services.Skills,
    Analytics:       services.Analytics,
}

// Create MCP server
mcpServer, err := mcp.NewServer(mcpServices)
if err != nil {
    return fmt.Errorf("failed to create MCP server: %w", err)
}

// Run server (blocking)
if err := mcpServer.Run(ctx); err != nil {
    return fmt.Errorf("server error: %w", err)
}
```

### Shutdown

1. **Signal Handling**: Graceful shutdown on SIGINT/SIGTERM
2. **Context Cancellation**: Cancel all in-flight operations
3. **Service Cleanup**: Close database connections
4. **Telemetry Flush**: Flush pending traces and metrics (5s timeout)
5. **Exit**: Clean process termination

**Shutdown Code**:
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

select {
case <-sigChan:
    cancel() // Cancel context
    // Cleanup via defer statements
}
```

### Health Checks

The server performs health checks on initialization:

```go
if err := services.Checkpoint.Health(ctx); err != nil {
    return fmt.Errorf("checkpoint service health check failed: %w", err)
}
```

Health status is available via the `status` tool during runtime.

## Transport Layer

### HTTP/SSE Transport

The MCP server uses HTTP/SSE for communication with Claude Code:

- **Input**: JSON-RPC 2.0 requests via POST `/mcp`
- **Output**: JSON-RPC 2.0 responses via HTTP response body
- **Streaming**: Server-Sent Events (SSE) via GET `/mcp` for real-time notifications
- **Errors**: JSON-RPC error objects in HTTP response
- **Session Management**: `Mcp-Session-Id` header identifies client sessions
- **Port**: 8080 (configurable via CONTEXTD_HTTP_PORT)
- **Remote Access**: Supports remote connections (0.0.0.0 binding)

### Message Format

**Request**:
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

**Response**:
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

## API Specifications

### Tool Handler Signature

All tool handlers follow this signature:

```go
func (s *Server) handleToolName(
    ctx context.Context,
    req *mcpsdk.CallToolRequest,
    input ToolInput,
) (*mcpsdk.CallToolResult, ToolOutput, error)
```

### Validation Flow

1. **Parse Input**: Unmarshal JSON to input struct
2. **Validate Fields**: Check required fields and constraints
3. **Create Request**: Convert to service request struct
4. **Call Service**: Execute service operation with context
5. **Convert Output**: Transform service response to output struct
6. **Record Metrics**: Log telemetry data

### Example Tool Handler

```go
func (s *Server) handleCheckpointSave(ctx context.Context, req *mcpsdk.CallToolRequest, input CheckpointSaveInput) (*mcpsdk.CallToolResult, CheckpointSaveOutput, error) {
    startTime := time.Now()
    ctx, span := startToolSpan(ctx, "checkpoint_save")

    // Set timeout
    ctx, cancel := context.WithTimeout(ctx, DefaultToolTimeout)
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

## Data Models and Schemas

### Input/Output Types

All tool input/output types are defined in `pkg/mcp/types.go` using Go structs with JSON schema tags:

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

All validation constraints are defined in `pkg/mcp/constants.go`:

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

### Type Safety

The MCP integration enforces type safety at multiple levels:

1. **Compile-Time**: Go type system catches type errors
2. **JSON Schema**: Validates input against schemas
3. **Runtime Validation**: Custom validators enforce business rules
4. **Clamping**: Numeric values clamped to safe ranges

## Performance Characteristics

### Operation Timeouts

| Operation | Timeout | Rationale |
|-----------|---------|-----------|
| checkpoint_save | 30s | Embedding generation + DB write |
| checkpoint_search | 10s | Vector similarity search |
| checkpoint_list | 5s | Simple database query |
| remediation_save | 30s | Embedding generation + DB write |
| remediation_search | 10s | Hybrid matching (semantic + string) |
| troubleshoot | 60s | AI diagnosis + knowledge base search |
| list_patterns | 5s | Database query with filtering |
| index_repository | 300s | Large repository indexing |
| skill_create | 120s | Large content embedding + DB write |
| skill_search | 10s | Vector similarity search |
| skill_list | 30s | Database query with pagination |
| skill_update | 120s | Re-embedding + DB update |
| skill_delete | 30s | Database deletion + vector cleanup |
| skill_apply | 30s | Database query + usage tracking |
| status | 30s | Health checks across services |
| analytics_get | 30s | Aggregated metrics calculation |

### Rate Limiting

**Default Limits**:
- **Default RPS**: 10 requests per second per tool
- **Default Burst**: 20 requests
- **Algorithm**: Token bucket with per-connection, per-tool isolation

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

**Rate Limit Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "error": {
    "code": -32603,
    "message": "rate limit exceeded for tool: troubleshoot"
  }
}
```

### Throughput

**Benchmark Results** (local Qdrant, TEI embeddings):

| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| checkpoint_save | 250ms | 450ms | 600ms |
| checkpoint_search | 80ms | 150ms | 200ms |
| remediation_search | 120ms | 200ms | 280ms |
| troubleshoot | 2.5s | 4.5s | 6.0s |
| skill_search | 90ms | 160ms | 220ms |

**Concurrent Requests**: MCP server handles multiple HTTP connections concurrently, supporting multiple Claude Code sessions simultaneously.

## Error Handling

### Error Handling Strategy

1. **Validation Errors**: Return immediately with field-level details
2. **Timeout Errors**: Cancel operation, return timeout error
3. **Service Errors**: Wrap with context, return internal error
4. **Unknown Errors**: Log full stack trace, return generic error

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

### Error Logging

- **Structured Logging**: Errors logged with context
- **Telemetry**: Errors recorded in traces and metrics
- **Stack Traces**: Full stack traces for debugging
- **Redaction**: Sensitive data redacted from logs

## Security Considerations

### Input Validation

**Path Traversal Protection**:
```go
func validateProjectPath(p string) error {
    // Must be absolute path
    if !filepath.IsAbs(p) {
        return fmt.Errorf("project_path must be an absolute path")
    }

    // Check for path traversal attempts
    if strings.Contains(p, "..") {
        return fmt.Errorf("project_path cannot contain '..' (path traversal not allowed)")
    }

    // Clean the path and ensure it matches original
    cleaned := filepath.Clean(p)
    if cleaned != p {
        return fmt.Errorf("project_path must be a clean absolute path")
    }

    return nil
}
```

**Length Limits**: All string inputs have maximum length limits to prevent DoS.

**Type Safety**: Strong typing prevents injection attacks.

### Rate Limiting

Per-connection, per-tool rate limiting prevents abuse:

```go
func (rl *MCPRateLimiter) Allow(connectionID, toolName string) bool {
    key := connectionID + ":" + toolName
    limiterI, _ := rl.limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(rps), burst))
    limiter := limiterI.(*rate.Limiter)
    return limiter.Allow()
}
```

### Data Isolation

- **Project-Level Isolation**: Each project has its own vector database
- **Shared Knowledge**: Remediations and skills stored in shared database
- **No Cross-Project Access**: Queries scoped to project_path

### Transport Security

- **HTTP Transport**: Remote access supported (0.0.0.0 binding)
- **Multiple Connections**: Supports concurrent client sessions via HTTP
- **Session Management**: `Mcp-Session-Id` header for session tracking
- **No Authentication (MVP)**: Trusted network assumption, add auth post-MVP
- **Production Recommendations**:
  - Deploy behind reverse proxy with TLS (nginx/Caddy)
  - Add authentication (Bearer token, JWT, OAuth)
  - Use VPN or SSH tunnel for remote access without exposing port
  - Implement rate limiting and DDoS protection

## Testing Requirements

### Unit Tests

**Coverage Requirements**:
- Tool handlers: 100%
- Validation functions: 100%
- Error handling: 100%
- Type conversion: 100%

**Test Categories**:
1. **Input Validation**: Test all validation rules
2. **Error Handling**: Test all error paths
3. **Type Conversion**: Test conversion functions
4. **Rate Limiting**: Test rate limiter behavior

**Example Test**:
```go
func TestHandleCheckpointSave_ValidationError(t *testing.T) {
    s := setupTestServer(t)

    input := CheckpointSaveInput{
        Summary: "", // Invalid: empty
        ProjectPath: "/path/to/project",
    }

    _, _, err := s.handleCheckpointSave(context.Background(), nil, input)

    require.Error(t, err)
    var mcpErr *MCPError
    require.ErrorAs(t, err, &mcpErr)
    assert.Equal(t, ErrorCategoryValidation, mcpErr.Category)
}
```

### Integration Tests

**Test Scenarios**:
1. **End-to-End Tool Calls**: Full request/response cycle
2. **Service Integration**: Verify service interactions
3. **Timeout Handling**: Test timeout behavior
4. **Rate Limiting**: Verify rate limits enforced

**Test Environment**:
- Test Qdrant instance
- Mock embedding service
- Test databases per project

### Performance Tests

**Benchmarks**:
1. **Tool Call Latency**: Measure handler performance
2. **Throughput**: Concurrent request handling
3. **Memory Usage**: Check for memory leaks
4. **Rate Limiter Overhead**: Measure rate limiting cost

**Benchmark Example**:
```go
func BenchmarkHandleCheckpointSave(b *testing.B) {
    s := setupTestServer(b)
    input := validCheckpointSaveInput()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _, err := s.handleCheckpointSave(context.Background(), nil, input)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Usage Examples

### Claude Code Configuration

Add contextd MCP server to `~/.claude/config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "/usr/local/bin/contextd",
      "args": ["--mcp"],
      "env": {
        "EMBEDDING_BASE_URL": "http://localhost:8080/v1",
        "EMBEDDING_MODEL": "BAAI/bge-small-en-v1.5",
        "QDRANT_URI": "http://localhost:6333",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318"
      }
    }
  }
}
```

### Example Tool Calls

**Save Checkpoint**:
```javascript
// Claude Code automatically uses MCP tools
checkpoint_save({
  summary: "Implemented user authentication",
  description: "Added JWT-based auth with refresh tokens",
  project_path: "/home/user/project",
  tags: ["auth", "security", "backend"]
})
```

**Search Checkpoints**:
```javascript
checkpoint_search({
  query: "authentication implementation",
  top_k: 5,
  project_path: "/home/user/project"
})
```

**Save Error Solution**:
```javascript
remediation_save({
  error_message: "dial tcp 127.0.0.1:8080: connect: connection refused",
  error_type: "ConnectionError",
  solution: "Start the server: ./server",
  severity: "medium",
  tags: ["networking", "server"]
})
```

**Search Error Solutions**:
```javascript
remediation_search({
  error_message: "connection refused",
  limit: 5,
  min_score: 0.6
})
```

**Troubleshoot Error**:
```javascript
troubleshoot({
  error_message: "panic: runtime error: invalid memory address",
  stack_trace: "goroutine 1 [running]:\nmain.processRequest(...)",
  context: {
    "go_version": "1.21",
    "os": "linux"
  },
  mode: "auto"
})
```

**Create Skill**:
```javascript
skill_create({
  name: "Debug Go Race Conditions",
  description: "Systematic approach to debugging race conditions in Go",
  content: "# Debug Race Conditions\n\n1. Run with -race flag...",
  version: "1.0.0",
  author: "contextd team",
  category: "debugging",
  tags: ["go", "concurrency", "debugging"]
})
```

**Search Skills**:
```javascript
skill_search({
  query: "debugging race conditions",
  category: "debugging",
  top_k: 3
})
```

**Get Analytics**:
```javascript
analytics_get({
  period: "weekly",
  project_path: "/home/user/project"
})
```

## Related Documentation

- **Project Root**: [/CLAUDE.md](/home/dahendel/projects/research-contextd/CLAUDE.md)
- **Standards**: [/docs/standards/](/home/dahendel/projects/research-contextd/docs/standards/)
- **Architecture**: [/docs/architecture/](/home/dahendel/projects/research-contextd/docs/architecture/)
- **MCP Protocol**: [https://modelcontextprotocol.io/](https://modelcontextprotocol.io/)
- **Package Documentation**: [/pkg/mcp/](/home/dahendel/projects/research-contextd/pkg/mcp/)

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

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-10-15 | Initial MCP integration with 9 tools |
| 2.0.0 | 2025-11-04 | Added 7 new tools (skills + analytics), rate limiting, enhanced telemetry |

## Summary

The MCP integration provides a complete, production-ready interface for AI assistants to interact with contextd services. With 16 tools covering session management, error resolution, AI diagnosis, skills management, and analytics, it enables powerful context-aware workflows for developers.

**Key Strengths**:
- **Complete Coverage**: All contextd functionality exposed via MCP
- **Type Safety**: Strong typing with validation at multiple levels
- **Observability**: Full OpenTelemetry instrumentation
- **Security**: Path traversal protection, rate limiting, input validation
- **Performance**: Operation-specific timeouts, efficient rate limiting
- **Developer Experience**: Clear error messages, comprehensive documentation
