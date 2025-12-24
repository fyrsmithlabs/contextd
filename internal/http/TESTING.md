# HTTP Server Testing Guide

Manual testing guide for the contextd HTTP API.

## Prerequisites

1. Start the server (once implemented in main.go):
   ```bash
   go run ./cmd/contextd
   ```

   Or for manual testing, create a test server:
   ```bash
   cat > /tmp/test-server.go <<'EOF'
   package main

   import (
       httpserver "github.com/fyrsmithlabs/contextd/internal/http"
       "github.com/fyrsmithlabs/contextd/internal/secrets"
       "go.uber.org/zap"
   )

   func main() {
       scrubber, _ := secrets.New(nil)
       logger, _ := zap.NewProduction()
       defer logger.Sync()

       cfg := &httpserver.Config{
           Host: "localhost",
           Port: 9090,
       }

       server, _ := httpserver.NewServer(scrubber, logger, cfg)
       server.Start()
   }
   EOF

   go run /tmp/test-server.go
   ```

## Test Cases

### 1. Health Check

```bash
curl -v http://localhost:9090/health
```

**Expected Response:**
```json
{"status":"ok"}
```

**Status Code:** 200 OK

---

### 2. Scrub AWS Key

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": "my api key is AKIAIOSFODNN7EXAMPLE"}'
```

**Expected Response:**
```json
{
  "content": "my api key is [REDACTED]",
  "findings_count": 1
}
```

**Status Code:** 200 OK

---

### 3. Scrub GitHub Token

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"}'
```

**Expected Response:**
```json
{
  "content": "token: [REDACTED]",
  "findings_count": 1
}
```

---

### 4. Scrub Multiple Secrets

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{
    "content": "AWS_KEY=AKIAIOSFODNN7EXAMPLE\nGITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
  }'
```

**Expected Response:**
```json
{
  "content": "AWS_KEY=[REDACTED]\nGITHUB_TOKEN=[REDACTED]",
  "findings_count": 2
}
```

---

### 5. Clean Content (No Secrets)

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": "This is just regular text with no secrets."}'
```

**Expected Response:**
```json
{
  "content": "This is just regular text with no secrets.",
  "findings_count": 0
}
```

---

### 6. Empty Content (Error Case)

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": ""}'
```

**Expected Response:**
```json
{
  "message": "content field is required"
}
```

**Status Code:** 400 Bad Request

---

### 7. Invalid JSON (Error Case)

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

**Expected Response:**
```json
{
  "message": "invalid request body"
}
```

**Status Code:** 400 Bad Request

---

### 8. Large Content

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d @- <<'EOF'
{
  "content": "This is a long document with lots of text.\nAnd somewhere in here is a secret: AKIAIOSFODNN7EXAMPLE\nBut the rest is just regular content that should pass through unchanged.\nMore lines...\nMore lines...\nMore lines..."
}
EOF
```

**Expected Response:**
Content should have the secret redacted but everything else intact.

---

## Performance Testing

### Measure Response Time

```bash
curl -X POST http://localhost:9090/api/v1/scrub \
  -H "Content-Type: application/json" \
  -d '{"content": "test: AKIAIOSFODNN7EXAMPLE"}' \
  -w "\nTime: %{time_total}s\n"
```

**Expected:** < 0.1 seconds for small content

### Load Testing (with hey)

```bash
# Install hey: go install github.com/rakyll/hey@latest

hey -n 1000 -c 10 \
  -m POST \
  -H "Content-Type: application/json" \
  -d '{"content": "test: AKIAIOSFODNN7EXAMPLE"}' \
  http://localhost:9090/api/v1/scrub
```

**Expected:** Low latency, high throughput

---

## Python Integration Test

```python
import requests

# Start a session
response = requests.post(
    "http://localhost:9090/api/v1/scrub",
    json={"content": "my key is AKIAIOSFODNN7EXAMPLE"}
)

print(f"Status: {response.status_code}")
print(f"Response: {response.json()}")
print(f"Secret redacted: {'AKIAIOSFODNN7EXAMPLE' not in response.json()['content']}")
```

---

## Claude Code Hook Integration

Example `.claude/hooks/post-tool.yaml`:

```yaml
# Post-tool hook to scrub secrets from tool output
url: http://localhost:9090/api/v1/scrub
method: POST
headers:
  Content-Type: application/json
body:
  content: "{{tool_output}}"
response_path: content
```

Test this by:

1. Save the hook configuration
2. Run a Claude Code command that might expose secrets
3. Verify the output is scrubbed

---

## Troubleshooting

### Server Not Responding

Check if server is running:
```bash
curl http://localhost:9090/health
```

Check if port is in use:
```bash
lsof -i :9090
```

### Secrets Not Being Detected

1. Verify scrubber configuration in server
2. Check logs for scrubbing results
3. Test with known secret patterns

### Performance Issues

1. Check server logs for slow requests
2. Monitor CPU/memory usage
3. Test with different content sizes

---

## Automated Testing

Run the full test suite:

```bash
# Unit tests
go test -v ./internal/http/...

# With coverage
go test -cover ./internal/http/...

# With race detection
go test -race ./internal/http/...
```
