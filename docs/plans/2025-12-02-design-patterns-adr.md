# Architecture Decision Record: Hook Lifecycle Patterns

**Status**: Accepted
**Created**: 2025-12-02
**Context**: Designing hook lifecycle for contextd alpha

---

## ADR-001: Service Registry Interface Pattern

### Context

contextd has multiple consumers (MCP server, HTTP server) that need access to the same services (checkpoint, remediation, reasoningbank, hooks, distiller). The original approach passed individual services to each constructor:

```go
// Before: many parameters, tight coupling
func NewMCPServer(
    checkpoint checkpoint.Service,
    remediation remediation.Service,
    reasoningbank *reasoningbank.Service,
    // ... 6+ more
) *Server
```

### Decision

Use a **Registry interface** that provides accessor methods for all services:

```go
type Registry interface {
    Checkpoint() checkpoint.Service
    Remediation() remediation.Service
    Memory() *reasoningbank.Service
    Hooks() *hooks.HookManager
    Distiller() *reasoningbank.Distiller
    Scrubber() secrets.Scrubber
}
```

Call pattern: `registry.Checkpoint().Save(ctx, input)`

### Why This Pattern

1. **Idiomatic Go**: Interfaces defined by consumer, accept interfaces return structs
2. **Single mock for tests**: Mock Registry once, not 6+ individual services
3. **Clear dependency boundary**: main.go builds registry, passes everywhere
4. **Extensible**: Add new services without changing constructor signatures
5. **Discoverable**: IDE autocomplete shows available services

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Pass individual services | Constructor bloat, signature changes cascade |
| Concrete struct registry | Can't mock for tests, violates interface principle |
| Consumer-defined interfaces | Interface explosion, each consumer redeclares |
| Global singleton | Untestable, hidden dependencies |

### Consequences

- All service access goes through registry accessor methods
- New services require adding to Registry interface
- Tests mock Registry interface, not individual services

---

## ADR-002: Claude-Generated Session Summaries

### Context

The Distiller needs a `SessionSummary` with task, approach, outcome, and tags to extract learnings. Three options for generating this:

1. Claude generates it (most accurate, requires prompting)
2. contextd infers it (automatic, but lossy)
3. Hybrid (contextd tracks, Claude provides outcome)

### Decision

**Claude-generated** with **skill enforcement** and **Claude Code hook backstop**.

### Why This Approach

1. **Accuracy**: Only the agent knows what it was trying to do and whether it succeeded
2. **Skills minimize "when it works" gap**: If skill says "call session_end before exiting", Claude does it
3. **Structured input**: Required fields (task/approach/outcome/tags) ensure quality
4. **Backstop exists**: PreCompact hook prompts if skill doesn't fire

### The Hybrid Pitfalls

We explicitly rejected hybrid because:

- **Incomplete picture**: contextd only sees tool calls, not reasoning
- **Timing mismatch**: Process may terminate before MCP call completes
- **Context chicken-and-egg**: At context limit, Claude can't generate quality summary
- **Attribution ambiguity**: Claude says "success" but contextd tracked failures - which wins?

### Consequences

- Depends on agent cooperation (skills enforce this)
- Summary quality varies with agent capability
- Backup mechanisms needed (Claude Code hooks)

---

## ADR-003: HTTP Primary, MCP Fallback for Threshold

### Context

Claude Code knows context percentage; contextd doesn't. Need to notify contextd when threshold reached.

### Decision

**HTTP endpoint primary**, MCP tool fallback, **configurable**.

```yaml
hooks:
  context_threshold:
    method: "http"      # http | mcp | both
    fallback: true      # try mcp if http fails
```

### Why HTTP Primary

1. **Reliability**: Shell script + curl has no LLM in the loop
2. **Speed**: Direct HTTP call vs. MCP message parsing
3. **Independence**: Works even if MCP server is busy/blocked
4. **Simplicity**: `curl POST /api/v1/threshold` - done

### Why MCP Fallback

1. **Ecosystem consistency**: Keeps everything in tool namespace
2. **Agent visibility**: Agent sees the tool call in its context
3. **Graceful degradation**: If HTTP server down, MCP still works

### Consequences

- HTTP server must have service access (Registry pattern enables this)
- Two code paths to maintain (HTTP handler + MCP tool)
- Configuration adds complexity but flexibility

---

## ADR-004: Prompt-Then-Prime for Session Start

### Context

On session start, should we:
1. Auto-resume from last checkpoint (automatic)
2. Prompt user to resume (interactive)
3. Just prime memories (minimal)

### Decision

**Prompt for checkpoint resume** → **then memory prime**.

### Why Prompt Instead of Auto-Resume

1. **User control**: User may want fresh start, not continuation
2. **Context awareness**: Checkpoint may be stale or irrelevant
3. **Transparency**: User knows what context is being loaded
4. **Safety**: Don't pollute context with outdated information

### Why Memory Prime After

1. **Always useful**: Relevant memories help even for new tasks
2. **Low cost**: Top 3 memories is small context footprint
3. **Discovery**: Surfaces forgotten learnings

### Flow

```
session_start called
  └─> Check for recent checkpoints
      └─> If found: return summary, agent asks "Resume from X?"
          └─> User confirms: checkpoint_resume
          └─> User declines: skip
  └─> Memory prime (always)
      └─> Search memories for project
      └─> Return top 3 relevant
```

### Consequences

- Extra round-trip for resume confirmation
- Agent must handle the prompt (skill teaches this)
- Better user experience than surprising context injection

---

## Pattern: Living the Problem You're Solving

### Observation

This design session itself demonstrated contextd's value proposition. We:

1. Re-discovered the skills system spec that existed from a previous session
2. Re-discussed hook design that was partially covered before
3. Spent time recovering context that should have been persisted

### Lesson

**The meta-test for contextd**: After implementing this design, the next session about contextd should immediately surface this ADR and the design doc via `memory_search`.

If an agent asks "how should I design service access in contextd?", the answer should come from memory, not re-derivation.

### Application

When designing systems, ask: "If I had to explain this decision in 6 months, what would I need to remember?" Write that down. Record it as a memory. Make it searchable.

---

## References

- Design doc: `docs/plans/2025-12-02-hook-lifecycle-design.md`
- Skills spec: `docs/specs/skills-system.md`
- Hooks package: `internal/hooks/hooks.go`
- Distiller: `internal/reasoningbank/distiller.go`
