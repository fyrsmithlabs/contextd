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
| Multi-tenant | Payload-based filtering (default) or filesystem isolation |
| Isolation | PayloadIsolation (default), FilesystemIsolation, NoIsolation |
| Security | Fail-closed: ErrMissingTenant if no tenant context |
| Client wrapper | Interface-based (provider-agnostic) |
| Code parsing | Repository service with semantic indexing |
| Index scope | Per-repository with tenant isolation |
| Telemetry | OpenTelemetry instrumentation on all operations |

## Provider Comparison

| Feature | chromem (Default) | Qdrant |
|---------|-------------------|--------|
| Installation | `brew install contextd` - just works | External service required |
| Storage | `~/.config/contextd/vectorstore` (gob files) | External Qdrant server |
| Dependencies | Pure Go, zero CGO | Qdrant server + gRPC |
| Embeddings | FastEmbed (local ONNX) | FastEmbed (local ONNX) |
| Embedding Source | Local embedder (both use FastEmbed) | Local embedder (both use FastEmbed) |
| Default Dims | 384 | 384 |
| Performance | 1K docs in 0.3ms, 100K in 40ms | Better for millions of docs |
| Use Case | Local dev, simple setups, `brew install` | Production, high scale |
| Persistence | Automatic gob files with optional gzip | External DB manages |
| Isolation | PayloadIsolation or FilesystemIsolation | PayloadIsolation (recommended) |

## Package Structure

```
internal/
├── vectorstore/        # Vector storage implementations
│   ├── interface.go    # Store interface
│   ├── chromem.go      # chromem (embedded) implementation
│   ├── qdrant.go       # Qdrant (gRPC) implementation
│   ├── factory.go      # Provider factory
│   ├── isolation.go    # Tenant isolation modes
│   ├── tenant.go       # Tenant context helpers
│   └── filter.go       # Filter utilities
│
└── repository/         # Repository indexing & semantic search
    ├── service.go      # Main service logic
    ├── adapter.go      # Indexing adapter
    └── types.go        # Core types
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
| FR-Q01 | gRPC connection with TLS support | P1 |
| FR-Q02 | Payload-based tenant isolation | P1 |
| FR-Q03 | Fail closed on missing tenant | P1 |
| FR-Q04 | Local embedding via FastEmbed | P1 |
| FR-Q05 | Collection + point CRUD | P1 |
| FR-Q06 | Circuit breaker and retry logic | P1 |
| FR-Q07 | Configurable message size limits | P1 |

### Repository Indexer

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-R01 | Semantic code search with grep fallback | P1 |
| FR-R02 | Repository indexing via MCP tools | P1 |
| FR-R03 | Tenant-isolated repository access | P1 |
| FR-R04 | Chunked document processing | P1 |
| FR-R05 | Metadata tracking (file path, repo info) | P1 |
| FR-R06 | Integration with vectorstore abstraction | P1 |

### Performance

| Metric | Target |
|--------|--------|
| Search latency | <50ms (100K points) |
| Index update | <5s (100 changed files) |
| Connection setup | <1s with TLS |

## Detailed Documentation

@./security.md - Tenant isolation (PayloadIsolation, FilesystemIsolation), credential protection, validation

## Multi-Tenant Isolation

contextd uses **payload-based tenant isolation** as the default strategy:

| Mode | Description | Use Case |
|------|-------------|----------|
| `PayloadIsolation` | Single collection with metadata filtering | **Default, recommended** |
| `FilesystemIsolation` | Separate database per tenant/project | Legacy, migration available |
| `NoIsolation` | No tenant filtering | **Testing only** |

### Tenant Context

All operations require tenant context via Go's `context.Context`:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Create tenant-scoped context (REQUIRED)
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required
    TeamID:    "platform",     // Optional
    ProjectID: "contextd",     // Optional
})

// All operations automatically filtered by tenant
results, err := store.Search(ctx, "query", 10)
```

### Security Guarantees

| Behavior | Description |
|----------|-------------|
| **Fail-closed** | Missing tenant context returns `ErrMissingTenant` |
| **Filter injection blocked** | User-provided tenant filters rejected with `ErrTenantFilterInUserFilters` |
| **Metadata enforced** | Tenant fields always set from context, never from user input |

See @./security.md for complete details.

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
    # isolation: payload              # Default: PayloadIsolation

  # Qdrant Configuration (external service)
  qdrant:
    host: ${QDRANT_HOST:-localhost}
    port: ${QDRANT_PORT:-6334}        # gRPC port (NOT 6333 HTTP)
    api_key: ${QDRANT_API_KEY}
    use_tls: ${QDRANT_TLS:-false}
    vector_size: 384                  # Default for FastEmbed
    max_message_size: 52428800        # 50MB for large documents
    # isolation: payload              # Default: PayloadIsolation

# Embeddings (shared by all providers - both chromem and Qdrant use local FastEmbed)
embeddings:
  provider: fastembed                 # fastembed (local ONNX) or tei (external)
  model: BAAI/bge-small-en-v1.5       # 384 dims (default)
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_VECTORSTORE_PROVIDER` | `chromem` | Provider selection (`chromem` or `qdrant`) |
| `CONTEXTD_VECTORSTORE_CHROMEM_PATH` | `~/.config/contextd/vectorstore` | chromem storage directory |
| `CONTEXTD_VECTORSTORE_CHROMEM_COMPRESS` | `true` | Enable gzip compression |
| `QDRANT_HOST` | `localhost` | Qdrant host |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port (NOT 6333 HTTP) |
| `QDRANT_API_KEY` | - | Qdrant API key (optional) |
| `QDRANT_TLS` | `false` | Enable TLS for Qdrant connection |

### Supported Embedding Models

| Model | Dimensions | Notes |
|-------|------------|-------|
| `BAAI/bge-small-en-v1.5` | 384 | Default, fast, good quality |
| `BAAI/bge-base-en-v1.5` | 768 | Higher quality, slower |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | Alternative |

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| chromem provider | ✅ Complete | Default embedded database |
| Qdrant provider | ✅ Complete | gRPC client with retry/circuit breaker |
| PayloadIsolation | ✅ Complete | Default tenant isolation mode |
| FilesystemIsolation | ✅ Complete | Legacy mode for backward compatibility |
| FastEmbed integration | ✅ Complete | Local ONNX embeddings for both providers |
| Repository indexing | ✅ Complete | Semantic search with grep fallback |
| MCP tools | ✅ Complete | All tools registered and functional |
| Security tests | ✅ Complete | Multi-tenant isolation verified |

## Acceptance Criteria

- [x] chromem embedded database with gob persistence
- [x] Qdrant gRPC connection with TLS support
- [x] Payload-based tenant isolation (fail closed)
- [x] FastEmbed local embeddings for both providers
- [x] Repository semantic search with grep fallback
- [x] Multi-tenant isolation tests pass
- [x] Filter injection prevention (ErrTenantFilterInUserFilters)
- [x] OpenTelemetry instrumentation
- [x] Test coverage ≥80%

## Dependencies

**External**:
- `github.com/philippgille/chromem-go` - Embedded vector database (default)
- `github.com/qdrant/go-client` - Qdrant gRPC client (optional)
- FastEmbed SDK - Local ONNX embeddings
- OpenTelemetry - Instrumentation and tracing

**Internal**:
- `internal/config` - Koanf configuration
- `internal/logging` - Zap structured logging
- `internal/embeddings` - FastEmbed provider
- `internal/repository` - Repository indexing and semantic search

## References

- [chromem-go](https://github.com/philippgille/chromem-go) - Embedded vector database
- [Qdrant Go Client](https://github.com/qdrant/go-client) - Official gRPC client
- [Security Spec](./security.md) - Multi-tenant isolation and security
- [ReasoningBank Spec](../reasoning-bank/SPEC.md) - Memory service using vectorstore
