# ReasoningBank Design

**Feature**: ReasoningBank (Layer 2)
**Status**: Draft
**Created**: 2025-11-22

## MCP Tool Definitions

### memory_search

```json
{
  "name": "memory_search",
  "description": "Search for relevant strategies and patterns",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Natural language query"
      },
      "scope": {
        "type": "string",
        "enum": ["project", "team", "org", "all"],
        "default": "all",
        "description": "Scope to search within"
      },
      "outcome": {
        "type": "string",
        "enum": ["success", "failure", "all"],
        "default": "all",
        "description": "Filter by outcome type"
      },
      "limit": {
        "type": "integer",
        "default": 5,
        "description": "Maximum results"
      },
      "min_confidence": {
        "type": "number",
        "default": 0.5,
        "description": "Minimum confidence threshold"
      }
    },
    "required": ["query"]
  }
}
```

**Response**:
```json
{
  "memories": [
    {
      "id": "mem_abc123",
      "title": "Go Error Wrapping Pattern",
      "description": "When handling errors in Go services",
      "content": "Always wrap errors with context using fmt.Errorf...",
      "outcome": "success",
      "confidence": 0.92,
      "usage_count": 15,
      "relevance": 0.87,
      "scope": "project"
    }
  ],
  "total_found": 12,
  "tokens_used": 320
}
```

### memory_record

```json
{
  "name": "memory_record",
  "description": "Explicitly capture a strategy or pattern",
  "inputSchema": {
    "type": "object",
    "properties": {
      "title": {
        "type": "string",
        "description": "Short, descriptive title"
      },
      "description": {
        "type": "string",
        "description": "When/why to apply this strategy"
      },
      "content": {
        "type": "string",
        "description": "Detailed steps or approach"
      },
      "outcome": {
        "type": "string",
        "enum": ["success", "failure"],
        "description": "Whether this is a pattern to follow or avoid"
      },
      "tags": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Categorization tags"
      }
    },
    "required": ["title", "description", "content", "outcome"]
  }
}
```

**Response**:
```json
{
  "id": "mem_xyz789",
  "message": "Memory recorded successfully",
  "initial_confidence": 0.8
}
```

### memory_feedback

```json
{
  "name": "memory_feedback",
  "description": "Provide feedback on a retrieved memory",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_id": {
        "type": "string",
        "description": "ID of the memory"
      },
      "helpful": {
        "type": "boolean",
        "description": "Whether the memory was helpful"
      },
      "comment": {
        "type": "string",
        "description": "Optional feedback comment"
      }
    },
    "required": ["memory_id", "helpful"]
  }
}
```

**Response**:
```json
{
  "success": true,
  "new_confidence": 0.95,
  "message": "Feedback recorded"
}
```

## Distillation Pipeline

### Extraction Prompt Template

```
You are analyzing a completed agent session to extract reusable strategies.

Session Outcome: {{outcome}}

Session Trace:
{{trace}}

For SUCCESSFUL outcomes, extract strategies that worked:
- What approach led to success?
- What patterns can be reused?
- What context made this work?

For FAILED outcomes, extract anti-patterns to avoid:
- What approach failed?
- Why did it fail?
- What should be done instead?

Format each extraction as:

## Memory {{n}}
**Title**: <concise name, max 50 chars>
**Description**: <when to use/avoid this, max 200 chars>
**Content**: <detailed steps or explanation>
**Tags**: <comma-separated tags>
**Outcome**: success | failure

Extract 0-3 memories. Only extract if there's genuine reusable value.
If nothing is extractable, respond with: NO_EXTRACTIONS
```

### Distillation Worker

```go
package distiller

import (
    "context"
    "encoding/json"
)

type Distiller struct {
    queue         Queue
    llm           LLMClient
    memoryManager MemoryManager
    embedder      Embedder
}

func (d *Distiller) Process(ctx context.Context) error {
    for {
        item, err := d.queue.Dequeue(ctx)
        if err != nil {
            return err
        }

        session, err := d.sessionStore.Get(ctx, item.SessionID)
        if err != nil {
            d.queue.Fail(ctx, item.ID, err)
            continue
        }

        prompt := d.buildPrompt(session.Trace, item.Outcome)
        response, err := d.llm.Complete(ctx, prompt)
        if err != nil {
            d.queue.Retry(ctx, item.ID)
            continue
        }

        if response == "NO_EXTRACTIONS" {
            d.queue.Complete(ctx, item.ID)
            continue
        }

        memories, err := d.parseExtractions(response)
        if err != nil {
            d.queue.Fail(ctx, item.ID, err)
            continue
        }

        for _, mem := range memories {
            mem.SourceSession = item.SessionID
            mem.Confidence = d.initialConfidence(item.Outcome)
            mem.Embedding = d.embedder.Embed(mem.Title + " " + mem.Description)

            _, err := d.memoryManager.Record(ctx, mem)
            if err != nil {
                // Log but continue
                continue
            }
        }

        d.queue.Complete(ctx, item.ID)
    }
}

func (d *Distiller) initialConfidence(outcome Outcome) float64 {
    switch outcome {
    case OutcomeSuccess:
        return 0.7
    case OutcomeFailure:
        return 0.6 // anti-patterns start lower
    default:
        return 0.5
    }
}
```

## Confidence Evaluation

### Signal Weights

```go
const (
    WeightExplicitPositive = 0.30
    WeightExplicitNegative = -0.20
    WeightImplicitSuccess  = 0.10
    WeightImplicitFailure  = -0.05
    WeightCodeStable       = 0.20
    WeightCodeReverted     = -0.15
    WeightMonthlyDecay     = -0.05
)

func (e *ConfidenceEvaluator) Calculate(mem *Memory, signals []Signal) float64 {
    delta := 0.0

    for _, sig := range signals {
        switch sig.Type {
        case SignalExplicitFeedback:
            if sig.Positive {
                delta += WeightExplicitPositive
            } else {
                delta += WeightExplicitNegative
            }
        case SignalTaskCompletion:
            if sig.Positive {
                delta += WeightImplicitSuccess
            } else {
                delta += WeightImplicitFailure
            }
        case SignalCodeStability:
            if sig.Positive {
                delta += WeightCodeStable
            } else {
                delta += WeightCodeReverted
            }
        }
    }

    // Apply time decay
    monthsOld := time.Since(mem.CreatedAt).Hours() / (24 * 30)
    delta += monthsOld * WeightMonthlyDecay

    newConfidence := mem.Confidence + delta

    // Clamp to [0.0, 1.0]
    if newConfidence < 0.0 {
        return 0.0
    }
    if newConfidence > 1.0 {
        return 1.0
    }
    return newConfidence
}
```

### Consensus Logic

```go
func (e *ConfidenceEvaluator) RequiresConsensus(mem *Memory) bool {
    // New memories need multiple signals before major confidence changes
    if mem.UsageCount < 3 {
        return true
    }
    return false
}

func (e *ConfidenceEvaluator) ApplyWithConsensus(ctx context.Context, mem *Memory, signal Signal) error {
    // Get recent signals for this memory
    signals, err := e.getRecentSignals(ctx, mem.ID, 7*24*time.Hour)
    if err != nil {
        return err
    }

    signals = append(signals, signal)

    // Require at least 2 agreeing signals for confidence change
    positiveCount := 0
    negativeCount := 0
    for _, s := range signals {
        if s.Positive {
            positiveCount++
        } else {
            negativeCount++
        }
    }

    if positiveCount >= 2 || negativeCount >= 2 {
        return e.ApplyDelta(ctx, mem, signals)
    }

    // Store signal but don't update confidence yet
    return e.storeSignal(ctx, mem.ID, signal)
}
```

## Configuration

```yaml
reasoning_bank:
  embedding:
    model: "text-embedding-3-small"  # or local model
    dimensions: 1536

  search:
    default_limit: 5
    max_limit: 20
    min_confidence: 0.5
    scope_weights:
      project: 1.0
      team: 0.9
      org: 0.8

  distillation:
    queue_batch_size: 10
    worker_count: 2
    extraction_model: "claude-3-haiku"
    max_extractions_per_session: 3

  confidence:
    initial_success: 0.7
    initial_failure: 0.6
    initial_explicit: 0.8
    min_for_injection: 0.5
    consensus_threshold: 2
    prune_threshold: 0.3
    decay_interval: "24h"
```

## Sequence Diagrams

### Memory Search and Usage

```
Agent              MCP Server           MemoryManager           Qdrant
  │                    │                     │                    │
  │─ memory_search ───►│                     │                    │
  │                    │─── Search() ───────►│                    │
  │                    │                     │─── embed(query) ──►│
  │                    │                     │◄── vector ─────────│
  │                    │                     │─── search(vector) ─►│
  │                    │                     │◄── results ────────│
  │                    │◄── memories ────────│                    │
  │◄── results ────────│                     │                    │
  │                    │                     │                    │
  │  [agent uses memory]                     │                    │
  │                    │                     │                    │
  │─ memory_feedback ─►│                     │                    │
  │                    │─── Feedback() ─────►│                    │
  │                    │                     │─── update conf ───►│
  │                    │◄── success ─────────│                    │
  │◄── response ───────│                     │                    │
```

### Session Distillation

```
SessionManager         Distiller            LLM              MemoryManager
      │                    │                  │                    │
      │─ QueueSession() ──►│                  │                    │
      │                    │                  │                    │
      │           [background worker]         │                    │
      │                    │                  │                    │
      │                    │─── extract() ───►│                    │
      │                    │◄── memories ─────│                    │
      │                    │                  │                    │
      │                    │─── Record() ────────────────────────►│
      │                    │◄── id ───────────────────────────────│
      │                    │                  │                    │
```

## Testing Strategy

### Unit Tests

- Memory CRUD operations
- Search ranking algorithm
- Confidence calculation
- Distillation parsing
- Scope cascade logic

### Integration Tests

- End-to-end memory search with Qdrant
- Distillation pipeline with real LLM
- Confidence updates across sessions
- Cross-scope search behavior

### Performance Tests

- Search latency <100ms for 10K memories
- Distillation throughput: 100 sessions/minute
- Concurrent search handling
