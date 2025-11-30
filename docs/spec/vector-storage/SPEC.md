---
title: Vector Storage
status: Draft
created: 2025-11-25
author: contextd team
version: 1.0.0
---

# Vector Storage Specification

## Overview

Vector storage infrastructure for contextd semantic search over memories, remediations, and codebase.

**Core Components**:
- **VectorDB Client** - Thin wrapper over Qdrant gRPC
- **Code Indexer** - AST-based codebase indexing with go-git

## Quick Reference

| Aspect | Decision |
|--------|----------|
| Vector DB | Qdrant (gRPC on port 6334) |
| Embedding | Qdrant built-in inference (no local ONNX) |
| Local model | `qdrant/bm25` (sparse, no token limit) |
| Cloud model | `sentence-transformers/all-minilm-l6-v2` (384 dims) |
| Multi-tenant | Context-based routing (from validated session) |
| Client wrapper | Thin (uses Qdrant types directly) |
| Code parsing | tree-sitter (semantic units, not chunks) |
| Index scope | Per-branch, per-worktree, delta updates |
| Git integration | go-git ref watching + 10 min poll fallback |

## Package Structure

```
internal/
├── vectordb/           # Qdrant client wrapper
│   ├── client.go       # Interface + implementation
│   ├── context.go      # Tenant context helpers
│   └── testing.go      # Mock client
│
└── codeindex/          # Codebase indexing
    ├── indexer.go      # Main indexing logic
    ├── parser.go       # tree-sitter AST parsing
    ├── textify.go      # Code → natural language
    ├── git.go          # go-git integration
    └── watcher.go      # Ref watcher + polling
```

## Requirements Summary

### VectorDB Client

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-Q01 | gRPC connection with TLS | P1 |
| FR-Q02 | Context-based tenant routing | P1 |
| FR-Q03 | Fail closed on missing tenant | P1 |
| FR-Q04 | Document-based inference (Qdrant embeds) | P1 |
| FR-Q05 | Collection + point CRUD | P1 |

### Code Indexer

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-I01 | AST extraction via tree-sitter | P1 |
| FR-I02 | Dual embedding (NLP + code model) | P1 |
| FR-I03 | BM25 for large functions (>512 tokens) | P1 |
| FR-I04 | Per-branch delta updates | P1 |
| FR-I05 | go-git ref watching | P2 |
| FR-I06 | `ctxd index` CLI command | P1 |

### Performance

| Metric | Target |
|--------|--------|
| Search latency | <50ms (100K points) |
| Index update | <5s (100 changed files) |
| Connection setup | <1s with TLS |

## Detailed Documentation

@./architecture.md - System design, multi-tenant routing, data flow
@./api.md - VectorDB client interface, code indexer interface
@./codeindex.md - AST parsing, textify, git integration, delta updates
@./security.md - Tenant isolation, credential protection, validation
@./testing.md - Unit tests, integration tests, security tests

## Configuration

```yaml
vectordb:
  host: ${QDRANT_HOST:-localhost}
  port: ${QDRANT_PORT:-6334}
  api_key: ${QDRANT_API_KEY}
  use_tls: ${QDRANT_TLS:-false}
  inference:
    model: ${QDRANT_MODEL:-qdrant/bm25}

codeindex:
  enabled: true
  languages: [go, typescript, python, rust]
  large_function_threshold: 512
  watch:
    enabled: true
    poll_interval: 10m
```

## Implementation Phases

| Phase | Scope | Duration |
|-------|-------|----------|
| 1 | VectorDB client + tenant routing | Week 1 |
| 2 | Code indexer core (Go only) | Week 2 |
| 3 | Git integration + watcher | Week 2-3 |
| 4 | CLI + session hook | Week 3 |
| 5 | Additional languages | Week 4 |

## Acceptance Criteria

- [ ] Qdrant gRPC connection with TLS
- [ ] Context-based tenant routing (fail closed)
- [ ] Document-based inference works
- [ ] tree-sitter extracts Go semantic units
- [ ] Delta indexing via git diff
- [ ] `ctxd index` CLI command
- [ ] Multi-tenant isolation tests pass
- [ ] Test coverage ≥80%

## Dependencies

**External**:
- `github.com/qdrant/go-client` - Qdrant gRPC
- `github.com/go-git/go-git/v5` - Git operations
- `github.com/smacker/go-tree-sitter` - AST parsing

**Internal**:
- `internal/config` - Configuration
- `internal/logging` - Structured logging

## References

- [Qdrant Code Search Tutorial](https://qdrant.tech/documentation/advanced-tutorials/code-search/)
- [Collection Architecture Spec](../collection-architecture/SPEC.md)
- [ReasoningBank Spec](../reasoning-bank/SPEC.md)
