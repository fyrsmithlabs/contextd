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
| **Provider** | chromem (default, embedded) or Qdrant (external) |
| chromem | Pure Go, embedded, zero-config, gob persistence |
| Qdrant | gRPC on port 6334, external service |
| Embedding | FastEmbed (local ONNX) for both providers |
| Default Dims | 384 (bge-small-en-v1.5) |
| Multi-tenant | Context-based routing (from validated session) |
| Client wrapper | Interface-based (provider-agnostic) |
| Code parsing | tree-sitter (semantic units, not chunks) |
| Index scope | Per-branch, per-worktree, delta updates |
| Git integration | go-git ref watching + 10 min poll fallback |

## Provider Comparison

| Feature | chromem (Default) | Qdrant |
|---------|-------------------|--------|
| Installation | `brew install contextd` - just works | External service required |
| Storage | `~/.config/contextd/vectorstore/` (gob files) | External Qdrant server |
| Dependencies | Pure Go, zero CGO | Qdrant server + gRPC |
| Embeddings | FastEmbed (local ONNX) | FastEmbed (local ONNX) |
| Default Dims | 384 | 384 |
| Performance | 1K docs in 0.3ms, 100K in 40ms | Better for millions of docs |
| Use Case | Local dev, simple setups, `brew install` | Production, high scale |
| Persistence | Automatic gob files with optional gzip | External DB manages |

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

### Provider Interface

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-P01 | Provider-agnostic Store interface | P1 |
| FR-P02 | Factory pattern for provider creation | P1 |
| FR-P03 | Configuration-driven provider selection | P1 |
| FR-P04 | Telemetry for all providers | P1 |

### chromem Provider (Default)

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-C01 | Embedded gob-based storage (no external DB) | P1 |
| FR-C02 | Pure Go, zero CGO dependencies | P1 |
| FR-C03 | Automatic persistence to disk | P1 |
| FR-C04 | Optional gzip compression | P1 |
| FR-C05 | FastEmbed integration for embeddings | P1 |
| FR-C06 | OpenTelemetry instrumentation | P1 |

### Qdrant Provider

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
# Vector Store Provider Selection
vectorstore:
  provider: chromem                   # "chromem" (default) or "qdrant"

  # chromem Configuration (embedded, zero-config, pure Go)
  chromem:
    path: ~/.config/contextd/vectorstore  # Gob file storage directory
    compress: true                    # Enable gzip compression
    default_collection: contextd_default
    vector_size: 384                  # Must match embedder output

  # Qdrant Configuration (external service)
  qdrant:
    host: ${QDRANT_HOST:-localhost}
    port: ${QDRANT_PORT:-6334}
    api_key: ${QDRANT_API_KEY}
    use_tls: ${QDRANT_TLS:-false}
    vector_size: 384                  # Default for FastEmbed

# Embeddings (shared by all providers)
embeddings:
  provider: fastembed                 # fastembed or tei
  model: BAAI/bge-small-en-v1.5       # 384 dims

codeindex:
  enabled: true
  languages: [go, typescript, python, rust]
  large_function_threshold: 512
  watch:
    enabled: true
    poll_interval: 10m
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_VECTORSTORE_PROVIDER` | `chromem` | Provider selection |
| `CONTEXTD_VECTORSTORE_CHROMEM_PATH` | `~/.config/contextd/vectorstore` | chromem storage directory |
| `CONTEXTD_VECTORSTORE_CHROMEM_COMPRESS` | `true` | Enable gzip compression |
| `CONTEXTD_QDRANT_HOST` | `localhost` | Qdrant host |
| `CONTEXTD_QDRANT_PORT` | `6334` | Qdrant gRPC port |

### Supported Embedding Models

| Model | Dimensions | Notes |
|-------|------------|-------|
| `BAAI/bge-small-en-v1.5` | 384 | Default, fast, good quality |
| `BAAI/bge-base-en-v1.5` | 768 | Higher quality, slower |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | Alternative |

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
- `github.com/philippgille/chromem-go` - Embedded vector database (default)
- `github.com/qdrant/go-client` - Qdrant gRPC (optional)
- `github.com/go-git/go-git/v5` - Git operations
- `github.com/smacker/go-tree-sitter` - AST parsing

**Internal**:
- `internal/config` - Configuration
- `internal/logging` - Structured logging
- `internal/embeddings` - FastEmbed integration

## References

- [Qdrant Code Search Tutorial](https://qdrant.tech/documentation/advanced-tutorials/code-search/)
- [Collection Architecture Spec](../collection-architecture/SPEC.md)
- [ReasoningBank Spec](../reasoning-bank/SPEC.md)
