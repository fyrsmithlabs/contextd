# Interface Migration to Shared Packages

Move interface definitions from internal/remediation/interfaces.go to shared packages (internal/vectorstore, internal/embeddings) as noted in the TODO comments. Improves code organization and enables reuse.

## Rationale
Current interface locations violate package boundaries (TODOs in code). This technical debt makes the codebase harder to maintain and understand. Clean architecture enables faster feature development. Addresses technical debt from code TODOs.

## User Stories
- As a contributor, I want interfaces in logical locations so that I can find and understand them
- As a maintainer, I want clean package boundaries so that changes don't cause unexpected side effects

## Acceptance Criteria
- [x] VectorStore interface lives in internal/vectorstore package
- [x] EmbeddingProvider interface lives in internal/embeddings package
- [x] All imports are updated across the codebase
- [x] No duplicate interface definitions remain
- [x] Backward compatibility is maintained

## Completion Notes (2026-01-12)

The interfaces had already been migrated to their proper locations:
- `Store` interface: `internal/vectorstore/interface.go`
- `Embedder` interface: `internal/vectorstore/interface.go` (used by embeddings via `vectorstore.Embedder`)
- `Provider` interface: `internal/embeddings/provider.go` (extends `vectorstore.Embedder`)

The cleanup task deleted dead code that was left behind:
- Deleted `internal/remediation/interfaces.go` - contained duplicate `Embedder` and `QdrantClient` interfaces with TODO comments but no consumers
- Deleted `internal/qdrant/adapter.go` - `RemediationAdapter` implemented `remediation.QdrantClient` but had no consumers
- Deleted `internal/embeddings/adapter.go` - `RemediationEmbedder` implemented `remediation.Embedder` but had no consumers

The `internal/qdrant/client.go` already had its own types (`Point`, `ScoredPoint`, `Filter`, `Condition`, `RangeCondition`), and the remediation service used `vectorstore.Store` directly.
