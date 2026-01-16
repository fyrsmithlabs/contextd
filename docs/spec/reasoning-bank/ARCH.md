# ReasoningBank Architecture

**Feature**: ReasoningBank (Layer 2)
**Status**: Implemented
**Created**: 2025-11-22
**Updated**: 2026-01-16

## System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                    contextd MCP Server                          │
│                    (internal/mcp/)                              │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ ReasoningBank   │  │   Distiller     │  │ Confidence      │
│ Service         │  │   (async)       │  │ Calculator      │
│ (internal/      │  │                 │  │                 │
│ reasoningbank/) │  │                 │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   VectorStore Abstraction                       │
│                   (internal/vectorstore/)                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ chromem (default, embedded) OR Qdrant (optional)        │   │
│  │ Database-per-project isolation via StoreProvider        │   │
│  │ Payload-based tenant filtering                          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### ReasoningBank Service

**Responsibility**: CRUD operations on memories, search, and feedback handling.

**Location**: `internal/reasoningbank/service.go`

```go
// Service interface (simplified view of actual implementation)
type Service interface {
    Search(ctx context.Context, projectID, query string, limit int) ([]Memory, error)
    Record(ctx context.Context, projectID string, memory Memory) (string, error)
    Feedback(ctx context.Context, memoryID string, helpful bool) error
    RecordOutcome(ctx context.Context, memoryID string, succeeded bool, sessionID string) error
    Get(ctx context.Context, memoryID string) (*Memory, error)
    GetByProjectID(ctx context.Context, projectID, memoryID string) (*Memory, error)
}

type Memory struct {
    ID           string            `json:"id"`
    ProjectID    string            `json:"project_id"`
    Title        string            `json:"title"`
    Description  string            `json:"description"`
    Content      string            `json:"content"`
    Outcome      Outcome           `json:"outcome"` // success, failure
    Confidence   float64           `json:"confidence"`
    UsageCount   int               `json:"usage_count"`
    Tags         []string          `json:"tags"`
    SourceSession string           `json:"source_session,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
    LastUsed     *time.Time        `json:"last_used,omitempty"`
}
```

### Distiller

**Responsibility**: Extract memories from completed sessions asynchronously.

```go
type Distiller interface {
    QueueSession(ctx context.Context, sessionID string, outcome Outcome) error
    Process(ctx context.Context) error // background worker
}

type ExtractionPrompt struct {
    SessionTrace string
    Outcome      Outcome
}
```

### Confidence Evaluator

**Responsibility**: Update memory confidence based on multiple signals.

```go
type ConfidenceEvaluator interface {
    OnFeedback(ctx context.Context, memoryID string, helpful bool) error
    OnUsage(ctx context.Context, memoryID string, taskSucceeded bool) error
    OnCodeStability(ctx context.Context, memoryID string, reverted bool) error
    DecayAll(ctx context.Context) error // periodic decay
}
```

## Data Flow

### Memory Search Flow

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent   │────►│ MCP Handler  │────►│ ReasoningBank│────►│ VectorStore  │
│          │     │(internal/mcp)│     │   Service    │     │  (chromem/   │
└──────────┘     └──────────────┘     └──────────────┘     │   Qdrant)    │
                                              │            └──────────────┘
                                              ▼
                                      ┌──────────────┐
                                      │  FastEmbed   │
                                      │  (local ONNX)│
                                      └──────────────┘
```

**Sequence**:
1. Agent calls `memory_search(project_id, query, limit)` via MCP
2. ReasoningBank service embeds the query using FastEmbed
3. Search project's vectorstore with confidence filtering (MinConfidence=0.7)
4. Results ranked by semantic similarity
5. Return top-k memories (scrubbed for secrets)

### Distillation Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│SessionManager│────►│  Distiller   │────►│    LLM       │────►│MemoryManager │
│  (end)       │     │   Queue      │     │  (extract)   │     │  (store)     │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
```

**Sequence**:
1. Session ends with outcome
2. Session trace queued for distillation
3. Background worker processes queue
4. LLM extracts strategies/anti-patterns
5. Memories stored with initial confidence

### Confidence Update Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Signal     │────►│ Confidence   │────►│   Qdrant     │
│  (feedback,  │     │ Evaluator    │     │  (update)    │
│   usage...)  │     └──────────────┘     └──────────────┘
└──────────────┘
```

**Signals**:
- Explicit feedback: +0.3 (helpful) / -0.2 (not helpful)
- Implicit success: +0.1 (task completed)
- Code stability: +0.2 (no reverts for 7 days)
- Time decay: -0.05 per month

## Collection Schema

```
Collection: memories (per-project database via StoreProvider)

Documents:
├── id: UUID
├── vector: [384] (FastEmbed all-MiniLM-L6-v2)
└── metadata:
    ├── tenant_id: string (required for isolation)
    ├── project_id: string
    ├── title: string
    ├── description: string
    ├── content: string
    ├── outcome: "success" | "failure"
    ├── confidence: float (0.0 - 1.0)
    ├── usage_count: int
    ├── tags: string[]
    ├── source_session: string?
    ├── created_at: timestamp
    ├── updated_at: timestamp
    └── last_used: timestamp?
```

**Note**: Team/org scoping (mentioned in original architecture) is NOT implemented.
All memories are project-scoped with database-per-project isolation.

## Search Strategy

### Current Implementation (Project-Scoped)

```go
// Simplified view of actual implementation in internal/reasoningbank/service.go
func (s *Service) Search(ctx context.Context, projectID, query string, limit int) ([]Memory, error) {
    // Get project-specific store via StoreProvider
    store, err := s.stores.GetProjectStore(ctx, tenant, "", projectID)
    if err != nil {
        return nil, err
    }

    // Search with semantic similarity
    results, err := store.Search(ctx, query, limit)
    if err != nil {
        return nil, err
    }

    // Post-filter by confidence threshold (MinConfidence = 0.7)
    var filtered []Memory
    for _, mem := range results {
        if mem.Confidence >= MinConfidence {
            filtered = append(filtered, mem)
        }
    }

    return filtered, nil
}
```

### Ranking Formula

```
score = semantic_similarity (cosine distance from vectorstore, 0.0 - 1.0)

Post-filtering: confidence >= 0.7 (MinConfidence)
```

**Note**: Scope cascade (project -> team -> org) and complex ranking formulas
described in original architecture are NOT implemented. Search is project-scoped only.

## Integration Points

### Layer 1 (Context-Folding)

- Provides memories for branch injection
- Receives branch outcomes for distillation

### Layer 3 (Institutional Knowledge)

- High-confidence memories promoted to team/org scope
- Cross-project patterns detected and consolidated

### Consolidation Pipeline

- Similar memories merged periodically
- Low-confidence memories pruned
