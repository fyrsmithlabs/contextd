# HTTP Server Usage Example

This document provides a complete example of how to start and use the contextd HTTP server.

## Standalone Server Example

Create a file `cmd/http-server/main.go`:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    httpserver "github.com/fyrsmithlabs/contextd/internal/http"
    "github.com/fyrsmithlabs/contextd/internal/secrets"
    "go.uber.org/zap"
)

func main() {
    // Create logger
    logger, err := zap.NewProduction()
    if err != nil {
        panic(err)
    }
    defer logger.Sync()

    // Create scrubber with default configuration
    scrubber, err := secrets.New(nil)
    if err != nil {
        logger.Fatal("failed to create scrubber", zap.Error(err))
    }

    // Configure server
    cfg := &httpserver.Config{
        Host: "localhost",
        Port: 9090,
    }

    // Create server
    server, err := httpserver.NewServer(scrubber, logger, cfg)
    if err != nil {
        logger.Fatal("failed to create server", zap.Error(err))
    }

    // Start server in background
    go func() {
        logger.Info("starting server", zap.String("addr", "localhost:9090"))
        if err := server.Start(); err != nil {
            logger.Error("server error", zap.Error(err))
        }
    }()

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    logger.Info("received shutdown signal")

    // Graceful shutdown with 10 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Error("shutdown error", zap.Error(err))
        os.Exit(1)
    }

    logger.Info("server stopped gracefully")
}
```

## Running the Server

```bash
# Build and run
go run cmd/http-server/main.go

# Or build binary
go build -o http-server cmd/http-server/main.go
./http-server
```

## Testing the Server

Once the server is running, test it with curl:

```bash
# Health check
curl http://localhost:9090/health

# Scrub secrets
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": "my api key is AKIAIOSFODNN7EXAMPLE"}'
```

**Expected output:**
```json
{
  "content": "my api key is [REDACTED]",
  "findings_count": 1
}
```

## Environment Variables

The server can be configured via environment variables (when integrated with main config):

```bash
# Server configuration
export HTTP_SERVER_HOST=localhost
export HTTP_SERVER_PORT=9090

# Scrubber configuration
export SCRUBBER_ENABLED=true
export SCRUBBER_REDACTION_STRING="[REDACTED]"

# Run server
go run cmd/http-server/main.go
```

## Docker Example

Create a `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o http-server cmd/http-server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/http-server .

EXPOSE 9090
CMD ["./http-server"]
```

Build and run:

```bash
# Build
docker build -t contextd-http .

# Run
docker run -p 9090:9090 contextd-http
```

## Integration with contextd Main

When integrated into the main contextd server (`cmd/contextd/main.go`), the HTTP server will be started alongside the MCP server:

```go
package main

import (
    "context"

    httpserver "github.com/fyrsmithlabs/contextd/internal/http"
    "github.com/fyrsmithlabs/contextd/internal/mcp"
    "github.com/fyrsmithlabs/contextd/internal/secrets"
    // ... other imports
)

func main() {
    // ... initialization code ...

    // Create shared scrubber
    scrubber, _ := secrets.New(secretsConfig)

    // Start HTTP server
    httpCfg := &httpserver.Config{
        Host: cfg.Server.Host,
        Port: cfg.Server.Port,
    }

    httpSrv, _ := httpserver.NewServer(scrubber, logger, httpCfg)

    go func() {
        if err := httpSrv.Start(); err != nil {
            logger.Error("http server error", zap.Error(err))
        }
    }()

    // Start MCP server
    mcpSrv := mcp.NewServer(...)
    mcpSrv.Run()

    // Shutdown both servers on exit
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        httpSrv.Shutdown(ctx)
    }()
}
```

## Client Examples

### Python

```python
import requests

class ContextdClient:
    def __init__(self, base_url="http://localhost:9090"):
        self.base_url = base_url

    def health(self):
        """Check server health"""
        resp = requests.get(f"{self.base_url}/health")
        return resp.json()

    def scrub(self, content):
        """Scrub secrets from content"""
        resp = requests.post(
            f"{self.base_url}/api/v1/scrub",
            json={"content": content}
        )
        resp.raise_for_status()
        return resp.json()

# Usage
client = ContextdClient()

# Health check
print(client.health())  # {"status": "ok"}

# Scrub secrets
result = client.scrub("my key is AKIAIOSFODNN7EXAMPLE")
print(result["content"])  # "my key is [REDACTED]"
print(f"Found {result['findings_count']} secrets")
```

### JavaScript/Node.js

```javascript
const axios = require('axios');

class ContextdClient {
  constructor(baseURL = 'http://localhost:9090') {
    this.client = axios.create({ baseURL });
  }

  async health() {
    const { data } = await this.client.get('/health');
    return data;
  }

  async scrub(content) {
    const { data } = await this.client.post('/api/v1/scrub', { content });
    return data;
  }
}

// Usage
const client = new ContextdClient();

(async () => {
  // Health check
  console.log(await client.health());

  // Scrub secrets
  const result = await client.scrub('my key is AKIAIOSFODNN7EXAMPLE');
  console.log(result.content);
  console.log(`Found ${result.findings_count} secrets`);
})();
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type ScrubRequest struct {
    Content string `json:"content"`
}

type ScrubResponse struct {
    Content       string `json:"content"`
    FindingsCount int    `json:"findings_count"`
}

func scrub(content string) (*ScrubResponse, error) {
    reqBody, _ := json.Marshal(ScrubRequest{Content: content})

    resp, err := http.Post(
        "http://localhost:9090/api/v1/scrub",
        "application/json",
        bytes.NewReader(reqBody),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result ScrubResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

func main() {
    result, _ := scrub("my key is AKIAIOSFODNN7EXAMPLE")
    fmt.Println(result.Content)
    fmt.Printf("Found %d secrets\n", result.FindingsCount)
}
```

## Monitoring and Observability

The server logs all requests with structured logging:

```json
{
  "level": "info",
  "ts": "2025-11-30T14:27:26.793Z",
  "caller": "http/server.go:55",
  "msg": "http request",
  "method": "POST",
  "uri": "/api/v1/scrub",
  "status": 200,
  "duration": 0.012345,
  "request_id": "abc123def456"
}
```

Monitor these logs for:
- Request volume (`msg: "http request"`)
- Error rates (`status: 400`, `status: 500`)
- Latency (`duration` field)
- Request IDs for tracing
