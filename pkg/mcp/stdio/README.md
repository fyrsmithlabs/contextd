# pkg/mcp/stdio - stdio MCP Server

**Package**: `github.com/fyrsmithlabs/contextd/pkg/mcp/stdio`
**Purpose**: stdio MCP transport server for Claude Code integration

---

## Overview

This package implements the stdio MCP server using the official MCP Go SDK. It provides native Claude Code integration via stdin/stdout transport.

**Key features**:
- 23 MCP tools (full capability set)
- Real-time progress notifications
- HTTP daemon polling for async operations
- Graceful shutdown handling
- Comprehensive error handling

---

## Architecture

### Component Structure

```
pkg/mcp/stdio/
├── server.go           # Main stdio server implementation
├── tools.go            # Tool registration and handlers
├── progress.go         # Progress monitoring and polling
├── types.go            # Request/response types
├── validation.go       # Input validation
├── errors.go           # Error handling
└── server_test.go      # Comprehensive test suite
```

### Server Lifecycle

```
1. Claude Code spawns: contextd --mcp
2. Server initializes:
   - Load config
   - Create HTTP client
   - Connect to HTTP daemon
   - Register 23 tools
3. Server runs: Listen on stdin, write to stdout
4. Server shuts down: SIGTERM/SIGKILL sequence
```

---

## Usage

### Basic Server

```go
package main

import (
    "context"
    "log"

    "github.com/fyrsmithlabs/contextd/pkg/mcp/stdio"
)

func main() {
    // Create stdio server
    server, err := stdio.NewServer(&stdio.Config{
        HTTPDaemonURL: "http://localhost:9090",
        ProgressPollInterval: 500 * time.Millisecond,
        OperationTimeout: 5 * time.Minute,
    })
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Run server on stdio transport
    if err := server.Run(context.Background()); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### With Services

```go
// Create services
services := &stdio.Services{
    Checkpoint: checkpointService,
    Remediation: remediationService,
    Troubleshoot: troubleshootService,
    Skills: skillsService,
    Repository: repositoryService,
    VectorStore: vectorStoreService,
}

// Create server with services
server, err := stdio.NewServerWithServices(&stdio.Config{
    HTTPDaemonURL: "http://localhost:9090",
}, services)
```

---

## Tool Handlers

### Synchronous Tools

**Tools that return immediately**:

```go
func (s *Server) handleCheckpointSearch(ctx context.Context, session *mcpsdk.ServerSession, req *mcpsdk.CallToolRequest, args CheckpointSearchArgs) (*mcpsdk.CallToolResult, error) {
    // Validate input
    if err := args.Validate(); err != nil {
        return nil, err
    }

    // Call service directly
    results, err := s.services.Checkpoint.Search(ctx, args.Query, args.Limit)
    if err != nil {
        return nil, err
    }

    // Return results immediately
    return &mcpsdk.CallToolResult{
        Content: []mcpsdk.Content{
            &mcpsdk.TextContent{Text: formatResults(results)},
        },
    }, nil
}
```

### Asynchronous Tools

**Tools with progress notifications**:

```go
func (s *Server) handleCheckpointSave(ctx context.Context, session *mcpsdk.ServerSession, req *mcpsdk.CallToolRequest, args CheckpointSaveArgs) (*mcpsdk.CallToolResult, error) {
    // Extract progress token
    progressToken := req.Params.GetProgressToken()

    // Forward to HTTP daemon
    resp, err := s.httpClient.Post("http://localhost:9090/mcp/checkpoint/save",
        "application/json",
        marshalJSON(args))
    if err != nil {
        return nil, err
    }

    var result struct {
        OperationID string `json:"operation_id"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    // Start background poller
    go s.pollOperationStatus(ctx, session, result.OperationID, progressToken)

    // Return immediately
    return &mcpsdk.CallToolResult{
        Content: []mcpsdk.Content{
            &mcpsdk.TextContent{Text: "Checkpoint save in progress"},
        },
    }, nil
}
```

---

## Progress Monitoring

### Polling Implementation

```go
// progress.go
func (s *Server) pollOperationStatus(ctx context.Context, session *mcpsdk.ServerSession, operationID string, progressToken any) {
    ticker := time.NewTicker(s.config.ProgressPollInterval)
    defer ticker.Stop()

    timeout := time.After(s.config.OperationTimeout)

    for {
        select {
        case <-ctx.Done():
            return
        case <-timeout:
            s.sendProgressNotification(session, progressToken, 0, 100, "Operation timeout")
            return
        case <-ticker.C:
            status, err := s.queryOperationStatus(operationID)
            if err != nil {
                continue // Retry on next tick
            }

            // Send progress notification
            s.sendProgressNotification(session, progressToken, status.Progress, 100, status.Message)

            // Stop when complete
            if status.Done {
                return
            }
        }
    }
}

func (s *Server) sendProgressNotification(session *mcpsdk.ServerSession, progressToken any, progress, total float64, message string) {
    session.NotifyProgress(context.Background(), &mcpsdk.ProgressNotificationParams{
        ProgressToken: progressToken,
        Progress: progress,
        Total: total,
        Message: message,
    })
}

func (s *Server) queryOperationStatus(operationID string) (*OperationStatus, error) {
    resp, err := s.httpClient.Post("http://localhost:9090/mcp/status",
        "application/json",
        marshalJSON(map[string]string{"operation_id": operationID}))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var status OperationStatus
    if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
        return nil, err
    }

    return &status, nil
}
```

---

## Input Validation

### Validation Pattern

```go
// validation.go
type CheckpointSaveArgs struct {
    Content     string            `json:"content"`
    ProjectPath string            `json:"project_path"`
    Metadata    map[string]string `json:"metadata"`
}

func (a *CheckpointSaveArgs) Validate() error {
    if a.Content == "" {
        return errors.New("content is required")
    }

    if a.ProjectPath == "" {
        return errors.New("project_path is required")
    }

    // Validate project path (no path traversal)
    if strings.Contains(a.ProjectPath, "..") {
        return errors.New("invalid project_path: path traversal detected")
    }

    // Validate content length
    if len(a.Content) > 1000000 { // 1MB
        return errors.New("content exceeds 1MB limit")
    }

    return nil
}
```

**All tool arguments MUST implement `Validate()` method.**

---

## Error Handling

### Error Types

```go
// errors.go
var (
    ErrInvalidInput = errors.New("invalid input")
    ErrHTTPDaemonUnreachable = errors.New("HTTP daemon unreachable")
    ErrOperationTimeout = errors.New("operation timeout")
    ErrOperationFailed = errors.New("operation failed")
)

// Wrap errors with context
func (s *Server) handleCheckpointSave(...) error {
    if err := s.httpClient.Post(...); err != nil {
        return fmt.Errorf("failed to forward request to HTTP daemon: %w", err)
    }
}
```

### Error Responses

```go
// Return MCP-compliant error response
if err := args.Validate(); err != nil {
    return nil, &mcpsdk.Error{
        Code: mcpsdk.ErrorCodeInvalidParams,
        Message: err.Error(),
    }
}
```

---

## Testing

### Unit Tests

```go
func TestServer_HandleCheckpointSave(t *testing.T) {
    // Mock HTTP daemon
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{
            "operation_id": "op-123",
            "status": "pending",
        })
    }))
    defer mockServer.Close()

    // Create test server
    server := &Server{
        httpClient: &http.Client{},
        config: &Config{
            HTTPDaemonURL: mockServer.URL,
        },
    }

    // Test tool handler
    args := CheckpointSaveArgs{
        Content: "test checkpoint",
        ProjectPath: "/tmp/test",
    }

    result, err := server.handleCheckpointSave(context.Background(), nil, nil, args)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if result == nil {
        t.Fatal("Expected result, got nil")
    }
}
```

### Integration Tests

```go
func TestStdioServer_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Start HTTP daemon
    daemon := startHTTPDaemon(t)
    defer daemon.Stop()

    // Create stdio server
    server, err := stdio.NewServer(&stdio.Config{
        HTTPDaemonURL: daemon.URL,
    })
    if err != nil {
        t.Fatalf("Failed to create server: %v", err)
    }

    // Test full stdio cycle
    // (stdin → server → HTTP daemon → stdout)
}
```

**Run tests**:

```bash
# Unit tests
go test ./pkg/mcp/stdio/

# Integration tests
go test -v ./pkg/mcp/stdio/ -run Integration

# Coverage
go test -coverprofile=coverage.out ./pkg/mcp/stdio/
go tool cover -func=coverage.out
```

---

## Configuration

### Config Structure

```go
type Config struct {
    // HTTP daemon URL (required)
    HTTPDaemonURL string

    // Progress polling interval (default: 500ms)
    ProgressPollInterval time.Duration

    // Operation timeout (default: 5m)
    OperationTimeout time.Duration

    // Log level (default: info)
    LogLevel string
}
```

### Environment Variables

```bash
# HTTP daemon URL
export CONTEXTD_HTTP_URL=http://localhost:9090

# Progress polling
export MCP_PROGRESS_POLL_INTERVAL=500ms

# Operation timeout
export MCP_OPERATION_TIMEOUT=5m

# Log level
export CONTEXTD_LOG_LEVEL=debug
```

---

## Best Practices

### 1. Always Validate Input

```go
func (s *Server) handleTool(..., args ToolArgs) error {
    // ALWAYS validate first
    if err := args.Validate(); err != nil {
        return &mcpsdk.Error{
            Code: mcpsdk.ErrorCodeInvalidParams,
            Message: err.Error(),
        }
    }

    // Then process
    ...
}
```

### 2. Use Progress Notifications

```go
// For any operation >1s, use progress notifications
if longRunningOperation {
    progressToken := req.Params.GetProgressToken()
    go s.pollOperationStatus(ctx, session, operationID, progressToken)
}
```

### 3. Handle Context Cancellation

```go
func (s *Server) pollOperationStatus(ctx context.Context, ...) {
    for {
        select {
        case <-ctx.Done():
            return // Stop polling immediately
        case <-ticker.C:
            // Continue polling
        }
    }
}
```

### 4. Wrap Errors with Context

```go
if err := s.httpClient.Post(...); err != nil {
    return fmt.Errorf("failed to forward request: %w", err)
}
```

---

## Performance

### Metrics

- **Tool call latency**: <50ms (synchronous tools)
- **Progress poll overhead**: ~2ms per poll
- **Memory usage**: ~50MB baseline + ~1KB per active operation
- **CPU usage**: <5% (idle), <20% (active polling)

### Optimization

1. **Batch operations** when possible
2. **Adjust polling interval** based on use case
3. **Use connection pooling** for HTTP client
4. **Cache operation status** (TTL: 1s)

---

## Related Documentation

- **User guides**:
  - [STDIO-MCP-SETUP.md](../../../docs/guides/STDIO-MCP-SETUP.md)
  - [STDIO-MCP-MIGRATION.md](../../../docs/guides/STDIO-MCP-MIGRATION.md)
- **Architecture**: [stdio-transport.md](../../../docs/standards/architecture/stdio-transport.md)
- **Specification**: [SPEC.md](../../../docs/specs/stdio-mcp-integration/SPEC.md)
