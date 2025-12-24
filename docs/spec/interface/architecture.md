# contextd Interface Architecture

**Version**: 2.1.0
**Status**: Draft
**Date**: 2025-11-26

---

## Overview

Claude discovers tools via TOOL.md, then chooses between gRPC or HTTP/REST to call contextd. Both protocols served on the same port using cmux multiplexing.

**Design Principles** (from [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use)):
- **Lazy discovery**: Load tool schemas on-demand, not upfront
- **Programmatic calling**: Claude writes code → gRPC/HTTP, not sequential tool calls
- **Token efficiency**: Only final results enter context
- **Examples over schemas**: Realistic examples improve accuracy 72% → 90%
- **Dual-protocol**: gRPC for typed clients, HTTP for simplicity

```
┌─────────────────────────────────────────────────────────────────────┐
│ Claude                                                              │
│  1. Read TOOL.md → discover available tools (lazy)                  │
│  2. Read schema.json → get specific tool input/output + examples    │
│  3. Choose invocation method:                                       │
│     Option A: Write Python → gRPC call (typed, efficient)           │
│     Option B: Use curl/requests → HTTP/REST (simpler)               │
│  4. Only final output returns to context                            │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ gRPC OR HTTP/REST (same port :50051)
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ contextd Server (Dual-Protocol via cmux)                            │
│  ├── cmux Multiplexer                                               │
│  │   ├── gRPC Server (HTTP/2 + application/grpc)                    │
│  │   └── Echo HTTP Server (HTTP/1.1 REST API)                       │
│  ├── Shared Service Layer                                           │
│  │   ├── SafeExecService, SessionService, RefService                │
│  │   └── MemoryService, CheckpointService, PolicyService            │
│  ├── Secret Scrubber (gitleaks)                                     │
│  ├── Process Isolation (seccomp + namespaces)                       │
│  ├── Cache Layer (in-memory refs)                                   │
│  ├── Qdrant Client (memories, checkpoints)                          │
│  └── Audit Logger                                                   │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ Token-optimized response
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Code execution result → Claude Context (summary only)               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
contextd/
├── docs/
│   └── tools/
│       ├── TOOL.md            # Tool catalog (Claude reads this first)
│       └── schema.json        # All tool schemas + examples
├── api/
│   └── proto/
│       └── contextd.proto     # gRPC service definitions
└── cmd/contextd/              # Server binary
```

---

## Tool Categories

| Category | Service | Methods | Purpose |
|----------|---------|---------|---------|
| **SafeExec** | SafeExecService | Bash, Read, Write | Filesystem/shell with scrubbing |
| **Memory** | MemoryService | Search, Store, Feedback, Get | ReasoningBank |
| **Checkpoint** | CheckpointService | Save, List, Resume | Session persistence |
| **Policy** | PolicyService | Check, List, Get | Governance |
| **Skill** | SkillService | CRUD | Skill management |
| **Agent** | AgentService | CRUD | Agent configs |
| **Remediation** | RemediationService | Search, Record | Error patterns |

---

## Invocation Patterns

### Option A: gRPC (Typed, Efficient)

**Use when**: You want type safety, streaming, or minimal overhead.

```python
import grpc
from contextd.v1 import safeexec_pb2, safeexec_pb2_grpc

channel = grpc.insecure_channel('localhost:50051')
stub = safeexec_pb2_grpc.SafeExecServiceStub(channel)

response = stub.Bash(safeexec_pb2.BashRequest(
    cmd="ls -la",
    session_id="sess_abc123"
))
print(f"Exit: {response.exit_code}, Summary: {response.summary}")
```

### Option B: HTTP/REST (Simple, Universal)

**Use when**: You prefer curl, requests, or don't have gRPC client.

```bash
# Start a session
SESSION=$(curl -s -X POST http://localhost:50051/api/v1/session/start \
  -H "Content-Type: application/json" \
  -d '{"project_path": "/path/to/project"}' | jq -r '.session_id')

# Execute command
curl -X POST http://localhost:50051/api/v1/safeexec/bash \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"$SESSION\", \"cmd\": \"ls -la\"}"
```

```python
import requests

# Start session
resp = requests.post("http://localhost:50051/api/v1/session/start",
    json={"project_path": "/path/to/project"})
session_id = resp.json()["session_id"]

# Execute command
resp = requests.post("http://localhost:50051/api/v1/safeexec/bash",
    json={"session_id": session_id, "cmd": "ls -la"})
print(resp.json()["summary"])
```

### Protocol Comparison

| Aspect | gRPC | HTTP/REST |
|--------|------|-----------|
| Client setup | Requires protoc, grpcio | Any HTTP client |
| Type safety | Strongly typed | JSON schema validation |
| Performance | HTTP/2 multiplexing | HTTP/1.1 |
| Streaming | Bidirectional | Not supported |
| Debugging | grpcurl, reflection | curl, browser |
| Recommended for | Production clients | Quick scripts, testing |

---

## Schema Format

**schema.json** - Object keyed by service.method, includes examples:
```json
{
  "SafeExec.Bash": {
    "service": "SafeExecService",
    "method": "Bash",
    "description": "Execute shell command with secret scrubbing",
    "input": {
      "type": "object",
      "properties": {
        "cmd": {"type": "string", "description": "Shell command to execute"},
        "timeout": {"type": "integer", "default": 30, "description": "Timeout in seconds"},
        "session_id": {"type": "string", "description": "Session identifier"},
        "working_dir": {"type": "string", "description": "Working directory (optional)"}
      },
      "required": ["cmd", "session_id"]
    },
    "output": {
      "type": "object",
      "properties": {
        "summary": {"type": "string", "description": "Human-readable execution summary"},
        "stdout_preview": {"type": "string", "description": "First 500 chars of stdout"},
        "stderr_preview": {"type": "string", "description": "First 500 chars of stderr"},
        "stdout_ref": {"type": "string", "description": "Ref ID for full stdout (use RefService.GetContent)"},
        "stderr_ref": {"type": "string", "description": "Ref ID for full stderr"},
        "exit_code": {"type": "integer", "description": "Process exit code (0 = success)"},
        "tokens_used": {"type": "integer", "description": "Estimated tokens in response"}
      }
    },
    "examples": [
      {
        "description": "Simple command",
        "input": {"cmd": "ls -la /tmp", "session_id": "sess_abc123"},
        "output": {"summary": "Executed: ls -la /tmp (exit 0, 15 lines)", "exit_code": 0, "stdout_ref": "ref_xyz789", "tokens_used": 45}
      },
      {
        "description": "Command with extended timeout",
        "input": {"cmd": "find / -name '*.log' 2>/dev/null", "session_id": "sess_abc123", "timeout": 120},
        "output": {"summary": "Executed: find (exit 0, 2847 lines)", "exit_code": 0, "stdout_ref": "ref_def456", "tokens_used": 52}
      },
      {
        "description": "Failed command",
        "input": {"cmd": "cat /nonexistent", "session_id": "sess_abc123"},
        "output": {"summary": "Executed: cat (exit 1)", "exit_code": 1, "stderr_preview": "cat: /nonexistent: No such file or directory", "tokens_used": 38}
      }
    ]
  }
}
```

---

## TOOL.md Format

```markdown
# contextd Tools

**Endpoint**: `localhost:50051` (dual-protocol: gRPC + HTTP)

## Quick Start

### HTTP (Recommended for quick testing)
```bash
curl -X POST http://localhost:50051/api/v1/session/start \
  -H "Content-Type: application/json" \
  -d '{"project_path": "/path/to/project"}'
```

### gRPC (Recommended for production)
```python
import grpc
from contextd.v1 import safeexec_pb2, safeexec_pb2_grpc
channel = grpc.insecure_channel('localhost:50051')
```

## Tool Catalog

| Tool | gRPC | HTTP | Purpose |
|------|------|------|---------|
| Bash | SafeExecService.Bash | POST /api/v1/safeexec/bash | Run shell commands |
| Read | SafeExecService.Read | POST /api/v1/safeexec/read | Read file contents |
| Write | SafeExecService.Write | POST /api/v1/safeexec/write | Write file contents |
| Session Start | SessionService.Start | POST /api/v1/session/start | Begin session |
| Session End | SessionService.End | POST /api/v1/session/end | End session |
| Ref Get | RefService.GetContent | POST /api/v1/ref/content | Resolve content ref |

## Response Pattern

All responses include:
- `summary`: Human-readable result (always present)
- `*_preview`: First 500 chars (for quick inspection)
- `*_ref`: Reference ID for full content (use RefService.GetContent)
- `tokens_used`: Estimated token cost
```

---

## gRPC Services

@./api/contextd.proto

| Service | Methods | Purpose |
|---------|---------|---------|
| `SafeExecService` | Bash, Read, Write | Scrubbed filesystem/shell |
| `MemoryService` | Search, Store, Feedback, Get | ReasoningBank |
| `CheckpointService` | Save, List, Resume | Session persistence |
| `PolicyService` | Check, List, Get | Governance |
| `SkillService` | CRUD | Skill management |
| `AgentService` | CRUD | Agent configs |
| `RemediationService` | Search, Record | Error patterns |
| `RefService` | GetContent | Resolve content refs |
| `SessionService` | Start, End | Session lifecycle |

---

## Token Optimization

**Tiered responses** - only load what you need:

| Tier | Field | When to Use |
|------|-------|-------------|
| 1 | `summary` | Always returned, human-readable |
| 2 | `*_preview` | Quick inspection (first 500 chars) |
| 3 | `*_ref` | Full content via RefService.GetContent |

**Example flow**:
```python
# Tier 1: Just the summary
response = stub.Bash(BashRequest(cmd="ls -la", session_id="sess_123"))
print(response.summary)  # "Executed: ls -la (exit 0, 150 lines)"

# Tier 2: Need preview
if response.exit_code != 0:
    print(response.stderr_preview)

# Tier 3: Need full output
if need_full_output:
    full = ref_stub.GetContent(RefGetContentRequest(ref_id=response.stdout_ref))
    print(full.content)
```

---

## Session Management

- `session_id` required on all requests
- Server creates session on first use
- Sessions scope: checkpoints, memory queries, audit logs
- Sessions expire after configurable TTL

---

## Security

| Layer | Protection |
|-------|------------|
| **Transport** | Localhost dual-protocol (Phase 1), mTLS (future) |
| **Scrubbing** | gitleaks on all output (both protocols) |
| **Isolation** | seccomp + namespaces per execution |
| **Audit** | All operations logged (both protocols) |

---

## Maintenance

**Update when:**
- New gRPC services added
- Schema format changes
- Security model changes

**Keep**: Scannable, examples current, @imports for proto
