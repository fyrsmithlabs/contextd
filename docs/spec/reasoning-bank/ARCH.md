# ReasoningBank Architecture

**Feature**: ReasoningBank (Layer 2)
**Status**: Draft
**Created**: 2025-11-22

## System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                    contextd MCP Server                          │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ Memory Manager  │  │   Distiller     │  │ Confidence      │
│                 │  │   (async)       │  │ Evaluator       │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Qdrant                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ {org}_      │  │ {org}_{team}│  │ {org}_{team}│             │
│  │ memories    │  │ _memories   │  │ _{proj}_mem │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Memory Manager

**Responsibility**: CRUD operations on memories, search, and feedback handling.

```go
type MemoryManager interface {
    Search(ctx context.Context, req SearchRequest) ([]Memory, error)
    Record(ctx context.Context, memory Memory) (string, error)
    Feedback(ctx context.Context, id string, helpful bool, comment string) error
    Get(ctx context.Context, id string) (*Memory, error)
    Update(ctx context.Context, id string, updates MemoryUpdates) error
}

type Memory struct {
    ID           string            `json:"id"`
    Title        string            `json:"title"`
    Description  string            `json:"description"`
    Content      string            `json:"content"`
    Outcome      Outcome           `json:"outcome"` // success, failure, mixed
    Confidence   float64           `json:"confidence"`
    UsageCount   int               `json:"usage_count"`
    Tags         []string          `json:"tags"`
    Scope        Scope             `json:"scope"` // project, team, org
    Project      string            `json:"project,omitempty"`
    Team         string            `json:"team,omitempty"`
    Org          string            `json:"org"`
    SourceSession string           `json:"source_session,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    LastUsed     *time.Time        `json:"last_used,omitempty"`
    Embedding    []float32         `json:"-"` // not serialized to JSON
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
│  Agent   │────►│ MCP Handler  │────►│MemoryManager │────►│   Qdrant     │
└──────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                              │
                                              ▼
                                      ┌──────────────┐
                                      │  Embedder    │
                                      └──────────────┘
```

**Sequence**:
1. Agent calls `memory_search(query, scope, limit)`
2. MemoryManager embeds the query
3. Search each scope level (project → team → org)
4. Merge and rank results by relevance × confidence
5. Return top-k memories

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
Collection: {org}_{team}_{project}_memories

Points:
├── id: UUID
├── vector: [1536] (embedding)
└── payload:
    ├── title: string
    ├── description: string
    ├── content: string
    ├── outcome: "success" | "failure" | "mixed"
    ├── confidence: float (0.0 - 1.0)
    ├── usage_count: int
    ├── tags: string[]
    ├── source_session: string?
    ├── created_at: timestamp
    └── last_used: timestamp?
```

## Search Strategy

### Scope Cascade

```go
func (m *MemoryManager) Search(ctx context.Context, req SearchRequest) ([]Memory, error) {
    var allResults []Memory

    scopes := []string{
        fmt.Sprintf("%s_%s_%s_memories", req.Org, req.Team, req.Project),
        fmt.Sprintf("%s_%s_memories", req.Org, req.Team),
        fmt.Sprintf("%s_memories", req.Org),
    }

    for _, collection := range scopes {
        results, err := m.qdrant.Search(ctx, collection, SearchParams{
            Vector:         req.Embedding,
            Filter:         buildFilter(req),
            Limit:          req.Limit,
            ScoreThreshold: req.MinConfidence,
        })
        if err != nil {
            continue // skip unavailable collections
        }
        allResults = append(allResults, results...)
    }

    // Deduplicate and rank
    return m.rankAndDedupe(allResults, req.Limit), nil
}
```

### Ranking Formula

```
score = semantic_similarity × confidence × recency_boost × scope_weight

where:
- semantic_similarity: cosine distance from Qdrant (0.0 - 1.0)
- confidence: memory confidence score (0.0 - 1.0)
- recency_boost: 1.0 + 0.1 × (1 - days_since_used / 365)
- scope_weight: project=1.0, team=0.9, org=0.8
```

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
