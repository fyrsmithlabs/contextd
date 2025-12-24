# Vector Storage Architecture

## System Context

```
┌─────────────────────────────────────────────────────────────────────┐
│                        contextd Server                               │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │MemoryManager │  │  Distiller   │  │ CodeIndexer  │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         └─────────────────┼─────────────────┘                        │
│                           │                                          │
│                  ┌────────▼────────┐                                │
│                  │  Store Interface │ ← Provider-agnostic            │
│                  └────────┬────────┘                                │
│                           │                                          │
│              ┌────────────┴────────────┐                            │
│              ▼                         ▼                            │
│     ┌────────────────┐       ┌────────────────┐                    │
│     │  ChromemStore  │       │  QdrantStore   │                    │
│     │  (Default)     │       │  (External)    │                    │
│     └───────┬────────┘       └───────┬────────┘                    │
└─────────────┼────────────────────────┼──────────────────────────────┘
              │ Gob files              │ gRPC
              ▼                        ▼
┌──────────────────────────┐  ┌─────────────────────────────────────┐
│  ~/.config/contextd/     │  │      Qdrant (Local or Cloud)        │
│     vectorstore/         │  │  ┌─────────┐ ┌─────────┐ ┌───────┐ │
│  (Embedded, 384d)        │  │  │ org_a   │ │ org_b   │ │ org_c │ │
└──────────────────────────┘  │  └─────────┘ └─────────┘ └───────┘ │
                              └─────────────────────────────────────┘
```

## Provider Selection

| Config | Provider | Storage | Embeddings |
|--------|----------|---------|------------|
| `provider: chromem` | ChromemStore | Gob files (embedded) | FastEmbed (ONNX) |
| `provider: qdrant` | QdrantStore | External Qdrant | FastEmbed/TEI (ONNX) |

**Default**: Chromem for zero-config local development (`brew install contextd` just works).
**Migration**: Use `ctxd migrate --qdrant-collection=all` to migrate from Qdrant to Chromem.

## Multi-Tenant Routing

**Flow**: Session → Tenant Context → Database Selection

```go
// 1. Session middleware validates and sets tenant (entry point)
func SessionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session := validateSession(r)       // Validates session_id
        tenant := session.Tenant            // Trusted, from session store
        ctx := vectordb.WithTenant(r.Context(), tenant)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 2. Client extracts tenant from context (cannot be overridden)
func (c *Client) Search(ctx context.Context, req *SearchRequest) {
    tenant := vectordb.TenantFromContext(ctx)
    if tenant == nil {
        return nil, ErrMissingTenant  // Fail closed
    }
    db := tenant.Database()  // org_id → database name
    // Route to tenant's database
}
```

**Key Properties**:
- Tenant set once at validated session boundary
- Downstream code cannot override tenant
- Missing tenant = error (fail closed)
- Database name validated (defense-in-depth)

## Codebase Indexing Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Indexing Pipeline                              │
│                                                                      │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │
│  │ go-git  │───►│ Parser  │───►│ Textify │───►│ Qdrant  │         │
│  │ (diff)  │    │(tree-   │    │ (NLP    │    │ (upsert │         │
│  │         │    │ sitter) │    │  prep)  │    │  docs)  │         │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘         │
│                                                                      │
│  Git Events:                              Index Key:                 │
│  - Ref changes (commit, checkout)         {team}_{project}_codebase_ │
│  - 10 min poll (unstaged)                 {worktree_hash}            │
└─────────────────────────────────────────────────────────────────────┘
```

**Index Identity**:
- Per-branch with delta updates
- Per-worktree isolation (worktree_hash)
- Key: `{team}_{project}_codebase_{worktree_hash}`

**Delta Detection**:
- `git diff {last_sha}..HEAD` for committed changes
- `git status` for uncommitted/unstaged changes
- Combined = files to re-index

## Embedding Strategy

| Scenario | Model | Notes |
|----------|-------|-------|
| Local Qdrant | `qdrant/bm25` | Sparse vectors, no token limit |
| Qdrant Cloud | `sentence-transformers/all-minilm-l6-v2` | Dense 384-dim |
| Large functions | `qdrant/bm25` | Fallback when >512 tokens |

**Dual Embedding** (per Qdrant tutorial):
1. NLP model on "textified" code (signature + docstring + context)
2. Code model on raw code snippet
3. Search merges results from both
