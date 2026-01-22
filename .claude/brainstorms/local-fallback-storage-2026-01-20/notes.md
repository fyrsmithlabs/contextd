# Interview Notes - Local Fallback Storage

**Date:** 2026-01-20
**Topic:** Local fallback when remote vector store unavailable
**Tier:** STANDARD (11/15)

---

## Complexity Assessment

| Dimension | Score | Notes |
|-----------|-------|-------|
| Scope | 3/3 | New files, factory changes, config, sync logic |
| Integration | 2/3 | Store interface, health checks, background sync |
| Infrastructure | 2/3 | Local storage, connection monitoring, goroutines |
| Knowledge | 2/3 | Eventual consistency, sync semantics, conflicts |
| Risk | 2/3 | Data consistency, sync failures |

**Total: 11/15 → STANDARD**

---

## Interview Responses

### Q: Primary Goal
**A:** All of the above (Resilience, Disconnected usage, Cost optimization, Dev convenience)

### Q: Constraints
**A:** All selected:
- No data loss
- Backwards compatible
- Minimal dependencies
- Transparent to callers

### Q: Sync Timing
**A:** Immediate background sync when connection restores

### Q: Conflict Resolution
**A:** Local wins (recommended)

### Q: Health Detection
**A:** All of the above → Clarified to: gRPC state primary, periodic fallback

### Q: Storage Path
**A:** `.claude/contextd/store` (project-local, as originally specified)

### Q: Operations Buffered
**A:** All operations (writes + searches)

### Q: Offline Search
**A:** Local search (serve from local store when remote unavailable)

### Q: Enablement
**A:** Opt-in via config (recommended)

### Q: Implementation Approach
**A:** Decorator wrapper (recommended)

---

## Codebase Context

### Existing Architecture
- `Store` interface in `internal/vectorstore/interface.go`
- Factory pattern in `internal/vectorstore/factory.go`
- Two implementations: ChromemStore (embedded), QdrantStore (external)
- Tenant isolation via context

### Key Files Reviewed
- `internal/vectorstore/interface.go` - Store interface (~200 lines, 15 methods)
- `internal/vectorstore/factory.go` - Provider factory
- `internal/qdrant/client.go` - Qdrant client interface
- `docs/spec/vector-storage/architecture.md` - Existing architecture docs

### Relevant Patterns Found
- Provider-agnostic Store interface
- Tenant context via `ContextWithTenant()`
- Isolation modes (Payload, Filesystem, None)
- Gob persistence in ChromemStore

---

## Open Items from Interview

1. User requested "all of the above" for health detection - clarified to layered approach
2. User confirmed local-first write pattern (always write local, then remote)
3. User wants full offline capability (not just write buffering)
4. Consensus review requested before GitHub Issues

---

## Next Steps

1. ✅ Write design document
2. ⏳ Run consensus review (up to 3x)
3. ⏳ Create GitHub Issues
4. ⏳ Offer implementation options
