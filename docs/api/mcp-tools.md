# MCP Tools API Reference

contextd exposes its functionality through the Model Context Protocol (MCP). This document provides complete reference documentation for all available tools.

---

## Table of Contents

- [Overview](#overview)
- [Memory Tools](#memory-tools)
  - [memory_search](#memory_search)
  - [memory_record](#memory_record)
  - [memory_feedback](#memory_feedback)
  - [memory_outcome](#memory_outcome)
  - [memory_consolidate](#memory_consolidate)
  - [memory_consolidate_session](#memory_consolidate_session)
- [Checkpoint Tools](#checkpoint-tools)
  - [checkpoint_save](#checkpoint_save)
  - [checkpoint_list](#checkpoint_list)
  - [checkpoint_resume](#checkpoint_resume)
- [Remediation Tools](#remediation-tools)
  - [remediation_search](#remediation_search)
  - [remediation_record](#remediation_record)
  - [remediation_feedback](#remediation_feedback)
- [Context-Folding Tools](#context-folding-tools)
  - [branch_create](#branch_create)
  - [branch_return](#branch_return)
  - [branch_status](#branch_status)
- [Repository Tools](#repository-tools)
  - [repository_index](#repository_index)
  - [repository_search](#repository_search)
  - [semantic_search](#semantic_search)
- [Conversation Tools](#conversation-tools)
  - [conversation_index](#conversation_index)
  - [conversation_search](#conversation_search)
- [Utility Tools](#utility-tools)
  - [troubleshoot_diagnose](#troubleshoot_diagnose)
  - [reflect_report](#reflect_report)
  - [reflect_analyze](#reflect_analyze)
- [Security Notes](#security-notes)
- [Error Handling](#error-handling)

---

## Overview

ContextD provides 25 MCP tools organized into seven categories:

| Category | Tools | Purpose |
|----------|-------|---------|
| **Memory** | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome`, `memory_consolidate`, `memory_consolidate_session` | Cross-session learning and strategies |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Context persistence and recovery |
| **Remediation** | `remediation_search`, `remediation_record`, `remediation_feedback` | Error pattern tracking and fixes |
| **Context-Folding** | `branch_create`, `branch_return`, `branch_status` | Isolated sub-task execution with token budgets |
| **Repository** | `semantic_search`, `repository_index`, `repository_search` | Code indexing and semantic search |
| **Conversation** | `conversation_index`, `conversation_search` | Claude Code conversation indexing and search |
| **Utility** | `troubleshoot_diagnose`, `reflect_report`, `reflect_analyze` | Diagnostics and self-reflection |

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

### memory_outcome

Report whether a task succeeded after using a memory.

**Use Case**: After completing a task where you used a retrieved memory, call this to report whether the task succeeded. This helps the system learn which memories are actually useful in practice.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `memory_id` | string | Yes | ID of the memory that was used |
| `succeeded` | boolean | Yes | `true` if the task succeeded, `false` if it failed |
| `session_id` | string | No | Optional session ID for correlation |

#### Response

```json
{
  "recorded": true,
  "new_confidence": 0.87,
  "message": "Outcome recorded"
}
```

#### Example

```json
{
  "tool": "memory_outcome",
  "arguments": {
    "memory_id": "mem_abc123",
    "succeeded": true,
    "session_id": "sess_xyz"
  }
}
```

#### How It Works

- **Success**: When `succeeded=true`, the memory's confidence score increases, making it more likely to appear in future searches
- **Failure**: When `succeeded=false`, the memory's confidence score decreases, deprioritizing it in search results
- **Learning**: Over time, the system learns which memories lead to successful outcomes and surfaces them more prominently

#### When to Use

1. **After task completion**: Once you've finished a task using a memory's strategy
2. **Both success and failure**: Report outcomes whether the task succeeded or failed
3. **Before session end**: Ideally report outcomes before the session ends for best tracking

---

### memory_consolidate

Merge similar memories to reduce redundancy and improve knowledge quality.

**Use Case**: Periodically clean up your memory base by merging related memories into refined summaries.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `similarity_threshold` | float | No | Minimum similarity for consolidation (0-1, default: 0.8) |
| `dry_run` | boolean | No | Preview without making changes (default: false) |
| `max_clusters` | integer | No | Max clusters per run (0 = no limit) |

#### Response

```json
{
  "created_memories": ["mem_new1", "mem_new2"],
  "archived_memories": ["mem_old1", "mem_old2", "mem_old3"],
  "skipped_count": 5,
  "total_processed": 20,
  "duration_seconds": 2.5
}
```

---

### memory_consolidate_session

Flush and summarize a session's buffered turns into session-level memories. Only effective when granularity is set to `session`.

**Use Case**: At the end of a session, consolidate turn-level learnings into higher-level session memories.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `session_id` | string | Yes | Session ID to flush |

#### Response

```json
{
  "memory_ids": ["mem_abc", "mem_def"],
  "count": 2,
  "message": "Session sess_xyz flushed: 2 memories created"
}
```

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

### remediation_feedback

Provide feedback on whether a remediation was helpful. Updates the confidence score based on real-world results.

**Use Case**: After trying a remediation fix, report whether it actually worked. This improves future search ranking.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `remediation_id` | string | Yes | ID of the remediation to rate |
| `helpful` | boolean | Yes | `true` if the fix worked, `false` otherwise |
| `tenant_id` | string | No | Tenant identifier (auto-derived from git if not provided) |
| `project_path` | string | No | Project path (used to auto-derive tenant_id) |

#### Response

```json
{
  "remediation_id": "rem_abc123",
  "new_confidence": 0.65,
  "helpful": true
}
```

---

## Context-Folding Tools

Context-folding enables **active context management** by isolating complex sub-tasks with dedicated token budgets. Branches execute in isolation and return only scrubbed summaries to the main context, achieving **90%+ context compression**.

### branch_create

Create a new context-folding branch for isolated sub-task execution.

**Use Case**: When you need to perform complex operations (file exploration, research, debugging) without cluttering the main context. The branch gets its own token budget and only a summary returns to the parent context.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session_id` | string | Yes | Session identifier |
| `description` | string | Yes | Brief description of what the branch will do |
| `prompt` | string | No | Detailed prompt/instructions for the branch |
| `budget` | integer | No | Token budget for this branch (default: 8192) |
| `timeout_seconds` | integer | No | Timeout in seconds (default: 300) |

#### Response

```json
{
  "branch_id": "br_abc123",
  "budget_allocated": 8192,
  "depth": 0
}
```

#### Example

```json
{
  "tool": "branch_create",
  "arguments": {
    "session_id": "sess_xyz",
    "description": "Search 10 files for authentication function",
    "prompt": "Find the authenticate() function by searching through src/ directory",
    "budget": 4096,
    "timeout_seconds": 120
  }
}
```

#### Notes

- **Nesting**: Branches can be nested up to a configurable depth (default: 3 levels)
- **Budget**: If budget is exceeded, the branch is force-terminated
- **Isolation**: Each branch has its own context and token budget
- **Cleanup**: Child branches are automatically force-returned when parent returns

---

### branch_return

Return from a context-folding branch with results.

**Use Case**: After completing work in a branch, return a concise summary to the parent context. The message is automatically scrubbed for secrets before being passed to the parent.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `branch_id` | string | Yes | Branch ID to return from |
| `message` | string | Yes | Result message/summary from the branch |

#### Response

```json
{
  "success": true,
  "tokens_used": 3542,
  "message": "Found authenticate() function in src/auth/handler.go:42"
}
```

#### Example

```json
{
  "tool": "branch_return",
  "arguments": {
    "branch_id": "br_abc123",
    "message": "Found authenticate() function in src/auth/handler.go:42. Uses JWT tokens with HS256 signing."
  }
}
```

#### Secret Scrubbing

All return messages are automatically scrubbed for secrets using gitleaks patterns. Detected secrets are replaced with `[REDACTED]`.

**Example:**
```
Input:  "API key is sk_live_abc123xyz"
Output: "API key is [REDACTED]"
```

---

### branch_status

Get the status of a specific branch or the active branch for a session.

**Use Case**: Check branch state, budget usage, and depth information. Useful for monitoring progress and budget consumption during long-running operations.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `branch_id` | string | No* | Specific branch ID to check |
| `session_id` | string | No* | Session ID to get active branch for |

*Either `branch_id` or `session_id` must be provided (not both).

#### Response

```json
{
  "branch_id": "br_abc123",
  "session_id": "sess_xyz",
  "status": "active",
  "depth": 1,
  "budget_used": 2341,
  "budget_total": 8192,
  "budget_remaining": 5851
}
```

#### Status Values

| Status | Description |
|--------|-------------|
| `active` | Branch is currently executing |
| `completed` | Branch successfully returned |
| `failed` | Branch execution failed |
| `timeout` | Branch exceeded timeout limit |
| `none` | No active branch found (when querying by session_id) |

#### Example

```json
{
  "tool": "branch_status",
  "arguments": {
    "branch_id": "br_abc123"
  }
}
```

Or query by session to get the active branch:

```json
{
  "tool": "branch_status",
  "arguments": {
    "session_id": "sess_xyz"
  }
}
```

---

## Repository Tools

Repository tools enable semantic indexing and search of codebases, allowing agents to find relevant code by meaning rather than exact keyword matches.

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
  "branch": "main",
  "collection_name": "myapp_codebase",
  "files_indexed": 156,
  "include_patterns": ["*"],
  "exclude_patterns": [".git/**", "node_modules/**"],
  "max_file_size": 1048576
}
```

---

### repository_search

Semantic search over indexed repository code in _codebase collection.

**Use Case**: Find code by meaning rather than exact keyword match. Prefer using `collection_name` from `repository_index` output.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Semantic search query |
| `project_path` | string | Yes | Project path to search within (required for tenant context) |
| `collection_name` | string | No | Collection name from `repository_index` (preferred - avoids tenant_id derivation issues) |
| `tenant_id` | string | No | Tenant identifier (auto-derived from project_path if not provided) |
| `branch` | string | No | Filter by branch (empty = all branches) |
| `limit` | integer | No | Maximum results (default: 10) |
| `content_mode` | string | No | Content mode: `"minimal"` (default), `"preview"`, or `"full"` |

#### Content Modes

| Mode | Description | Response Fields |
|------|-------------|----------------|
| `minimal` | File paths and scores only (lowest token cost) | `file_path`, `score`, `branch` |
| `preview` | First 200 characters of content | Above + `content_preview` |
| `full` | Complete file content and metadata | Above + `content`, `metadata` |

#### Response

**Minimal mode (default):**
```json
{
  "results": [
    {
      "file_path": "cmd/main.go",
      "score": 0.89,
      "branch": "main"
    }
  ],
  "count": 1,
  "query": "entry point",
  "branch": "main",
  "content_mode": "minimal"
}
```

**Preview mode:**
```json
{
  "results": [
    {
      "file_path": "cmd/main.go",
      "content_preview": "func main() {\n\tctx := context.Background()\n\t// Initialize config\n\tcfg, err := config.Load()\n\tif err != nil {\n\t\tlog.Fatal(\"config load failed\", zap.Error(err))\n\t}\n...",
      "score": 0.89,
      "branch": "main"
    }
  ],
  "count": 1,
  "query": "entry point",
  "branch": "main",
  "content_mode": "preview"
}
```

**Full mode:**
```json
{
  "results": [
    {
      "file_path": "cmd/main.go",
      "content": "func main() { ... }",
      "score": 0.89,
      "branch": "main",
      "metadata": {}
    }
  ],
  "count": 1,
  "query": "entry point",
  "branch": "main",
  "content_mode": "full"
}
```

---

### semantic_search

Smart search that uses semantic understanding, falling back to grep if needed.

**Use Case**: Primary search tool. Tries to understand intent first, falls back to text search if no semantic matches found.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query (natural language or pattern) |
| `project_path` | string | Yes | Project path to search within |
| `tenant_id` | string | No | Tenant identifier |
| `branch` | string | No | Filter by branch |
| `limit` | integer | No | Maximum results (default: 10) |

#### Response

```json
{
  "results": [
    {
      "file_path": "internal/service.go",
      "content": "func Search() ...",
      "score": 0.95,
      "line_number": 0
    }
  ],
  "count": 1,
  "query": "search function",
  "source": "semantic"
}
```

---

## Conversation Tools

Conversation tools enable indexing and searching of Claude Code conversation history. This allows agents to learn from past interactions, decisions, and patterns across sessions.

### conversation_index

Index Claude Code conversation files for semantic search.

**Use Case**: Before working on a project, index conversation history to enable searching for past decisions, discussions, and patterns.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_path` | string | Yes | Path to project to index conversations for |
| `tenant_id` | string | No | Tenant identifier (auto-derived from project_path via git remote if not provided) |
| `session_ids` | array | No | Specific session IDs to index (empty = all sessions) |
| `enable_llm` | boolean | No | Enable LLM-based decision extraction (default: false). **NOTE**: LLM summarization is not yet implemented - this flag is reserved for future use. Currently uses heuristic extraction only. |
| `force` | boolean | No | Force reindexing of existing sessions (default: false) |

#### Response

```json
{
  "sessions_indexed": 5,
  "messages_indexed": 234,
  "decisions_extracted": 12,
  "files_referenced": ["internal/mcp/server.go", "cmd/contextd/main.go"],
  "error_count": 0
}
```

#### Example

```json
{
  "tool": "conversation_index",
  "arguments": {
    "project_path": "/home/user/projects/contextd",
    "force": false
  }
}
```

#### Notes

- Conversation files are read from `<project_path>/.claude/conversations/*.jsonl`
- Documents are indexed as type `message`, `decision`, or `summary`
- Heuristic decision extraction uses pattern matching for words like "decided", "choosing", "approach"
- LLM-based extraction (when implemented) will provide more accurate decision identification

---

### conversation_search

Search indexed Claude Code conversations for relevant past context.

**Use Case**: Find relevant past discussions, decisions, and patterns from conversation history to inform current work.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Semantic search query |
| `project_path` | string | Yes | Project path to search within |
| `tenant_id` | string | No | Tenant identifier (auto-derived from project_path via git remote if not provided) |
| `types` | array | No | Filter by document types: `"message"`, `"decision"`, or `"summary"` |
| `tags` | array | No | Filter by tags |
| `file_path` | string | No | Filter by file path discussed |
| `domain` | string | No | Filter by domain (e.g., `"kubernetes"`, `"frontend"`, `"database"`) |
| `limit` | integer | No | Maximum results to return (default: 10) |

#### Response

```json
{
  "query": "authentication implementation",
  "results": [
    {
      "id": "doc_abc123",
      "session_id": "sess_xyz",
      "type": "decision",
      "content": "Decided to use JWT tokens with HS256 signing for API authentication...",
      "score": 0.92,
      "timestamp": 1701234567,
      "tags": ["auth", "api"],
      "domain": "backend"
    }
  ],
  "total": 1,
  "took_ms": 45
}
```

#### Example

```json
{
  "tool": "conversation_search",
  "arguments": {
    "query": "how did we implement user authentication",
    "project_path": "/home/user/projects/myapp",
    "types": ["decision", "message"],
    "limit": 5
  }
}
```

#### Document Types

| Type | Description |
|------|-------------|
| `message` | Individual conversation messages |
| `decision` | Extracted decisions (e.g., "decided to use X", "choosing Y approach") |
| `summary` | Conversation summaries (future) |

---

## Utility Tools

Utility tools provide diagnostics, troubleshooting, and self-reflection capabilities.

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

### reflect_report

Generate a self-reflection report analyzing memories and patterns for a project.

**Use Case**: Periodically generate reports to understand behavioral patterns, success/failure trends, and get recommendations based on accumulated memories.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `project_path` | string | No | Project path for repository context (optional - enables saving report to disk) |
| `period_days` | integer | No | Number of days to analyze (default: 30) |
| `include_patterns` | boolean | No | Include pattern analysis (default: true) |
| `include_correlations` | boolean | No | Include correlation analysis (default: true) |
| `include_insights` | boolean | No | Include insights (default: true) |
| `max_insights` | integer | No | Maximum insights to include (default: 10) |
| `format` | string | No | Output format: `"json"` (default), `"text"`, or `"markdown"` |

#### Response

```json
{
  "report_id": "report_abc123",
  "project_id": "contextd",
  "generated_at": "2024-12-02T10:30:00Z",
  "period_days": 30,
  "summary": "Analysis of 42 memories shows 85% success rate with strong patterns in Go testing and API design",
  "statistics": {
    "total_memories": 42,
    "success_count": 36,
    "failure_count": 6,
    "avg_confidence": 0.78
  },
  "pattern_count": 8,
  "insight_count": 10,
  "format": "json",
  "formatted_text": "",
  "report_path": "/home/user/projects/contextd/.claude/reflections/reflection-20241202-103000.json"
}
```

#### Example

```json
{
  "tool": "reflect_report",
  "arguments": {
    "project_id": "contextd",
    "project_path": "/home/user/projects/contextd",
    "period_days": 30,
    "format": "markdown",
    "max_insights": 5
  }
}
```

#### Notes

- If `project_path` is provided, the report will be saved to `<project_path>/.claude/reflections/`
- `formatted_text` is only populated for `text` and `markdown` formats
- Default behavior includes all sections (patterns, correlations, insights) unless explicitly disabled

---

### reflect_analyze

Analyze memories for behavioral patterns.

**Use Case**: Identify recurring patterns in your work - what strategies consistently work, what approaches fail, and how behaviors are trending over time.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `min_confidence` | float | No | Minimum confidence threshold (default: 0.3) |
| `min_frequency` | integer | No | Minimum pattern frequency (default: 2) |
| `include_tags` | array | No | Filter to specific tags |
| `exclude_tags` | array | No | Exclude specific tags |
| `max_patterns` | integer | No | Maximum patterns to return (default: 20) |

#### Response

```json
{
  "project_id": "contextd",
  "pattern_count": 5,
  "patterns": [
    {
      "category": "success",
      "description": "Table-driven tests in Go provide better coverage",
      "frequency": 8,
      "confidence": 0.92,
      "examples": ["mem_abc123", "mem_def456"],
      "tags": ["go", "testing"]
    },
    {
      "category": "recurring",
      "description": "JWT token validation requires careful error handling",
      "frequency": 5,
      "confidence": 0.78,
      "examples": ["mem_xyz789"],
      "tags": ["auth", "security"]
    }
  ]
}
```

#### Example

```json
{
  "tool": "reflect_analyze",
  "arguments": {
    "project_id": "contextd",
    "min_confidence": 0.5,
    "min_frequency": 3,
    "include_tags": ["go", "testing"],
    "max_patterns": 10
  }
}
```

#### Pattern Categories

| Category | Description |
|----------|-------------|
| `success` | Patterns from consistently successful strategies |
| `failure` | Patterns from repeatedly failing approaches |
| `recurring` | Patterns that appear frequently regardless of outcome |
| `improving` | Patterns with increasing confidence over time |
| `declining` | Patterns with decreasing confidence over time |

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

> **📖 See Also:** [Comprehensive Error Codes Reference](./error-codes.md) - Complete catalog of all error codes with troubleshooting guides.

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

For complete error code documentation with troubleshooting guides and examples, see [error-codes.md](./error-codes.md).

---

## Related Documentation

- [Main Documentation](../CONTEXTD.md) - Quick start and overview
- [Architecture Overview](../architecture.md) - Detailed component descriptions
- [Hook Setup Guide](../HOOKS.md) - Claude Code lifecycle integration
- [Configuration Reference](../configuration.md) - All configuration options
- [Error Codes Reference](./error-codes.md) - Complete error codes with troubleshooting
- [Troubleshooting](../troubleshooting.md) - Common issues and fixes
