# Context-Folding TDD Implementation Plan

**Issue**: [#17](https://github.com/fyrsmithlabs/contextd/issues/17)
**Branch**: `feature/context-folding-17`
**Status**: Ready for Implementation
**Created**: 2025-12-13

---

## Overview

This plan implements Context-Folding via `branch()` and `return()` MCP tools following TDD (Red-Green-Refactor). Each phase writes tests FIRST, then implementation.

## Architectural Decisions (from Consensus Review)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Isolation Model | Goroutine + strict data separation | Cheap, sufficient for context isolation |
| Token Counting | Server-side proxy via MCP interceptor | Cannot trust agent self-reporting |
| Event Pattern | BudgetTracker emits events | Breaks circular dependency |
| Session Tracking | `active_branch_id` per session in contextd | MCP is stateless, contextd manages state |
| Secret Scrubbing | gitleaks on return() content | Existing infrastructure |

---

## Phase 1: Core Types and Interfaces

**Goal**: Define types, interfaces, and repository abstractions.

### 1.1 Tests First (RED)

```
internal/folding/types_test.go
internal/folding/interfaces_test.go
```

**Test Cases**:
- [ ] Branch struct JSON serialization roundtrip
- [ ] BranchStatus state transitions (Created→Active, Active→Completed, etc.)
- [ ] Invalid state transitions rejected (Completed→Active should fail)
- [ ] BranchRequest validation (empty description rejected)
- [ ] BranchRequest validation (description > 500 chars rejected)
- [ ] BranchRequest validation (prompt > 10000 chars rejected)

### 1.2 Implementation (GREEN)

```
internal/folding/types.go      - Branch, BranchStatus, BranchRequest, BranchResponse
internal/folding/errors.go     - ErrMaxDepthExceeded, ErrBudgetExhausted, etc.
internal/folding/interfaces.go - BranchRepository, TokenCounter, EventEmitter
```

**Key Types**:
```go
type Branch struct {
    ID             string
    SessionID      string
    ParentID       *string
    Depth          int
    Description    string
    Prompt         string
    BudgetTotal    int
    BudgetUsed     int
    TimeoutSeconds int
    Status         BranchStatus
    Result         *string
    Error          *string
    InjectedMemoryIDs []string
    CreatedAt      time.Time
    CompletedAt    *time.Time
}

type BranchRepository interface {
    Create(ctx context.Context, branch *Branch) error
    Get(ctx context.Context, id string) (*Branch, error)
    Update(ctx context.Context, branch *Branch) error
    ListBySession(ctx context.Context, sessionID string) ([]*Branch, error)
    GetActiveBySession(ctx context.Context, sessionID string) (*Branch, error)
    Delete(ctx context.Context, id string) error
}

type TokenCounter interface {
    Count(content string) (int, error)
}

type BranchEvent interface {
    Type() string
}

type BudgetExhaustedEvent struct {
    BranchID   string
    BudgetUsed int
    BudgetTotal int
}

type EventEmitter interface {
    Emit(event BranchEvent)
    Subscribe(handler func(BranchEvent))
}
```

---

## Phase 2: Branch Repository (In-Memory)

**Goal**: Implement branch storage with concurrency safety.

### 2.1 Tests First (RED)

```
internal/folding/repository_test.go
```

**Test Cases**:
- [ ] Create and Get branch roundtrip
- [ ] Update branch status persists
- [ ] ListBySession returns only matching branches
- [ ] GetActiveBySession returns nil when no active branch
- [ ] Concurrent Create operations don't corrupt state
- [ ] Concurrent Update operations are atomic (test budget increment race)
- [ ] Delete removes branch from store
- [ ] Get non-existent returns ErrNotFound

### 2.2 Implementation (GREEN)

```
internal/folding/repository.go - MemoryBranchRepository
```

**Key Implementation**:
- Use `sync.RWMutex` for thread-safe access
- Implement `map[string]*Branch` with copy-on-read to prevent mutation
- Atomic budget updates with mutex

---

## Phase 3: Budget Tracker (Event-Driven)

**Goal**: Track token usage, emit events on exhaustion (no circular deps).

### 3.1 Tests First (RED)

```
internal/folding/budget_test.go
```

**Test Cases**:
- [ ] Allocate budget initializes tracking
- [ ] Consume tokens updates usage
- [ ] Consume exceeding budget emits BudgetExhaustedEvent
- [ ] Consume at 80% emits BudgetWarningEvent
- [ ] Concurrent Consume operations respect total (race test)
- [ ] Remaining returns correct value
- [ ] IsExhausted returns true when usage >= budget
- [ ] Deallocate removes tracking

### 3.2 Implementation (GREEN)

```
internal/folding/budget.go - BudgetTracker
```

**Key Implementation**:
```go
type BudgetTracker struct {
    mu       sync.Mutex
    budgets  map[string]*budgetState
    emitter  EventEmitter
}

type budgetState struct {
    total int
    used  int64 // atomic for lock-free reads
}

func (t *BudgetTracker) Consume(branchID string, tokens int) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    state := t.budgets[branchID]
    newUsed := state.used + int64(tokens)

    if newUsed > int64(state.total) {
        t.emitter.Emit(BudgetExhaustedEvent{BranchID: branchID, ...})
        return ErrBudgetExhausted
    }

    state.used = newUsed

    if float64(newUsed)/float64(state.total) > 0.8 {
        t.emitter.Emit(BudgetWarningEvent{BranchID: branchID, ...})
    }

    return nil
}
```

---

## Phase 4: Branch Manager

**Goal**: Orchestrate branch lifecycle, handle events.

### 4.1 Tests First (RED)

```
internal/folding/manager_test.go
```

**Test Cases**:
- [ ] Create branch with valid request returns branch_id
- [ ] Create branch at max_depth+1 returns ErrMaxDepthExceeded
- [ ] Create branch with empty description returns ErrInvalidInput
- [ ] Create branch increments parent's child count
- [ ] Return completes branch and sets result
- [ ] Return from root session returns ErrCannotReturnFromRoot
- [ ] Return with active children force-returns children first
- [ ] ForceReturn sets status to appropriate terminal state
- [ ] BudgetExhaustedEvent triggers ForceReturn
- [ ] TimeoutEvent triggers ForceReturn
- [ ] Session end force-returns all active branches
- [ ] GetActive returns current active branch for session

### 4.2 Implementation (GREEN)

```
internal/folding/manager.go - BranchManager
```

**Key Implementation**:
```go
type BranchManager struct {
    repo    BranchRepository
    budget  *BudgetTracker
    injector *MemoryInjector
    config  *FoldingConfig
}

func (m *BranchManager) Create(ctx context.Context, req BranchRequest) (*BranchResponse, error) {
    // Validate input
    if err := m.validateRequest(req); err != nil {
        return nil, err
    }

    // Check depth limit
    parent, _ := m.repo.GetActiveBySession(ctx, req.SessionID)
    depth := 0
    if parent != nil {
        depth = parent.Depth + 1
    }
    if depth >= m.config.MaxDepth {
        return nil, ErrMaxDepthExceeded
    }

    // Create branch
    branch := &Branch{
        ID:          generateBranchID(),
        SessionID:   req.SessionID,
        ParentID:    parentIDOrNil(parent),
        Depth:       depth,
        Description: req.Description,
        Prompt:      req.Prompt,
        BudgetTotal: req.Budget,
        Status:      BranchStatusActive,
        CreatedAt:   time.Now(),
    }

    // Allocate budget
    m.budget.Allocate(branch.ID, req.Budget)

    // Inject memories (async in background)
    go m.injectMemoriesAsync(ctx, branch)

    // Save and set as active
    if err := m.repo.Create(ctx, branch); err != nil {
        return nil, err
    }

    return &BranchResponse{
        BranchID:        branch.ID,
        BudgetAllocated: req.Budget,
    }, nil
}
```

---

## Phase 5: Memory Injector

**Goal**: Inject relevant memories with budget allocation.

### 5.1 Tests First (RED)

```
internal/folding/injector_test.go
```

**Test Cases**:
- [ ] Inject retrieves memories from ReasoningBank
- [ ] Inject respects injection budget (20% of branch budget)
- [ ] Inject stops when budget exhausted (greedy fit)
- [ ] Inject handles ReasoningBank errors gracefully (returns empty)
- [ ] Inject records injected memory IDs on branch
- [ ] Inject with empty description returns no memories

### 5.2 Implementation (GREEN)

```
internal/folding/injector.go - MemoryInjector
```

---

## Phase 6: Return Handler with Secret Scrubbing

**Goal**: Implement return() with mandatory secret scrubbing.

### 6.1 Tests First (RED)

```
internal/folding/return_test.go
```

**Test Cases (CRITICAL - validates secret scrubbing)**:
- [ ] Return scrubs secrets from message (AWS_SECRET_KEY redacted)
- [ ] Return scrubs GitHub tokens from message
- [ ] Return scrubs API keys from message
- [ ] Return passes clean content unchanged
- [ ] Return with active children force-returns children first
- [ ] Return updates parent context with summary
- [ ] Return from non-existent branch returns ErrNotFound
- [ ] Return from already-completed branch returns ErrAlreadyCompleted
- [ ] Return with extract_memory=true queues for distillation

### 6.2 Implementation (GREEN)

```
internal/folding/return.go - ReturnHandler
```

**Key Implementation**:
```go
func (h *ReturnHandler) Return(ctx context.Context, branchID string, message string, extractMemory bool) error {
    branch, err := h.repo.Get(ctx, branchID)
    if err != nil {
        return err
    }

    if branch.Status != BranchStatusActive {
        return ErrAlreadyCompleted
    }

    // Check for active children
    children, _ := h.repo.ListChildren(ctx, branchID)
    for _, child := range children {
        if child.Status == BranchStatusActive {
            h.ForceReturn(ctx, child.ID, "parent returning")
        }
    }

    // CRITICAL: Scrub secrets from return message
    scrubbed, err := h.scrubber.Scrub(message)
    if err != nil {
        return fmt.Errorf("secret scrubbing failed: %w", err)
    }

    // Update branch
    branch.Result = &scrubbed
    branch.Status = BranchStatusCompleted
    branch.CompletedAt = ptr(time.Now())

    if err := h.repo.Update(ctx, branch); err != nil {
        return err
    }

    // Queue for memory extraction if requested
    if extractMemory {
        h.distiller.Queue(branch)
    }

    return nil
}
```

---

## Phase 7: MCP Tool Registration

**Goal**: Register branch() and return() as MCP tools.

### 7.1 Tests First (RED)

```
internal/mcp/folding_tools_test.go
```

**Test Cases**:
- [ ] branch tool schema validates required fields
- [ ] branch tool returns valid branch_id
- [ ] branch tool enforces max_budget limit from config
- [ ] return tool requires message field
- [ ] return tool validates branch_id exists
- [ ] Tools update session's active_branch_id
- [ ] Other MCP tools respect active_branch_id for budget tracking

### 7.2 Implementation (GREEN)

```
internal/mcp/tools.go - Add branch and return tools
internal/mcp/session.go - Add active_branch_id tracking
```

---

## Phase 8: Timeout Enforcement

**Goal**: Implement timeout with goroutine and context cancellation.

### 8.1 Tests First (RED)

```
internal/folding/timeout_test.go
```

**Test Cases**:
- [ ] Branch with timeout spawns timeout goroutine
- [ ] Timeout expiry triggers ForceReturn
- [ ] Normal return cancels timeout goroutine
- [ ] ForceReturn cancels timeout goroutine
- [ ] Timeout goroutine doesn't leak on branch completion

### 8.2 Implementation (GREEN)

```
internal/folding/timeout.go - TimeoutManager
```

---

## Phase 9: Integration Tests

**Goal**: End-to-end tests via MCP protocol.

### 9.1 Tests (RED then GREEN)

```
test/integration/folding_test.go
```

**Test Scenarios**:
- [ ] Full branch lifecycle: create → work → return
- [ ] Nested branches up to depth 3
- [ ] Budget exhaustion triggers force return
- [ ] Timeout triggers force return
- [ ] Secret in return message is scrubbed
- [ ] Memory injection from ReasoningBank
- [ ] Session end cleans up all branches
- [ ] Concurrent branches in same session

---

## Phase 10: Documentation & Config

### 10.1 Update Spec Documents

- [ ] DESIGN.md: Add input validation section
- [ ] DESIGN.md: Add concurrency controls section
- [ ] ARCH.md: Add isolation model decision
- [ ] ARCH.md: Add session context tracking
- [ ] SPEC.md: Add security requirements

### 10.2 Configuration

```yaml
context_folding:
  enabled: true
  default_budget: 8192
  max_budget: 32768
  max_depth: 3
  default_timeout_seconds: 300
  max_timeout_seconds: 600
  injection_budget_ratio: 0.2
  memory_min_confidence: 0.7
  memory_max_items: 10
  max_concurrent_branches_per_session: 10
  max_concurrent_branches_per_instance: 100
  branch_creation_rate_limit: 5/minute
```

---

## Test Coverage Targets

| Package | Target | Focus |
|---------|--------|-------|
| internal/folding/types | 95% | Validation logic |
| internal/folding/repository | 90% | Concurrency |
| internal/folding/budget | 95% | Race conditions |
| internal/folding/manager | 85% | Lifecycle |
| internal/folding/injector | 80% | Error handling |
| internal/folding/return | 95% | Secret scrubbing |
| internal/mcp/folding_tools | 85% | MCP integration |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Token counting accuracy | Use tiktoken-go with same model as LLM |
| Secret scrubbing false negatives | Use gitleaks with custom rules |
| Race conditions | Extensive concurrent tests, mutex verification |
| Memory leaks from timeout goroutines | Test goroutine counts before/after |
| Integration with existing services | Mock-first unit tests, integration tests last |

---

## Estimated Effort

| Phase | Complexity | Estimate |
|-------|------------|----------|
| Phase 1: Types | Low | Tests: 2h, Impl: 2h |
| Phase 2: Repository | Medium | Tests: 3h, Impl: 3h |
| Phase 3: BudgetTracker | Medium | Tests: 3h, Impl: 3h |
| Phase 4: BranchManager | High | Tests: 4h, Impl: 4h |
| Phase 5: MemoryInjector | Medium | Tests: 2h, Impl: 2h |
| Phase 6: ReturnHandler | High | Tests: 3h, Impl: 2h |
| Phase 7: MCP Tools | Medium | Tests: 2h, Impl: 2h |
| Phase 8: Timeout | Medium | Tests: 2h, Impl: 2h |
| Phase 9: Integration | High | Tests: 4h |
| Phase 10: Docs | Low | 2h |

**Total Estimated**: ~45 hours of focused work

---

## Success Criteria

- [ ] All tests pass (`go test ./...`)
- [ ] Coverage targets met (`go test -cover`)
- [ ] No race conditions (`go test -race`)
- [ ] Secret scrubbing validated with known secrets
- [ ] <100ms branch creation latency (without async injection)
- [ ] Concurrent branch tests pass consistently
- [ ] Integration tests demonstrate full lifecycle
