# Context-Folding Design

**Feature**: Context-Folding (Layer 1)
**Status**: Implemented
**Created**: 2025-11-22
**Updated**: 2026-01-16

**Implementation**: `internal/folding/`

**Note**: Tool names have been updated from `branch`/`return` to `branch_create`/`branch_return`
in the actual implementation. See SPEC.md for current tool definitions.

## MCP Tool Definitions

### branch

Creates an isolated context branch for subtask execution.

```json
{
  "name": "branch",
  "description": "Fork context into isolated subtask",
  "inputSchema": {
    "type": "object",
    "properties": {
      "description": {
        "type": "string",
        "description": "What this subtask is doing (used for memory retrieval)"
      },
      "prompt": {
        "type": "string",
        "description": "Instructions for the subtask"
      },
      "budget": {
        "type": "integer",
        "description": "Token limit for this branch",
        "default": 8192
      },
      "inject_memories": {
        "type": "boolean",
        "description": "Whether to inject relevant memories",
        "default": true
      },
      "timeout_seconds": {
        "type": "integer",
        "description": "Maximum execution time",
        "default": 300
      }
    },
    "required": ["description", "prompt"]
  }
}
```

**Response**:
```json
{
  "branch_id": "br_abc123",
  "injected_context": [
    {
      "type": "memory",
      "title": "File search pattern",
      "content": "When searching for function definitions..."
    }
  ],
  "budget_allocated": 8192,
  "parent_budget_remaining": 24000
}
```

### return

Completes a branch and returns results to parent context.

```json
{
  "name": "return",
  "description": "Complete subtask and return summary to parent",
  "inputSchema": {
    "type": "object",
    "properties": {
      "message": {
        "type": "string",
        "description": "Summary to pass back to parent context"
      },
      "extract_memory": {
        "type": "boolean",
        "description": "Whether to extract this branch as a memory candidate",
        "default": false
      }
    },
    "required": ["message"]
  }
}
```

**Response**:
```json
{
  "success": true,
  "tokens_used": 3420,
  "memory_queued": false
}
```

## Data Structures

### Branch Record

```go
package folding

import "time"

type BranchStatus string

const (
    BranchStatusCreated   BranchStatus = "created"
    BranchStatusActive    BranchStatus = "active"
    BranchStatusCompleted BranchStatus = "completed"
    BranchStatusTimeout   BranchStatus = "timeout"
    BranchStatusFailed    BranchStatus = "failed"
)

type Branch struct {
    ID             string        `json:"id"`
    SessionID      string        `json:"session_id"`
    ParentID       *string       `json:"parent_id,omitempty"`
    Depth          int           `json:"depth"`
    Description    string        `json:"description"`
    Prompt         string        `json:"prompt"`
    BudgetTotal    int           `json:"budget_total"`
    BudgetUsed     int           `json:"budget_used"`
    TimeoutSeconds int           `json:"timeout_seconds"`
    Status         BranchStatus  `json:"status"`
    Result         *string       `json:"result,omitempty"`
    Error          *string       `json:"error,omitempty"`
    InjectedMemories []string    `json:"injected_memories"` // memory IDs
    CreatedAt      time.Time     `json:"created_at"`
    CompletedAt    *time.Time    `json:"completed_at,omitempty"`
}

type BranchRequest struct {
    SessionID      string `json:"session_id"`
    Description    string `json:"description"`
    Prompt         string `json:"prompt"`
    Budget         int    `json:"budget"`
    InjectMemories bool   `json:"inject_memories"`
    TimeoutSeconds int    `json:"timeout_seconds"`
}

type BranchResponse struct {
    BranchID              string           `json:"branch_id"`
    InjectedContext       []InjectedItem   `json:"injected_context"`
    BudgetAllocated       int              `json:"budget_allocated"`
    ParentBudgetRemaining int              `json:"parent_budget_remaining"`
}

type InjectedItem struct {
    Type    string `json:"type"` // "memory", "policy", "standard"
    ID      string `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
    Tokens  int    `json:"tokens"`
}
```

## Algorithms

### Memory Injection Algorithm

```go
func (i *MemoryInjector) InjectForBranch(ctx context.Context, branch *Branch) ([]InjectedItem, int, error) {
    // Budget allocation: 20% of branch budget for injection
    injectionBudget := branch.BudgetTotal / 5

    // Query ReasoningBank for relevant memories
    embedding := i.embedder.Embed(branch.Description + " " + branch.Prompt)

    memories, err := i.reasoningBank.Search(ctx, MemorySearchRequest{
        Embedding:     embedding,
        Scope:         "all", // project → team → org
        MinConfidence: 0.7,
        Limit:         10,
    })
    if err != nil {
        return nil, 0, fmt.Errorf("memory search failed: %w", err)
    }

    // Fit to budget
    var injected []InjectedItem
    var totalTokens int

    for _, mem := range memories {
        tokens := i.tokenizer.Count(mem.Content)
        if totalTokens + tokens > injectionBudget {
            break
        }

        injected = append(injected, InjectedItem{
            Type:    "memory",
            ID:      mem.ID,
            Title:   mem.Title,
            Content: mem.Content,
            Tokens:  tokens,
        })
        totalTokens += tokens
    }

    return injected, totalTokens, nil
}
```

### Budget Enforcement Algorithm

```go
func (t *BudgetTracker) CheckAndEnforce(ctx context.Context, branchID string, newTokens int) error {
    branch, err := t.branchManager.Get(ctx, branchID)
    if err != nil {
        return err
    }

    projectedUsage := branch.BudgetUsed + newTokens

    if projectedUsage >= branch.BudgetTotal {
        // Force return with warning
        return t.branchManager.ForceReturn(ctx, branchID,
            fmt.Sprintf("budget exhausted: %d/%d tokens", projectedUsage, branch.BudgetTotal))
    }

    // Warn at 80% usage
    if float64(projectedUsage) / float64(branch.BudgetTotal) > 0.8 {
        t.notifier.Warn(ctx, branchID, "budget_warning", map[string]any{
            "used":  projectedUsage,
            "total": branch.BudgetTotal,
        })
    }

    return t.branchManager.ConsumeTokens(ctx, branchID, newTokens)
}
```

## Sequence Diagrams

### Normal Branch Lifecycle

```
Agent              MCP Server           BranchManager        MemoryInjector
  │                    │                     │                    │
  │─── branch() ──────►│                     │                    │
  │                    │─── Create() ───────►│                    │
  │                    │                     │─── Inject() ──────►│
  │                    │                     │◄── memories ───────│
  │                    │◄── Branch ──────────│                    │
  │◄── response ───────│                     │                    │
  │                    │                     │                    │
  │    [subtask work]  │                     │                    │
  │                    │                     │                    │
  │─── return() ──────►│                     │                    │
  │                    │─── Complete() ─────►│                    │
  │                    │◄── success ─────────│                    │
  │◄── response ───────│                     │                    │
```

### Budget Exhaustion

```
Agent              MCP Server           BudgetTracker        BranchManager
  │                    │                     │                    │
  │    [work...]       │                     │                    │
  │                    │─── Check() ────────►│                    │
  │                    │                     │── exhausted? ─────►│
  │                    │                     │◄── yes ────────────│
  │                    │                     │─── ForceReturn() ─►│
  │                    │◄── forced ──────────│                    │
  │◄── budget_warning ─│                     │                    │
  │◄── auto_return ────│                     │                    │
```

## Error Handling

| Error | Handling |
|-------|----------|
| Branch not found | Return 404, log warning |
| Budget exceeded | Force return with partial results |
| Timeout exceeded | Force return with timeout message |
| Parent branch closed | Orphan cleanup, log error |
| Memory injection fails | Continue without injection, log warning |

## Configuration

```yaml
context_folding:
  default_budget: 8192
  max_budget: 32768
  max_depth: 3
  default_timeout_seconds: 300
  max_timeout_seconds: 600
  injection_budget_ratio: 0.2  # 20% of branch budget
  memory_min_confidence: 0.7
  memory_max_items: 10
```

## Testing Strategy

### Unit Tests

- Branch creation with valid/invalid parameters
- Budget allocation and enforcement
- Memory injection within budget
- State transitions
- Nested branch handling

### Integration Tests

- Full branch lifecycle with MCP protocol
- Memory injection from real ReasoningBank
- Concurrent branch handling
- Timeout and budget enforcement

### Performance Tests

- Branch creation latency <50ms
- Memory injection latency <100ms
- 100 concurrent branches per instance
