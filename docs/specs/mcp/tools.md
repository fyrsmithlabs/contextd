# MCP Tool Definitions

**Parent**: [../SPEC.md](../SPEC.md)

This document provides detailed specifications for all 16 MCP tools.

---

## Session Management Tools

### 1. checkpoint_save

**Purpose**: Save a session checkpoint for resuming work later.

**Description**: Stores a session checkpoint with summary, description, project path, context metadata, and tags. Automatic vector embeddings are generated for semantic search.

**Input Schema**:
```json
{
  "summary": "string (required, max 500 chars)",
  "description": "string (optional, max 5000 chars)",
  "project_path": "string (required, absolute path)",
  "context": "object (optional, key-value metadata)",
  "tags": "array of strings (optional, max 20 tags)"
}
```

**Output Schema**:
```json
{
  "id": "string (checkpoint ID)",
  "summary": "string",
  "created_at": "timestamp",
  "token_count": "integer (embedding tokens)"
}
```

**Timeout**: 30 seconds

---

### 2. checkpoint_search

**Purpose**: Search checkpoints using semantic similarity.

**Description**: Finds relevant checkpoints based on query meaning, with optional filtering by project path and tags. Uses vector similarity search with cosine distance.

**Input Schema**:
```json
{
  "query": "string (required, max 1000 chars)",
  "top_k": "integer (optional, default: 5, max: 100)",
  "project_path": "string (optional, filter by project)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "summary": "string",
      "description": "string",
      "project_path": "string",
      "context": "object",
      "tags": "array of strings",
      "score": "float (similarity score 0-1)",
      "distance": "float (cosine distance)",
      "created_at": "timestamp"
    }
  ],
  "query": "string (original query)",
  "top_k": "integer"
}
```

**Timeout**: 10 seconds

---

### 3. checkpoint_list

**Purpose**: List recent checkpoints with pagination.

**Description**: Supports filtering by project path and sorting by creation/update time. Useful for browsing recent work.

**Input Schema**:
```json
{
  "limit": "integer (optional, default: 10, max: 100)",
  "offset": "integer (optional, default: 0)",
  "project_path": "string (optional, filter by project)",
  "sort_by": "string (optional, created_at|updated_at)"
}
```

**Output Schema**:
```json
{
  "checkpoints": [
    {
      "id": "string",
      "summary": "string",
      "description": "string",
      "project_path": "string",
      "context": "object",
      "tags": "array of strings",
      "created_at": "timestamp"
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

**Timeout**: 5 seconds

---

## Error Resolution Tools

### 4. remediation_save

**Purpose**: Store an error solution for future reference.

**Description**: Saves error message, type, solution, stack trace, and metadata with vector embeddings for intelligent matching. Supports severity levels and project-specific context.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "error_type": "string (required)",
  "solution": "string (required)",
  "project_path": "string (optional)",
  "context": "object (optional, error context)",
  "tags": "array of strings (optional)",
  "severity": "string (optional, low|medium|high|critical)",
  "stack_trace": "string (optional, max 50000 chars)"
}
```

**Output Schema**:
```json
{
  "id": "string (remediation ID)",
  "error_message": "string",
  "error_type": "string",
  "solution": "string",
  "created_at": "timestamp"
}
```

**Timeout**: 30 seconds

---

### 5. remediation_search

**Purpose**: Find similar error solutions using hybrid matching.

**Description**: Returns ranked results with match scores using 70% semantic similarity + 30% string matching. Includes detailed match breakdowns for transparency.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "stack_trace": "string (optional, for better matching)",
  "limit": "integer (optional, default: 5, max: 100)",
  "min_score": "float (optional, 0-1, default: 0.5)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "error_message": "string",
      "error_type": "string",
      "solution": "string",
      "tags": "array of strings",
      "match_score": "float (combined score)",
      "semantic_score": "float (70% weight)",
      "string_score": "float (30% weight)",
      "stack_trace_match": "boolean",
      "error_type_match": "boolean",
      "context": "object",
      "created_at": "timestamp"
    }
  ],
  "query": "string (original error message)",
  "total": "integer"
}
```

**Timeout**: 10 seconds

---

## AI Diagnosis Tools

### 6. troubleshoot

**Purpose**: AI-powered error diagnosis and troubleshooting.

**Description**: Analyzes error messages and stack traces, identifies root causes, generates hypotheses, and recommends diagnostic steps and solutions. Includes similar issues from knowledge base.

**Input Schema**:
```json
{
  "error_message": "string (required, max 10000 chars)",
  "stack_trace": "string (optional)",
  "context": "object (optional, environment, versions, etc)",
  "category": "string (optional, configuration|resource|dependency|etc)",
  "mode": "string (optional, auto|interactive|guided, default: auto)",
  "tags": "array of strings (optional)",
  "top_k": "integer (optional, similar issues, default: 5)"
}
```

**Output Schema**:
```json
{
  "session_id": "string (troubleshooting session ID)",
  "root_cause": "string",
  "confidence": "string (high|medium|low)",
  "confidence_score": "float (0-1)",
  "category": "string",
  "severity": "string",
  "hypotheses": [
    {
      "description": "string",
      "probability": "float",
      "evidence": "array of strings",
      "category": "string",
      "verification_steps": "array of strings"
    }
  ],
  "similar_issues": [
    {
      "id": "string",
      "error_pattern": "string",
      "root_cause": "string",
      "solution": "string",
      "match_score": "float",
      "semantic_score": "float",
      "success_rate": "float",
      "severity": "string",
      "category": "string",
      "tags": "array of strings",
      "confidence": "string",
      "is_destructive": "boolean",
      "safety_warnings": "array of strings"
    }
  ],
  "recommended_actions": [
    {
      "step": "integer",
      "description": "string",
      "commands": "array of strings",
      "expected_outcome": "string",
      "destructive": "boolean",
      "safety_notes": "string"
    }
  ],
  "diagnostic_steps": "array of strings",
  "time_taken_ms": "float",
  "diagnosed_at": "timestamp"
}
```

**Timeout**: 60 seconds

---

### 7. list_patterns

**Purpose**: Browse troubleshooting patterns from the knowledge base.

**Description**: Supports filtering by category, severity, and minimum success rate. Useful for learning from past solutions.

**Input Schema**:
```json
{
  "category": "string (optional, filter by category)",
  "severity": "string (optional, critical|high|medium|low)",
  "min_success_rate": "float (optional, 0-1)",
  "limit": "integer (optional, default: 10, max: 100)"
}
```

**Output Schema**:
```json
{
  "patterns": [
    {
      "id": "string",
      "error_pattern": "string",
      "category": "string",
      "severity": "string",
      "root_cause": "string",
      "solution": "string",
      "success_rate": "float",
      "tags": "array of strings",
      "usage_count": "integer",
      "last_used": "timestamp"
    }
  ],
  "total": "integer"
}
```

**Timeout**: 5 seconds

---

## Repository Indexing Tool

### 8. index_repository

**Purpose**: Index an existing repository or directory for semantic search.

**Description**: Creates searchable checkpoints from files matching include patterns while respecting exclude patterns and file size limits. Supports glob patterns for flexible file selection.

**Input Schema**:
```json
{
  "path": "string (required, absolute path to repository)",
  "include_patterns": "array of strings (optional, e.g., ['*.md', '*.txt'])",
  "exclude_patterns": "array of strings (optional, e.g., ['*.log', 'node_modules/**'])",
  "max_file_size": "integer (optional, bytes, default: 1MB, max: 10MB)"
}
```

**Output Schema**:
```json
{
  "path": "string (repository path indexed)",
  "files_indexed": "integer",
  "include_patterns": "array of strings",
  "exclude_patterns": "array of strings",
  "max_file_size": "integer",
  "indexed_at": "timestamp"
}
```

**Timeout**: 300 seconds (5 minutes)

**Security Note**: Path traversal protection is enforced. All indexed files must be within the specified repository path.

---

## Skills Management Tools

### 9. skill_create

**Purpose**: Create a new reusable skill/workflow template.

**Description**: Skills can be searched semantically and applied to similar situations. Supports versioning, categorization, and metadata.

**Input Schema**:
```json
{
  "name": "string (required, max 200 chars)",
  "description": "string (required, max 2000 chars)",
  "content": "string (required, markdown, max 50000 chars)",
  "version": "string (required, semver, e.g., '1.0.0')",
  "author": "string (required)",
  "category": "string (required, debugging|deployment|analysis|etc)",
  "prerequisites": "array of strings (optional)",
  "expected_outcome": "string (optional)",
  "tags": "array of strings (optional)",
  "metadata": "object (optional)"
}
```

**Output Schema**:
```json
{
  "id": "string (skill ID)",
  "name": "string",
  "version": "string",
  "token_count": "integer",
  "created_at": "timestamp"
}
```

**Timeout**: 120 seconds (longer due to large content embedding)

---

### 10. skill_search

**Purpose**: Search for skills using semantic similarity.

**Description**: Find relevant workflows and templates based on query meaning, with optional filtering by category and tags.

**Input Schema**:
```json
{
  "query": "string (required, max 1000 chars)",
  "top_k": "integer (optional, default: 5, max: 100)",
  "category": "string (optional, filter by category)",
  "tags": "array of strings (optional, filter by tags)"
}
```

**Output Schema**:
```json
{
  "results": [
    {
      "id": "string",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "prerequisites": "array of strings",
      "expected_outcome": "string",
      "tags": "array of strings",
      "usage_count": "integer",
      "success_rate": "float",
      "score": "float",
      "distance": "float",
      "metadata": "object",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "query": "string",
  "top_k": "integer"
}
```

**Timeout**: 10 seconds

---

### 11. skill_list

**Purpose**: List all skills with pagination and filtering.

**Description**: Supports filtering by category, tags, and sorting by creation date, usage count, or success rate.

**Input Schema**:
```json
{
  "limit": "integer (optional, default: 10, max: 100)",
  "offset": "integer (optional, default: 0)",
  "category": "string (optional, filter by category)",
  "tags": "array of strings (optional, filter by tags)",
  "sort_by": "string (optional, created_at|updated_at|usage_count|success_rate)"
}
```

**Output Schema**:
```json
{
  "skills": [
    {
      "id": "string",
      "name": "string",
      "description": "string",
      "content": "string",
      "version": "string",
      "author": "string",
      "category": "string",
      "prerequisites": "array of strings",
      "expected_outcome": "string",
      "tags": "array of strings",
      "usage_count": "integer",
      "success_rate": "float",
      "metadata": "object",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": "integer",
  "limit": "integer",
  "offset": "integer"
}
```

**Timeout**: 30 seconds

---

### 12. skill_update

**Purpose**: Update an existing skill.

**Description**: Allows modifying name, description, content, version, tags, and metadata. All fields are optional except ID.

**Input Schema**:
```json
{
  "id": "string (required, skill ID)",
  "name": "string (optional)",
  "description": "string (optional)",
  "content": "string (optional)",
  "version": "string (optional)",
  "category": "string (optional)",
  "prerequisites": "array of strings (optional)",
  "expected_outcome": "string (optional)",
  "tags": "array of strings (optional)",
  "metadata": "object (optional)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "name": "string",
  "version": "string",
  "updated_at": "timestamp"
}
```

**Timeout**: 120 seconds

---

### 13. skill_delete

**Purpose**: Delete a skill by ID.

**Description**: This action cannot be undone. Removes skill from database and vector store.

**Input Schema**:
```json
{
  "id": "string (required, skill ID to delete)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "message": "string (confirmation)"
}
```

**Timeout**: 30 seconds

---

### 14. skill_apply

**Purpose**: Apply a skill to the current context.

**Description**: Returns the skill content and tracks usage statistics. Optionally records success/failure for success rate calculation.

**Input Schema**:
```json
{
  "id": "string (required, skill ID)",
  "success": "boolean (optional, for tracking)"
}
```

**Output Schema**:
```json
{
  "id": "string",
  "name": "string",
  "content": "string (skill content to apply)",
  "prerequisites": "array of strings",
  "expected_outcome": "string",
  "usage_count": "integer",
  "success_rate": "float"
}
```

**Timeout**: 30 seconds

---

## System Operations Tools

### 15. status

**Purpose**: Get contextd service status and health information.

**Description**: Shows service health, version, uptime, and system metrics. Useful for monitoring and debugging.

**Input Schema**:
```json
{}
```

**Output Schema**:
```json
{
  "status": "string (healthy|degraded|unhealthy)",
  "version": "string (service version)",
  "uptime": "string (optional)",
  "services": {
    "checkpoint": {
      "status": "string (healthy|unhealthy|unknown)",
      "error": "string (optional)"
    }
  },
  "metrics": {
    "tools_available": "integer",
    "mcp_server": "string"
  },
  "last_updated": "timestamp"
}
```

**Timeout**: 30 seconds

---

### 16. analytics_get

**Purpose**: Get context usage analytics and metrics.

**Description**: Tracks token reduction, feature adoption, performance metrics, and business impact. Shows average token savings, search precision, and time saved.

**Input Schema**:
```json
{
  "period": "string (optional, daily|weekly|monthly|all-time, default: weekly)",
  "project_path": "string (optional, filter by project)",
  "start_date": "string (optional, YYYY-MM-DD)",
  "end_date": "string (optional, YYYY-MM-DD)"
}
```

**Output Schema**:
```json
{
  "period": "string",
  "start_date": "timestamp",
  "end_date": "timestamp",
  "total_sessions": "integer",
  "avg_token_reduction_pct": "float",
  "total_time_saved_min": "float",
  "search_precision": "float",
  "estimated_cost_save_usd": "float",
  "top_features": [
    {
      "feature": "string",
      "count": "integer",
      "avg_latency_ms": "float",
      "success_rate": "float"
    }
  ],
  "performance": {
    "avg_search_latency_ms": "float",
    "avg_checkpoint_latency_ms": "float",
    "cache_hit_rate": "float",
    "overall_success_rate": "float"
  }
}
```

**Timeout**: 30 seconds

---

## Summary

**Tool Categories**:
- **Session Management** (3 tools): checkpoint_save, checkpoint_search, checkpoint_list
- **Error Resolution** (2 tools): remediation_save, remediation_search
- **AI Diagnosis** (2 tools): troubleshoot, list_patterns
- **Repository Indexing** (1 tool): index_repository
- **Skills Management** (6 tools): skill_create, skill_search, skill_list, skill_update, skill_delete, skill_apply
- **System Operations** (2 tools): status, analytics_get

**Total**: 16 MCP tools providing complete contextd functionality.
