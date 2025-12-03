# MCP Tools API Reference

ContextD exposes its functionality through the Model Context Protocol (MCP). This document provides complete reference documentation for all available tools.

---

## Overview

ContextD provides 10 MCP tools organized into four categories:

| Category | Tools | Purpose |
|----------|-------|---------|
| **Memory** | `memory_search`, `memory_record`, `memory_feedback` | Cross-session learning and strategies |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Context persistence and recovery |
| **Remediation** | `remediation_search`, `remediation_record` | Error pattern tracking and fixes |
| **Utility** | `repository_index`, `troubleshoot_diagnose` | Code indexing and diagnostics |

---

## Memory Tools

Memory tools implement the ReasoningBank system for cross-session learning. Memories are strategies, insights, or learnings that proved useful (or harmful) and should be remembered for future sessions.

### memory_search

Search for relevant memories from past sessions.

**Use Case**: At the start of a task, search for relevant strategies that worked before.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier (typically the repository path) |
| `query` | string | Yes | Natural language search query |
| `limit` | integer | No | Maximum results to return (default: 5) |

#### Response

```json
{
  "memories": [
    {
      "id": "mem_abc123",
      "title": "Use table-driven tests for Go",
      "content": "When writing Go tests, table-driven tests with subtests provide better coverage...",
      "outcome": "success",
      "confidence": 0.85,
      "tags": ["go", "testing"]
    }
  ],
  "count": 1
}
```

#### Example

```json
{
  "tool": "memory_search",
  "arguments": {
    "project_id": "contextd",
    "query": "debugging Go concurrency issues",
    "limit": 3
  }
}
```

---

### memory_record

Record a new memory from the current session.

**Use Case**: After solving a problem or discovering a useful pattern, record it for future reference.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `title` | string | Yes | Brief, descriptive title (max 100 chars) |
| `content` | string | Yes | Full description of the strategy or learning |
| `outcome` | string | Yes | `"success"` or `"failure"` |
| `tags` | array | No | Tags for categorization |

#### Response

```json
{
  "id": "mem_xyz789",
  "title": "Use context.WithTimeout for API calls",
  "outcome": "success",
  "confidence": 0.5
}
```

#### Example

```json
{
  "tool": "memory_record",
  "arguments": {
    "project_id": "contextd",
    "title": "Always validate Qdrant collection exists before insert",
    "content": "When inserting into Qdrant, always call CollectionExists first or handle the error gracefully. Auto-creation can cause schema mismatches.",
    "outcome": "success",
    "tags": ["qdrant", "vectors", "error-handling"]
  }
}
```

---

### memory_feedback

Provide feedback on a memory to adjust its confidence score.

**Use Case**: After using a memory, indicate whether it was helpful. This improves future search ranking.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `memory_id` | string | Yes | ID of the memory to rate |
| `helpful` | boolean | Yes | `true` if the memory was helpful, `false` otherwise |

#### Response

```json
{
  "memory_id": "mem_abc123",
  "new_confidence": 0.92,
  "helpful": true
}
```

#### Confidence Scoring

- Initial confidence: 0.5
- Positive feedback: +0.1 (capped at 1.0)
- Negative feedback: -0.15 (floored at 0.0)
- Memories below 0.1 confidence are deprioritized in search

---

## Checkpoint Tools

Checkpoints save and restore session context, enabling recovery from context overflow or session interruption.

### checkpoint_save

Save a checkpoint of the current session state.

**Use Case**: Before a risky operation or when context is approaching limits.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session_id` | string | Yes | Current session identifier |
| `tenant_id` | string | Yes | Tenant/user identifier |
| `project_path` | string | Yes | Path to the project |
| `name` | string | No | Human-readable checkpoint name |
| `description` | string | No | Detailed description |
| `summary` | string | No | Brief summary for quick reference |
| `context` | string | No | Contextual information to restore |
| `full_state` | string | No | Complete session state (JSON) |
| `token_count` | integer | No | Estimated token count |
| `threshold` | float | No | Context threshold that triggered save (0-100) |
| `auto_created` | boolean | No | `true` if system-triggered |
| `metadata` | object | No | Additional key-value metadata |

#### Response

```json
{
  "id": "cp_abc123",
  "session_id": "sess_xyz",
  "summary": "Completed user auth implementation",
  "token_count": 45000,
  "auto_created": false
}
```

---

### checkpoint_list

List available checkpoints for a session or project.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tenant_id` | string | Yes | Tenant identifier |
| `session_id` | string | No | Filter by session ID |
| `project_path` | string | No | Filter by project path |
| `limit` | integer | No | Maximum results (default: 20) |
| `auto_only` | boolean | No | Only return auto-created checkpoints |

#### Response

```json
{
  "checkpoints": [
    {
      "id": "cp_abc123",
      "session_id": "sess_xyz",
      "name": "Before refactor",
      "description": "State before major refactoring",
      "summary": "Auth system working, starting refactor",
      "token_count": 45000,
      "threshold": 70,
      "auto_created": false,
      "created_at": "2024-12-02T10:30:00Z"
    }
  ],
  "count": 1
}
```

---

### checkpoint_resume

Resume from a checkpoint at a specified detail level.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `checkpoint_id` | string | Yes | Checkpoint to resume |
| `tenant_id` | string | Yes | Tenant identifier |
| `level` | string | Yes | Detail level: `"summary"`, `"context"`, or `"full"` |

#### Resume Levels

| Level | Description | Token Cost |
|-------|-------------|------------|
| `summary` | Brief summary only | Lowest |
| `context` | Summary + contextual information | Medium |
| `full` | Complete session state | Highest |

#### Response

```json
{
  "checkpoint_id": "cp_abc123",
  "session_id": "sess_xyz",
  "content": "Resumed context content...",
  "token_count": 5000,
  "level": "context"
}
```

---

## Remediation Tools

Remediations track error patterns and their fixes, enabling faster resolution of recurring issues.

### remediation_search

Search for known fixes to an error.

**Use Case**: When encountering an error, search for previously recorded solutions.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Error message or pattern |
| `tenant_id` | string | Yes | Tenant identifier |
| `scope` | string | No | Scope: `"project"`, `"team"`, or `"org"` |
| `category` | string | No | Error category filter |
| `min_confidence` | float | No | Minimum confidence (0-1) |
| `limit` | integer | No | Maximum results (default: 10) |
| `team_id` | string | No | Team ID for team/project scope |
| `project_path` | string | No | Project path for project scope |
| `include_hierarchy` | boolean | No | Search parent scopes (project->team->org) |

#### Error Categories

- `build` - Build/compilation errors
- `runtime` - Runtime errors
- `test` - Test failures
- `config` - Configuration issues
- `dependency` - Dependency problems
- `network` - Network/API errors
- `database` - Database errors
- `unknown` - Uncategorized

#### Response

```json
{
  "remediations": [
    {
      "id": "rem_abc123",
      "title": "Fix ONNX runtime version mismatch",
      "problem": "Error: API version 22 not available",
      "root_cause": "ONNX runtime version too old",
      "solution": "Upgrade to ONNX runtime v1.23.2 or later",
      "category": "dependency",
      "confidence": 0.9,
      "score": 0.95,
      "usage_count": 5
    }
  ],
  "count": 1
}
```

---

### remediation_record

Record a new remediation after fixing an error.

**Use Case**: After solving an error, record the fix for future reference.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Brief title |
| `problem` | string | Yes | Problem description |
| `root_cause` | string | Yes | Root cause analysis |
| `solution` | string | Yes | How to fix it |
| `category` | string | Yes | Error category (see list above) |
| `tenant_id` | string | Yes | Tenant identifier |
| `scope` | string | Yes | Scope level |
| `symptoms` | array | No | Observable symptoms |
| `code_diff` | string | No | Code changes (diff format) |
| `affected_files` | array | No | Files that were changed |
| `confidence` | float | No | Confidence score (default: 0.5) |
| `tags` | array | No | Tags for categorization |
| `team_id` | string | No | Team ID |
| `project_path` | string | No | Project path |
| `session_id` | string | No | Session that created this |

#### Response

```json
{
  "id": "rem_xyz789",
  "title": "Fix Docker build cache issue",
  "category": "build",
  "confidence": 0.5
}
```

---

## Utility Tools

### repository_index

Index a repository for semantic code search.

**Use Case**: Before starting work on a codebase, index it for better context retrieval.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Repository path to index |
| `tenant_id` | string | No | Tenant identifier (defaults to git username) |
| `include_patterns` | array | No | Glob patterns to include (default: `["*"]`) |
| `exclude_patterns` | array | No | Glob patterns to exclude |
| `max_file_size` | integer | No | Maximum file size in bytes (default: 1MB) |

#### Default Exclusions

When no exclude patterns are specified, ContextD reads from:
1. `.gitignore`
2. `.dockerignore`
3. `.contextdignore`

Fallback exclusions if no ignore files found:
- `.git/**`
- `node_modules/**`
- `vendor/**`
- `__pycache__/**`

#### Response

```json
{
  "path": "/home/user/projects/myapp",
  "files_indexed": 156,
  "include_patterns": ["*"],
  "exclude_patterns": [".git/**", "node_modules/**"],
  "max_file_size": 1048576
}
```

---

### troubleshoot_diagnose

Diagnose an error using AI and known patterns.

**Use Case**: When encountering a complex error, get AI-powered diagnosis and recommendations.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `error_message` | string | Yes | Error message to diagnose |
| `error_context` | string | No | Additional context (stack trace, logs, etc.) |

#### Response

```json
{
  "error_message": "connection refused: dial tcp 127.0.0.1:6334",
  "root_cause": "Qdrant server not running or not listening on expected port",
  "hypotheses": [
    {
      "description": "Qdrant container not started",
      "confidence": 0.8,
      "evidence": ["Port 6334 is the Qdrant gRPC port"]
    },
    {
      "description": "Port conflict with another service",
      "confidence": 0.3,
      "evidence": []
    }
  ],
  "recommendations": [
    "Check if Qdrant container is running: docker ps | grep qdrant",
    "Verify Qdrant is listening: curl http://localhost:6333/health",
    "Check for port conflicts: lsof -i :6334"
  ],
  "related_patterns": [],
  "confidence": 0.75
}
```

---

## Security Notes

### Secret Scrubbing

All tool responses are automatically scrubbed for secrets using gitleaks patterns. This includes:

- API keys and tokens
- Passwords and credentials
- Private keys
- Connection strings
- AWS/GCP/Azure credentials

Detected secrets are replaced with `[REDACTED]`.

### Multi-Tenant Isolation

- All data is isolated by `tenant_id`
- Remediations can be scoped to project, team, or organization
- Cross-tenant data access is not possible

---

## Error Handling

All tools return errors in a consistent format:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Checkpoint not found: cp_abc123",
    "details": {}
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `NOT_FOUND` | Requested resource doesn't exist |
| `INVALID_INPUT` | Invalid or missing parameters |
| `INTERNAL_ERROR` | Server-side error |
| `RATE_LIMITED` | Too many requests |
| `UNAUTHORIZED` | Invalid tenant ID or permissions |
