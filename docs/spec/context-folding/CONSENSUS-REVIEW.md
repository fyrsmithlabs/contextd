# Context-Folding Consensus Design Review

**Date**: 2025-12-13
**Status**: REQUIRES CHANGES BEFORE IMPLEMENTATION
**Overall Verdict**: REQUEST CHANGES

---

## Executive Summary

| Reviewer | Verdict | Critical | High | Medium | Low |
|----------|---------|----------|------|--------|-----|
| Security | CONCERNS | 1 | 4 | 5 | 2 |
| Correctness | CONCERNS | 1 | 3 | 6 | 2 |
| Performance | CONCERNS | 0 | 3 | 5 | 2 |
| Architecture | CONCERNS | 2 | 3 | 3 | 0 |
| **TOTAL** | | **4** | **13** | **19** | **6** |

---

## Critical Findings (Must Fix Before Implementation)

### CRITICAL-1: Secret Leakage via Branch Returns
**Agreed by:** Security

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:76-88 |
| Issue | `return(message)` allows arbitrary content back to parent without secret scrubbing |
| Impact | Secrets can leak from branches into logs, checkpoints, and memories |
| Fix | ALL `return(message)` content MUST pass through gitleaks secret scrubbing |

### CRITICAL-2: Orphaned Child Branches on Parent Return
**Agreed by:** Correctness, Architecture

| Aspect | Detail |
|--------|--------|
| Location | ARCH.md:139-161, DESIGN.md:276-282 |
| Issue | No specification for what happens when parent returns while children are active |
| Impact | Resource leaks, crashes, budget tracking errors |
| Fix | Check for active children before return; force-return children recursively first |

### CRITICAL-3: Weak Context Isolation Model
**Agreed by:** Architecture, Security

| Aspect | Detail |
|--------|--------|
| Location | ARCH.md:196-200, RESEARCH.md:192-199 |
| Issue | No architectural mechanism for branch isolation (subprocess vs. goroutine deferred) |
| Impact | Entire branching concept unimplementable without this decision |
| Fix | Make explicit decision NOW: goroutine with strict data separation |

### CRITICAL-4: BudgetTracker/BranchManager Circular Dependency
**Agreed by:** Architecture

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:210-234, ARCH.md:65-76 |
| Issue | BudgetTracker directly calls BranchManager methods, creating circular dependency |
| Impact | Cannot unit test or instantiate components independently |
| Fix | Use event-driven pattern: BudgetTracker emits events, BranchManager subscribes |

---

## High Priority Findings (Block Merge)

### HIGH-1: Unvalidated User Input (Prompt Injection Risk)
**Agreed by:** Security, Correctness

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:19-26, SPEC.md:FR-001 |
| Issue | `description` and `prompt` fields accept arbitrary input without validation |
| Impact | Prompt injection, DoS via large inputs, downstream exploits |
| Fix | Add max length limits (500 chars / 2000 chars), sanitize control characters |

### HIGH-2: Race Condition Between Budget and Timeout
**Agreed by:** Correctness, Security

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:209-235 |
| Issue | Budget exhaustion and timeout can fire simultaneously, causing duplicate state transitions |
| Impact | Duplicate return messages, inconsistent state |
| Fix | State transition locking: acquire exclusive lock before ForceReturn, check status |

### HIGH-3: Concurrent Branch Token Consumption Race
**Agreed by:** Correctness, Performance

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:209-235, ARCH.md:182-193 |
| Issue | No locking on token consumption; concurrent tool calls can bypass budget check |
| Impact | Budget overruns, budget enforcement defeated |
| Fix | Atomic token consumption with mutex or compare-and-swap |

### HIGH-4: Memory Injection Synchronous Blocking
**Agreed by:** Performance

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:166-205 |
| Issue | Branch creation blocks on embedding + vector search + token counting (200-500ms) |
| Impact | Fails <100ms latency requirement (SC-002) |
| Fix | Async memory injection, return branch_id immediately with `injecting: true` |

### HIGH-5: No Authentication/Authorization Model
**Agreed by:** Security

| Aspect | Detail |
|--------|--------|
| Location | ARCH.md:200 |
| Issue | "JWT claims propagate to branches" mentioned but no design |
| Impact | Unauthorized users create branches, privilege escalation possible |
| Fix | Add auth section: validate JWT on branch/return, define required permissions |

### HIGH-6: Missing Token Counting Abstraction
**Agreed by:** Architecture, Correctness

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:189-190, RESEARCH.md:196-199 |
| Issue | Token counting strategy deferred as "open question" |
| Impact | Cannot implement budget enforcement without this decision |
| Fix | Define `TokenCounter` interface, choose server-side counting via proxy |

### HIGH-7: MCP Tool Protocol Fit Issues
**Agreed by:** Architecture

| Aspect | Detail |
|--------|--------|
| Location | DESIGN.md:7-98, RESEARCH.md:194-199 |
| Issue | MCP is request/response but branch() creates long-lived context; unclear how subsequent tools know active branch |
| Impact | Pattern may not fit MCP model |
| Fix | Add session context tracking: contextd maintains `active_branch_id` per session |

---

## Medium Priority Findings (Should Fix in Follow-up)

| # | Reviewer(s) | Issue | Fix |
|---|-------------|-------|-----|
| M1 | Security | Memory injection poisoning attacks | Validate memory content, track provenance |
| M2 | Security | No rate limiting on branch creation | Add configurable limits per session/instance |
| M3 | Security, Correctness | Orphan cleanup not fully specified | Define recursive cleanup with timeout |
| M4 | Correctness | Memory injection failure handling inconsistent | Return best-effort, log warnings, proceed |
| M5 | Correctness | Budget edge case at exactly limit | Use `>` not `>=`, add idempotent check |
| M6 | Correctness | Nested depth exceeded handling missing | Add validation, return clear error |
| M7 | Correctness | Session end cleanup partially specified | Define bounded-time synchronous cleanup |
| M8 | Correctness | Timeout implementation details missing | Specify goroutine with context.WithTimeout |
| M9 | Correctness | Parent budget deduction logic undefined | Choose pooled or independent, document |
| M10 | Performance | Token counting on every tool call (I/O) | In-memory caching with write-through |
| M11 | Performance | O(n) token counting during injection | Pre-compute and cache token counts |
| M12 | Performance | No connection pooling for vectorstore | Implement configurable pool |
| M13 | Performance | Nested budget accounting O(depth) | Single deduction at creation, reconcile on return |
| M14 | Performance | No GC for completed branches | Archive after 1 hour, delete after 24 |
| M15 | Architecture | Missing interface definitions | Define interfaces for all components |
| M16 | Architecture | Inconsistent error handling model | Define BranchError hierarchy |
| M17 | Architecture | Missing integration with existing services | Show service registry integration |
| M18 | Architecture | MemoryInjector violates SRP | Extract budget allocation to BudgetTracker |
| M19 | Architecture | Missing BranchRepository abstraction | Define persistence interface |

---

## Consensus Recommendations

### Immediate Actions (Before Implementation)

1. **Resolve isolation model**: Choose goroutine + strict data separation (CRITICAL-3)
2. **Add secret scrubbing to returns**: Use existing gitleaks integration (CRITICAL-1)
3. **Define parent-child lifecycle**: Recursive force-return before parent return (CRITICAL-2)
4. **Fix circular dependency**: Event-driven budget exhaustion (CRITICAL-4)
5. **Add input validation**: Max lengths, sanitization (HIGH-1)
6. **Implement concurrency controls**: Mutex on budget, state transitions (HIGH-2, HIGH-3)
7. **Define token counting interface**: Server-side proxy approach (HIGH-6)
8. **Add session context tracking**: active_branch_id per MCP session (HIGH-7)

### Design Updates Required

Update these documents before implementation:
- [ ] DESIGN.md: Add input validation, concurrency controls, token counter interface
- [ ] ARCH.md: Add isolation model decision, session context tracking, event-driven pattern
- [ ] SPEC.md: Add security requirements (scrubbing, auth, rate limits)

### TDD Focus Areas

Tests MUST validate these invariants:
1. **Secret scrubbing on return** - Test that secrets in return message are redacted
2. **Child cleanup before parent return** - Test orphan prevention
3. **Budget atomicity** - Test concurrent consumption respects limits
4. **Branch isolation** - Test that siblings cannot access each other's data
5. **Input validation** - Test rejection of oversized/malicious inputs
6. **Timeout enforcement** - Test forced return on timeout
7. **State machine transitions** - Test all valid/invalid transitions

---

## Record to Contextd

This review identified patterns that should be recorded for future reference.
