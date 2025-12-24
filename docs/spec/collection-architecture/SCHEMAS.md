# Collection Schemas

**Feature**: Collection Architecture
**Status**: Draft
**Created**: 2025-11-22

## Database Context

All collections exist within an organization's dedicated database:

```
Database: {org_id}
└── Collection: {scope_prefix}_{type}
    ├── id: UUID
    ├── vector: [float32 × dimensions]
    └── payload: { ... }
```

## Common Fields

All collections share base metadata in payload:

```json
{
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "created_by": "user_id"
}
```

---

## org_memories / {team}_memories / {team}_{project}_memories

Strategies and patterns from successful/failed sessions.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "description": "string (max 500)",
    "content": "string (max 5000)",
    "outcome": "success | failure | mixed",
    "confidence": "float (0.0 - 1.0)",
    "usage_count": "int",
    "tags": ["string"],
    "source_session": "string?",
    "promoted_from": ["memory_id"]?,
    "promoted_to": "memory_id?",
    "created_at": "timestamp",
    "last_used": "timestamp?"
  }
}
```

**Indexes**: `confidence`, `outcome`, `tags`, `created_at`, `last_used`

---

## org_remediations / {team}_remediations / {team}_{project}_remediations

Bug fixes and solutions with deduplication support.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "problem": "string (max 1000)",
    "symptoms": ["string (max 200)"],
    "root_cause": "string (max 1000)",
    "solution": "string (max 3000)",
    "code_diff": "string?",
    "affected_files": ["string"],
    "status": "canonical | merged | pending",
    "canonical_id": "uuid?",
    "occurrence_count": "int",
    "source_sessions": ["session_id"],
    "merged_from": ["remediation_id"]?,
    "confidence": "float (0.0 - 1.0)",
    "verified": "boolean",
    "created_at": "timestamp",
    "last_seen": "timestamp"
  }
}
```

**Indexes**: `status`, `confidence`, `verified`, `symptoms`, `created_at`

---

## org_policies

Security and compliance requirements.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "category": "security | compliance | process | other",
    "severity": "critical | high | medium | low",
    "requirement": "string (max 2000)",
    "enforcement": "string (max 500)",
    "tools": ["string"],
    "exceptions_process": "string?",
    "last_reviewed": "timestamp",
    "effective_date": "timestamp",
    "expires_at": "timestamp?"
  }
}
```

**Indexes**: `category`, `severity`, `effective_date`

---

## org_coding_standards / {team}_coding_standards

Coding conventions and style rules.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "language": "string",
    "category": "string",
    "rule": "string (max 1000)",
    "good_example": "string (max 2000)",
    "bad_example": "string (max 2000)",
    "rationale": "string (max 500)",
    "linter": "string?",
    "linter_rule": "string?",
    "severity": "error | warning | info",
    "auto_fixable": "boolean"
  }
}
```

**Indexes**: `language`, `category`, `severity`

---

## org_repo_standards

Repository structure and organization rules.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "type": "directory_structure | ci_cd | branching | other",
    "standard": "string (max 200)",
    "required_dirs": ["string"],
    "forbidden_dirs": ["string"],
    "required_files": ["string"],
    "rationale": "string (max 500)",
    "template_repo": "string?",
    "linter": "string?"
  }
}
```

**Indexes**: `type`, `standard`

---

## org_skills

Reusable agent capabilities and workflows.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "name": "string (max 50)",
    "description": "string (max 500)",
    "trigger": "string (max 200)",
    "steps": ["string (max 500)"],
    "prompt_template": "string (max 5000)",
    "required_tools": ["string"],
    "required_collections": ["string"],
    "success_rate": "float",
    "avg_tokens_saved": "int",
    "usage_count": "int",
    "active": "boolean"
  }
}
```

**Indexes**: `name`, `active`, `success_rate`

---

## org_agents

Agent configurations and personas.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "name": "string (max 50)",
    "description": "string (max 500)",
    "system_prompt": "string (max 10000)",
    "skills": ["skill_name"],
    "collections_access": ["collection_pattern"],
    "model_preference": "string",
    "temperature": "float (0.0 - 2.0)",
    "max_tokens": "int",
    "tools_enabled": ["string"],
    "active": "boolean"
  }
}
```

**Indexes**: `name`, `active`

---

## org_anti_patterns

Failure-derived warnings and things to avoid.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "title": "string (max 100)",
    "category": "security | performance | reliability | other",
    "what_happened": "string (max 1000)",
    "source_incident": "string?",
    "why_bad": "string (max 1000)",
    "instead": "string (max 1000)",
    "detection": "string (max 500)",
    "severity": "critical | high | medium | low",
    "occurrence_count": "int",
    "last_seen": "timestamp"
  }
}
```

**Indexes**: `category`, `severity`, `last_seen`

---

## org_feedback

Explicit user ratings and corrections.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "target_type": "memory | remediation | skill | agent",
    "target_id": "uuid",
    "user_id": "string",
    "rating": "helpful | not_helpful | neutral",
    "comment": "string (max 500)?",
    "session_context": "string (max 200)?",
    "created_at": "timestamp"
  }
}
```

**Indexes**: `target_type`, `target_id`, `rating`, `created_at`

---

## {team}_{project}_codebase

File and function embeddings for semantic code search.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "file_path": "string",
    "chunk_type": "file | function | class | block",
    "name": "string?",
    "content": "string (max 5000)",
    "language": "string",
    "line_start": "int",
    "line_end": "int",
    "git_sha": "string",
    "last_indexed": "timestamp"
  }
}
```

**Indexes**: `file_path`, `chunk_type`, `language`, `git_sha`

---

## {team}_{project}_sessions

Session traces for distillation.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "session_id": "string",
    "user_id": "string",
    "task_description": "string (max 500)",
    "outcome": "success | failure | partial | abandoned",
    "trace": "string (compressed)",
    "tokens_used": "int",
    "duration_seconds": "int",
    "branches_created": "int",
    "memories_injected": ["memory_id"],
    "distillation_status": "pending | processing | completed | skipped",
    "created_at": "timestamp",
    "ended_at": "timestamp"
  }
}
```

**Indexes**: `outcome`, `distillation_status`, `created_at`

---

## {team}_{project}_checkpoints

Saved context snapshots for session resumption.

```json
{
  "id": "uuid",
  "vector": [1536],
  "payload": {
    "session_id": "string",
    "user_id": "string",
    "summary": "string (max 1000)",
    "context_snapshot": "string (compressed)",
    "original_tokens": "int",
    "compressed_tokens": "int",
    "tags": ["string"],
    "created_at": "timestamp",
    "expires_at": "timestamp?"
  }
}
```

**Indexes**: `session_id`, `user_id`, `tags`, `created_at`
