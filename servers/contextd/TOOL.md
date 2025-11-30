# contextd

> Secure tool execution with secret scrubbing and session management.

## Endpoint

**Port**: `50051` (dual-protocol: gRPC + HTTP on same port)

| Protocol | Use Case | Connection |
|----------|----------|------------|
| **HTTP/REST** | Quick testing, curl, simple scripts | `http://localhost:50051/api/v1` |
| **gRPC** | Production, typed clients, streaming | `localhost:50051` |

## Available Tools

| Tool | gRPC Method | HTTP Endpoint | Purpose |
|------|-------------|---------------|---------|
| `safe_bash` | SafeExecService.Bash | POST /api/v1/safeexec/bash | Execute shell commands with scrubbing |
| `safe_read` | SafeExecService.Read | POST /api/v1/safeexec/read | Read files with path validation |
| `safe_write` | SafeExecService.Write | POST /api/v1/safeexec/write | Write files with path validation |
| `session_start` | SessionService.Start | POST /api/v1/session/start | Start new session |
| `session_end` | SessionService.End | POST /api/v1/session/end | End session |
| `ref_get` | RefService.GetContent | POST /api/v1/ref/content | Resolve content reference |

## Quick Start

### HTTP (Recommended for quick testing)

```bash
# Start session
curl -X POST http://localhost:50051/api/v1/session/start \
  -H "Content-Type: application/json" \
  -d '{"project_path": "/path/to/project"}'

# Execute command (use session_id from response)
curl -X POST http://localhost:50051/api/v1/safeexec/bash \
  -H "Content-Type: application/json" \
  -d '{"session_id": "sess_xxx", "cmd": "ls -la"}'

# Read file
curl -X POST http://localhost:50051/api/v1/safeexec/read \
  -H "Content-Type: application/json" \
  -d '{"session_id": "sess_xxx", "path": "/path/to/file.txt"}'
```

### gRPC (Recommended for production)

```python
import grpc
from contextd.v1 import safeexec_pb2, safeexec_pb2_grpc
from contextd.v1 import session_pb2, session_pb2_grpc

# Connect
channel = grpc.insecure_channel('localhost:50051')
session_stub = session_pb2_grpc.SessionServiceStub(channel)
exec_stub = safeexec_pb2_grpc.SafeExecServiceStub(channel)

# Start session
session = session_stub.Start(session_pb2.SessionStartRequest(
    project_path="/path/to/project"
))

# Execute command
response = exec_stub.Bash(safeexec_pb2.BashRequest(
    session_id=session.session_id,
    cmd="ls -la"
))
print(response.summary)
```

## Response Format

All responses include:
- `summary`: Human-readable result
- `*_preview`: First 500 chars (for quick inspection)
- `*_ref`: Reference ID for full content (use ref_get to resolve)
- `tokens_used`: Token estimate

## Security

- All output scrubbed for secrets (gitleaks)
- Path traversal prevented
- Command injection blocked
- Session-scoped operations
- Both protocols share same security layer

## Protocol Comparison

| Aspect | HTTP/REST | gRPC |
|--------|-----------|------|
| Setup | curl, requests | protoc + grpcio |
| Debugging | Easy (browser, curl) | grpcurl, reflection |
| Type Safety | JSON validation | Strongly typed |
| Best For | Testing, scripts | Production clients |

## Debugging

```bash
# Health check (HTTP)
curl http://localhost:50051/health

# gRPC reflection (list services)
grpcurl -plaintext localhost:50051 list

# gRPC reflection (describe service)
grpcurl -plaintext localhost:50051 describe contextd.v1.SafeExecService
```

**Ref**: `schema.json` for full request/response schemas
