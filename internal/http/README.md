# internal/http

HTTP API server for contextd, providing secret scrubbing endpoints for Claude Code hooks.

## Overview

This package implements an Echo-based HTTP server that exposes the secret scrubbing functionality via REST endpoints. It's designed to be called by Claude Code hooks to process tool output and redact secrets before they enter the context.

## Features

- **POST /api/v1/scrub** - Scrub secrets from text content
- **GET /health** - Health check endpoint
- Request ID tracking
- Request/response logging
- Panic recovery middleware
- Graceful shutdown support

## API Reference

### POST /api/v1/scrub

Scrubs secrets from the provided content using the gitleaks-based scrubber.

**Request:**
```json
{
  "content": "my api key is sk-abc123..."
}
```

**Response:**
```json
{
  "content": "my api key is [REDACTED]...",
  "findings_count": 1
}
```

**Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid request body or missing content field
- `500 Internal Server Error` - Server error

### GET /health

Simple health check endpoint.

**Response:**
```json
{
  "status": "ok"
}
```

**Status Codes:**
- `200 OK` - Server is healthy

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "time"

    httpserver "github.com/fyrsmithlabs/contextd/internal/http"
    "github.com/fyrsmithlabs/contextd/internal/secrets"
    "go.uber.org/zap"
)

func main() {
    // Create scrubber
    scrubber, err := secrets.New(nil)
    if err != nil {
        panic(err)
    }

    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Configure server
    cfg := &httpserver.Config{
        Host: "localhost",
        Port: 9090,
    }

    // Create server
    server, err := httpserver.NewServer(scrubber, logger, cfg)
    if err != nil {
        panic(err)
    }

    // Start server
    if err := server.Start(); err != nil {
        logger.Fatal("server error", zap.Error(err))
    }
}
```

### With Graceful Shutdown

```go
// Start server in background
go func() {
    if err := server.Start(); err != nil {
        logger.Error("server error", zap.Error(err))
    }
}()

// Wait for interrupt signal
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

// Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    logger.Error("shutdown error", zap.Error(err))
}
```

## Configuration

The server accepts the following configuration:

```go
type Config struct {
    Host string  // Server host (default: "localhost")
    Port int     // Server port (default: 9090)
}
```

## Testing

Run the test suite:

```bash
go test ./internal/http/...
```

Check test coverage:

```bash
go test -coverprofile=cover.out ./internal/http/...
go tool cover -html=cover.out
```

Current coverage: **100%**

## Integration with Claude Code Hooks

This HTTP server is designed to be called by Claude Code hooks. Example hook configuration:

```yaml
# .claude/hooks/post-tool.yaml
url: http://localhost:9090/api/v1/scrub
method: POST
body:
  content: "{{tool_output}}"
```

When a tool executes, Claude Code will:
1. Execute the tool and capture output
2. Send output to the scrub endpoint
3. Receive scrubbed content
4. Use scrubbed content in the context

## Dependencies

- **Echo v4**: HTTP framework
- **Zap**: Structured logging
- **gitleaks**: Secret detection (via internal/secrets)

## Middleware

The server includes the following middleware:

1. **Recovery**: Recovers from panics and returns 500 errors
2. **RequestID**: Adds unique request IDs to all responses
3. **Logging**: Logs all HTTP requests with duration and status

## Performance

The server is optimized for low latency:

- Request processing: < 10ms (excluding scrubbing time)
- Scrubbing performance: < 100ms for 1KB content
- Concurrent request handling via Echo's HTTP/2 support

## Error Handling

The server follows REST error handling conventions:

- `400 Bad Request` - Client errors (invalid JSON, missing fields)
- `500 Internal Server Error` - Server errors (logged for debugging)

All errors include a JSON body with a `message` field.

## Logging

The server logs all requests with:

- HTTP method
- Request URI
- Status code
- Duration
- Request ID

Example log output:

```json
{
  "level": "info",
  "ts": "2025-11-30T14:27:26.793Z",
  "msg": "http request",
  "method": "POST",
  "uri": "/api/v1/scrub",
  "status": 200,
  "duration": 0.012345,
  "request_id": "abc123def456"
}
```

## Security

- **Localhost only**: Server binds to localhost by default
- **No authentication**: Currently designed for localhost-only access
- **Secret scrubbing**: All content is scrubbed before returning
- **Input validation**: Request bodies are validated

## Future Enhancements

Potential future additions (not yet implemented):

- Authentication/authorization
- Rate limiting
- HTTPS/TLS support
- Additional endpoints (batch scrubbing, config updates)
- Metrics endpoint (Prometheus-compatible)
