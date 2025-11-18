# Feature: Checkpoint System

**Version**: 2.0.0
**Status**: Production
**Last Updated**: 2024-11-04

---

## Overview

The checkpoint system provides session state management and semantic search capabilities for Claude Code workflows. It enables users to save development session snapshots with rich metadata, then retrieve them later using natural language queries.

**Purpose**: Save and retrieve development session context across time boundaries

**Key Benefit**: Context recovery and knowledge reuse through semantic search

---

## Quick Reference

**Technology Stack**:
- Package: `pkg/checkpoint`
- Storage: Qdrant (vector database)
- Embeddings: OpenAI API or TEI (local)
- Protocol: MCP tools (checkpoint_save, checkpoint_search, checkpoint_list)
- Isolation: Database-per-project

**Core Capabilities**:
- Session snapshot creation with automatic embeddings
- Semantic search using natural language
- Paginated listing with project scoping
- CRUD operations (Get, Delete)
- Token tracking for cost monitoring

**Performance Targets**:
- Create: 55-160ms
- Search: 60-270ms (cached: 20-100ms)
- List: <50ms (small datasets)
- Get by ID: <15ms

**Multi-Tenant Model**:
- Database-per-project physical isolation
- Format: `project_<hash>` where hash = SHA256(project_path)[:16]
- Eliminates filter injection attacks
- 10-16x faster queries via partition pruning

---

## Detailed Documentation

**Requirements & Design**:
@./checkpoint/requirements.md - Functional & non-functional requirements, data model, testing
@./checkpoint/architecture.md - Component design, multi-tenant isolation, vector embedding, search algorithm

**Implementation & Usage**:
@./checkpoint/workflows.md - Create, search, list, delete workflows with examples
@./checkpoint/api-reference.md - MCP tools, service API, data models, error formats
@./checkpoint/implementation.md - Current implementation status, testing strategy, future enhancements

---

## Design Principles

1. **Local-First Performance** - Instant writes to project-specific databases
2. **Security by Isolation** - Database-per-project eliminates filter injection
3. **Context Optimization** - Efficient storage minimizes token usage
4. **Semantic Retrieval** - Natural language search over keyword matching
5. **Observability** - Full OpenTelemetry instrumentation

---

## Architecture Summary

### Component Flow

```
MCP Tools → Checkpoint Service → Vector Store + Embedding Service
             ├─ Database scoping (multi-tenant)
             ├─ Embedding generation & caching
             └─ OpenTelemetry instrumentation
```

### Multi-Tenant Isolation

Each project gets dedicated database (`project_<hash>`):
- No shared collections between projects
- Physical isolation prevents data leakage
- Database boundary enforced at infrastructure level
- Audit trail per project

### Embedding Strategy

**Content Preparation**: `summary + "\n\n" + description`
**Providers**: OpenAI (1536d) or TEI (384d)
**Caching**: 15-minute LRU cache (30-40% hit ratio)
**Token Counting**: Word-based approximation (±20% accuracy)

---

## Usage Examples

### Create Checkpoint (MCP)

```json
{
  "name": "checkpoint_save",
  "arguments": {
    "summary": "Implemented user authentication",
    "description": "Added JWT-based auth with refresh tokens",
    "project_path": "/home/user/myproject",
    "tags": ["auth", "security"]
  }
}
```

### Search Checkpoints (MCP)

```json
{
  "name": "checkpoint_search",
  "arguments": {
    "query": "how did I implement authentication?",
    "project_path": "/home/user/myproject",
    "top_k": 5
  }
}
```

---

## Current Status

**Production Features**:
- ✅ Create with automatic embeddings
- ✅ Semantic search with tag filtering
- ✅ Paginated listing
- ✅ Get by ID
- ✅ Delete
- ✅ Multi-tenant isolation
- ✅ OpenTelemetry instrumentation

**Not Yet Implemented**:
- ❌ Update operation (planned: delete + re-insert)
- ❌ Sorting in List (planned: sort_by parameter)
- ❌ Date range filters (planned: created_after/before)

**Test Coverage**: ≥80% (100% for critical paths)

---

## Security Considerations

**Multi-Tenant Isolation**: Database-per-project eliminates filter injection
**Input Validation**: Summary (1-500 chars), path traversal prevention, tag limits
**Data Redaction**: API keys, passwords, tokens automatically redacted
**Rate Limiting**: 10 req/min save, 20 req/min search (configurable)
**No Auth (MVP)**: Trusted network model, add authentication post-MVP

---

## Next Steps

**Phase 1** (Core Improvements):
- Implement Update operation
- Add sorting to List operation
- Add date range filters

**Phase 2** (Performance):
- Streaming search results
- Batch checkpoint creation
- Compression for large descriptions

**Phase 3** (Collaboration):
- Shared checkpoints across projects
- Multi-user authentication and RBAC
- Checkpoint templates

---

## Summary

Checkpoint system provides semantic search over development session snapshots with database-per-project isolation for security and performance. Production-ready with ≥80% test coverage. Uses vector embeddings (OpenAI/TEI) for natural language search.

**Status**: Production (v2.0.0+)
**Package**: `pkg/checkpoint`
**Documentation**: See @imports above for detailed specifications

---

**Document Version**: 2.0.0
**Authors**: Claude Code (claude.ai/code)
