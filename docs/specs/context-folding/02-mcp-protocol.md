# 2. MCP Protocol

[← Back to Architecture](01-architecture.md) | [Next: Process Rewards →](03-process-rewards.md)

---

## Overview

Contextd exposes branch/fold operations via MCP tools over HTTP. Every tool returns branch state metadata to guide LLM decisions.

**Transport**: HTTP (not SSE - SSE deprecated for MCP)
**Base URL**: `http://localhost:9090/mcp`
**Discovery**: `/mcp/tools/list` returns all available tools

---

## Core Tools

### `context_branch`

Create sub-trajectory for focused subtask.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "description": "Brief summary of subtask (max 200 chars)"
    },
    "prompt": {
      "type": "string",
      "description": "Detailed instruction for branch execution"
    },
    "project_path": {
      "type": "string",
      "description": "Absolute path to project directory"
    }
  },
  "required": ["description", "prompt", "project_path"]
}
```

**Response**:
```json
{
  "branch_id": "br_abc123",
  "session_id": "sess_xyz789",
  "parent_branch_id": null,
  "created_at": "2025-01-17T10:30:00Z",
  "branch_depth": 1,
  "context_state": {
    "active_branch_id": "br_abc123",
    "branch_depth": 1,
    "total_tokens": 5200,
    "main_thread_tokens": 5000,
    "current_branch_tokens": 200
  }
}
```

**NATS State Created**:
```
Subject: {owner_id}/{project_hash}/{session_id}/branches/br_abc123
Data: {
  "id": "br_abc123",
  "description": "Search API logs for auth errors",
  "prompt": "Use grep to find all auth-related errors...",
  "parent_id": null,
  "status": "active",
  "created_at": "2025-01-17T10:30:00Z",
  "tokens": 200
}
```

### `context_return`

Fold branch and return to parent with summary.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "message": {
      "type": "string",
      "description": "Summary of branch outcome"
    },
    "project_path": {
      "type": "string",
      "description": "Absolute path to project directory"
    },
    "branch_id": {
      "type": "string",
      "description": "Branch to fold (defaults to current active branch)"
    }
  },
  "required": ["message", "project_path"]
}
```

**Response**:
```json
{
  "folded_at": "2025-01-17T10:35:00Z",
  "branch_id": "br_abc123",
  "parent_branch_id": null,
  "summary": {
    "tokens_folded": 8500,
    "tokens_saved": 8300,
    "operations_count": 12,
    "secrets_scrubbed": 2,
    "summary_redacted": false
  },
  "context_state": {
    "active_branch_id": null,
    "branch_depth": 0,
    "total_tokens": 5200,
    "main_thread_tokens": 5200,
    "current_branch_tokens": 0
  },
  "context_health": {
    "warning": "none",
    "main_thread_usage": 0.16
  }
}
```

**NATS State Updated**:
```
Subject: {owner_id}/{project_hash}/{session_id}/branches/br_abc123
Data: {
  "id": "br_abc123",
  "status": "folded",
  "folded_at": "2025-01-17T10:35:00Z",
  "summary": "Found 3 auth error patterns in logs",
  "tokens_folded": 8500
}
```

**Storage**:
- Main collection: Folded summary
- Archive collection: Full branch trajectory

---

## Session State Tools

### `context_branch_status`

Get current branch state.

**Input**: `{"project_path": "/path/to/project"}`

**Response**:
```json
{
  "session_id": "sess_xyz789",
  "active_branch_id": "br_abc123",
  "branch_depth": 2,
  "branch_path": ["main", "br_001", "br_abc123"],
  "token_breakdown": {
    "main_thread": 5000,
    "br_001": 3000,
    "br_abc123": 1200,
    "total": 9200,
    "folded_total": 18500
  },
  "context_limit": 32768,
  "usage_percent": 28
}
```

### `context_list_branches`

List all branches for session.

**Input**: `{"project_path": "/path/to/project"}`

**Response**:
```json
{
  "branches": [
    {
      "id": "br_001",
      "description": "Search logs",
      "status": "folded",
      "tokens": 8500,
      "created_at": "2025-01-17T10:25:00Z",
      "folded_at": "2025-01-17T10:30:00Z"
    },
    {
      "id": "br_002",
      "description": "Test endpoint",
      "status": "active",
      "tokens": 3200,
      "created_at": "2025-01-17T10:31:00Z"
    }
  ],
  "total_branches": 2,
  "active_branches": 1,
  "folded_branches": 1
}
```

### `context_rollback`

Rollback to specific branch point (checkpoint integration).

**Input**:
```json
{
  "project_path": "/path/to/project",
  "branch_id": "br_001",
  "restore_state": true
}
```

**Response**:
```json
{
  "rolled_back_to": "br_001",
  "branches_discarded": ["br_002", "br_003"],
  "tokens_recovered": 15000,
  "context_state": {
    "active_branch_id": "br_001",
    "branch_depth": 1,
    "total_tokens": 8000
  }
}
```

---

## State Tracking Headers

Every MCP response includes `X-Context-State` header with branch metadata:

```
X-Context-State: {
  "session_id": "sess_xyz789",
  "active_branch_id": "br_abc123",
  "branch_depth": 2,
  "total_tokens": 9200,
  "folded_tokens": 18500,
  "context_usage": 0.28
}
```

LLM uses this metadata to decide when to branch/fold.

---

## NATS Subject Hierarchy

```
{owner_id}/{project_hash}/{session_id}/
  ├── branches/
  │   ├── br_001                    # Branch state
  │   ├── br_002
  │   └── br_003
  │
  ├── events/
  │   ├── branch.created            # Event log
  │   ├── branch.folded
  │   └── operation.completed
  │
  └── meta/
      └── session_state             # Current active branch, depth
```

**TTL**: Session state persists for 24 hours, then auto-cleanup.

---

## Error Handling

**Branch Not Found**:
```json
{
  "error": {
    "code": -32602,
    "message": "Branch not found: br_xyz",
    "data": {
      "branch_id": "br_xyz",
      "session_id": "sess_123"
    }
  }
}
```

**Context Limit Exceeded** (optional hard limit):
```json
{
  "error": {
    "code": -32001,
    "message": "Context limit exceeded: 35000/32768 tokens",
    "data": {
      "current_tokens": 35000,
      "context_limit": 32768,
      "suggestion": "Fold current branch before continuing"
    }
  }
}
```

**Invalid Branch State**:
```json
{
  "error": {
    "code": -32003,
    "message": "Cannot fold branch: branch is not active",
    "data": {
      "branch_id": "br_001",
      "current_status": "folded"
    }
  }
}
```

---

## Tool Discovery

**GET /mcp/tools/list** returns updated tool manifest including context-folding tools:

```json
{
  "tools": [
    {
      "name": "context_branch",
      "description": "Create sub-trajectory for focused subtask",
      "inputSchema": {...}
    },
    {
      "name": "context_return",
      "description": "Fold branch and return to parent with summary",
      "inputSchema": {...}
    },
    ...
  ]
}
```

---

[← Back to Architecture](01-architecture.md) | [Next: Process Rewards →](03-process-rewards.md)
