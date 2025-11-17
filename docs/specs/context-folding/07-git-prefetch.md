# 7. Git Pre-Fetch Integration

[← Back to Secret Scrubbing](06-secret-scrubbing.md) | [Next: Implementation Phases →](08-implementation-phases.md)

---

## Overview

Existing pre-fetch engine (deterministic, automatic) becomes **LLM-directed**. Pre-fetch analyzer generates preview, LLM decides: load to main thread, create branch, or skip.

**Existing Infrastructure**: `pkg/prefetch` with git event detection, worktree support, 3 deterministic rules.

---

## Updated Flow

```
Git Event (branch switch)
    ↓
Detector (fsnotify watcher)
    ↓
Preview Generator (analyze diff size, file count)
    ↓
NATS Event: prefetch.preview.{owner_id}.{project_hash}.{event_id}
    ↓
MCP Tool: prefetch_handle
    ↓
LLM Decision: main_thread | create_branch | skip
```

---

## Preview Event

**NATS Subject**: `prefetch.preview.{owner_id}.{project_hash}.{event_id}`

**Payload**:
```json
{
  "event_type": "branch_switch",
  "from_branch": "main",
  "to_branch": "feature/auth",
  "preview": {
    "files_changed": 15,
    "lines_added": 450,
    "lines_removed": 120,
    "affected_files": ["auth/middleware.go", "auth/jwt.go"],
    "estimated_tokens": 8500
  }
}
```

---

## MCP Tool: `prefetch_handle`

**Input**:
```json
{
  "event_id": "evt_abc123",
  "decision": "create_branch | main_thread | skip"
}
```

**Response (create_branch)**:
```json
{
  "decision": "create_branch",
  "branch_id": "br_prefetch_abc",
  "summary": "Loaded feature/auth changes in isolated context",
  "tokens_loaded": 8500
}
```

---

## Multi-Tenant Routing

**Git Hook Integration**:
```bash
#!/bin/bash
# .git/hooks/post-checkout
PROJECT_PATH=$(git rev-parse --show-toplevel)
PROJECT_HASH=$(echo -n "$PROJECT_PATH" | sha256sum | cut -c1-8)
curl -X POST http://localhost:9090/internal/git-event \
  -H "X-Owner-ID: $OWNER_ID" \
  -H "X-Project-Hash: $PROJECT_HASH" \
  -d '{"event":"branch_switch"}'
```

**Worktree Isolation**:
- Different worktrees = different `PROJECT_PATH` → different `project_hash`
- Each gets separate pre-fetch events, separate NATS topics

---

[← Back to Secret Scrubbing](06-secret-scrubbing.md) | [Next: Implementation Phases →](08-implementation-phases.md)
