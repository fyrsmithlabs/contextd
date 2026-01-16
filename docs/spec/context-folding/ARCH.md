# Context-Folding Architecture

**Feature**: Context-Folding (Layer 1)
**Status**: Implemented
**Created**: 2025-11-22
**Updated**: 2026-01-16

**Implementation**: `internal/folding/`

## Architectural Decisions (from Consensus Review 2025-12-13)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Isolation Model** | Goroutine + strict data separation | Cheap, sufficient for context isolation, no IPC overhead |
| **Token Counting** | Server-side proxy via MCP interceptor | Cannot trust agent self-reporting (security) |
| **Event Pattern** | BudgetTracker emits events, BranchManager subscribes | Breaks circular dependency |
| **Session Tracking** | `active_branch_id` per session in contextd | MCP is stateless, contextd manages state |
| **Secret Scrubbing** | gitleaks on return() content | Leverage existing infrastructure |
| **Budget Accounting** | Independent per branch, deduct from parent at creation | No per-token propagation to ancestors |

## System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                         AI Agent                                │
│                    (Claude, GPT, etc.)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ MCP Protocol
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    contextd MCP Server                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Context-Folding Engine                      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │   │
│  │  │   Branch    │  │   Budget    │  │   Memory    │      │   │
│  │  │   Manager   │  │   Tracker   │  │   Injector  │      │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │   │
│  │         │               │                                │   │
│  │         │    events     │                                │   │
│  │         │◄──────────────┤                                │   │
│  │         │  (budget      │                                │   │
│  │         │   exhausted)  │                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Session Context Tracker                     │   │
│  │         (active_branch_id per MCP session)               │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ReasoningBank (Layer 2)                      │
│            (Memory retrieval for branch injection)              │
└─────────────────────────────────────────────────────────────────┘
```

## Isolation Model

**Decision**: Goroutine-based isolation with strict data separation.

Branches are isolated via:
1. **Separate goroutine** for timeout enforcement
2. **No shared memory access** - branches only see description, prompt, and injected memories
3. **Session context tracking** - contextd maintains `active_branch_id` to route tool calls
4. **Budget isolation** - each branch has independent budget counter

```
Session State:
┌────────────────────────────────────────┐
│ session_id: "sess_123"                 │
│ active_branch_id: "br_456"             │
│ branches: [br_001, br_456, br_789]     │
│ root_budget_remaining: 50000           │
└────────────────────────────────────────┘

Branch br_456 sees ONLY:
┌────────────────────────────────────────┐
│ description: "Find database config"    │
│ prompt: "Search for DB connection..."  │
│ injected_memories: [mem_001, mem_002]  │
│ budget_remaining: 8000                 │
└────────────────────────────────────────┘
```

## Component Architecture

### Branch Manager

**Responsibility**: Create, track, and terminate context branches.

```go
type BranchManager interface {
    Create(ctx context.Context, req BranchRequest) (*Branch, error)
    Return(ctx context.Context, branchID string, message string) error
    ForceReturn(ctx context.Context, branchID string, reason string) error
    Get(ctx context.Context, branchID string) (*Branch, error)
    List(ctx context.Context, sessionID string) ([]*Branch, error)
}

type Branch struct {
    ID          string
    SessionID   string
    ParentID    *string  // nil for root branches
    Description string
    Prompt      string
    Budget      int
    UsedTokens  int
    Status      BranchStatus  // active, completed, failed, timeout
    CreatedAt   time.Time
    CompletedAt *time.Time
    Result      *string
}
```

### Budget Tracker

**Responsibility**: Track token usage within branches, enforce limits.

```go
type BudgetTracker interface {
    Allocate(branchID string, budget int) error
    Consume(branchID string, tokens int) error
    Remaining(branchID string) (int, error)
    IsExhausted(branchID string) bool
}
```

### Memory Injector

**Responsibility**: Retrieve and inject relevant memories into branch context.

```go
type MemoryInjector interface {
    InjectForBranch(ctx context.Context, branch *Branch) ([]Memory, int, error)
    // Returns: injected memories, tokens consumed, error
}
```

## Data Flow

### Branch Creation Flow

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent   │────►│ MCP Handler  │────►│ BranchManager│────►│ BudgetTracker│
└──────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                              │
                                              ▼
                                      ┌──────────────┐
                                      │MemoryInjector│
                                      └──────────────┘
                                              │
                                              ▼
                                      ┌──────────────┐
                                      │ ReasoningBank│
                                      │   (Layer 2)  │
                                      └──────────────┘
```

**Sequence**:
1. Agent calls `branch(description, prompt)`
2. MCP Handler validates request
3. BranchManager creates Branch record
4. BudgetTracker allocates token budget
5. MemoryInjector queries ReasoningBank for relevant memories
6. Response includes branch_id and injected_context

### Branch Return Flow

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent   │────►│ MCP Handler  │────►│ BranchManager│────►│  Distiller   │
└──────────┘     └──────────────┘     └──────────────┘     │  (optional)  │
                                              │            └──────────────┘
                                              ▼
                                      ┌──────────────┐
                                      │ Parent Branch│
                                      │   or Main    │
                                      └──────────────┘
```

**Sequence**:
1. Agent calls `return(message)`
2. MCP Handler validates branch exists and is active
3. BranchManager marks branch completed
4. If `extract_memory=true`, queue for distillation
5. Return message delivered to parent context

## State Machine

```
                    ┌─────────┐
                    │ Created │
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
         ┌─────────│  Active │─────────┐
         │         └────┬────┘         │
         │              │              │
         ▼              ▼              ▼
    ┌─────────┐   ┌─────────┐   ┌─────────┐
    │ Timeout │   │Completed│   │ Failed  │
    └─────────┘   └─────────┘   └─────────┘
```

**Transitions**:
- `Created → Active`: Branch initialized, ready for work
- `Active → Completed`: Normal `return()` call
- `Active → Timeout`: Budget or time limit exceeded
- `Active → Failed`: Unrecoverable error

## Integration Points

### Layer 2 (ReasoningBank)

- **Read**: Query memories for branch injection
- **Write**: Queue successful branches for memory extraction

### Session Manager

- Branches belong to sessions
- Session end triggers cleanup of orphaned branches

### Distillation Pipeline

- Completed branches optionally queued for pattern extraction
- Failed branches queued for anti-pattern extraction

## Scalability Considerations

### Concurrent Branches

- Multiple agents can have concurrent sessions
- Each session can have multiple active branches
- Branch state stored in-memory with persistence for crash recovery

### Token Budget Pools

- Parent session has total budget
- Branches draw from parent pool or have independent allocation
- Configuration determines allocation strategy

## Security Considerations

### Branch Isolation

- Branches MUST NOT access sibling branch data
- Parent context protected from branch modifications
- JWT claims propagate to branches for RBAC
