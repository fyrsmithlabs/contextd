# Key Decisions - Local Fallback Storage

## Decision Log

### D1: Architecture Pattern
**Decision:** Decorator/wrapper pattern (FallbackStore wraps remote + local)
**Rationale:**
- Clean separation of concerns
- Testable in isolation
- No changes to existing Store implementations
- Transparent to callers

**Alternatives Considered:**
- Factory modification - couples factory to fallback logic
- Provider-level - tight coupling, hard to test

### D2: Sync Timing
**Decision:** Immediate background sync when connection restores
**Rationale:**
- Minimizes data staleness on remote
- User doesn't have to remember to sync
- Reduces risk of data loss if local fails

**Alternatives Considered:**
- Lazy on next write - delays sync unnecessarily
- Explicit command - user burden
- Configurable interval - unnecessary complexity

### D3: Conflict Resolution
**Decision:** Local wins (overwrite remote)
**Rationale:**
- Local changes are temporally newer
- Simple, predictable behavior
- No clock sync requirements
- Aligns with "offline-first" philosophy

**Alternatives Considered:**
- Remote wins - could lose user work
- Timestamp-based - clock sync complexity
- Merge - extremely complex, error-prone

### D4: Health Detection
**Decision:** gRPC state watcher (primary) + periodic ping (fallback)
**Rationale:**
- gRPC state is real-time and efficient
- Periodic ping catches cases gRPC misses
- Multiple strategies = more resilient

**Alternatives Considered:**
- Only periodic - slower detection
- Only gRPC state - might miss some failures
- On-demand only - delays detection

### D5: Storage Location
**Decision:** `.claude/contextd/store` (project-local)
**Rationale:**
- Visible to user, easy to inspect/delete
- Project-specific, no cross-contamination
- Consistent with `.claude/` conventions

**Alternatives Considered:**
- `~/.local/share/contextd/` - less visible
- Temp directory - lost on reboot
- Configurable - unnecessary complexity

### D6: Operations Buffered
**Decision:** All operations (writes + reads)
**Rationale:**
- Full offline functionality
- Better user experience
- Local store is complete working store

**Alternatives Considered:**
- Writes only - poor offline UX
- Writes + cache - partial solution
- Full mirror - expensive, out of scope

### D7: Enablement
**Decision:** Opt-in via config (`fallback.enabled: true`)
**Rationale:**
- Backwards compatible
- Users explicitly choose feature
- Can be changed to default-on later

**Alternatives Considered:**
- Always on for Qdrant - could surprise users
- CLI flag - not persistent
- Env var - not as discoverable

### D8: WAL Format
**Decision:** Gob-encoded files
**Rationale:**
- Already used by chromem
- No new dependencies
- Fast serialization
- Easy to debug (gob is readable)

**Alternatives Considered:**
- JSON - slower, larger
- SQLite - new dependency
- Custom binary - maintenance burden
