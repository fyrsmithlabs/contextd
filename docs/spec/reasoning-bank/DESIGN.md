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

**Side Effects** (self-improving confidence):
- Records explicit signal (positive=helpful)
- Updates project weights based on whether usage/outcome signals correctly predicted this feedback
- Triggers confidence recalculation for the memory

### memory_outcome

```json
{
  "name": "memory_outcome",
  "description": "Report whether a task succeeded after using a memory. Call this after completing a task that used a retrieved memory to help the system learn which memories are actually useful.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "memory_id": {
        "type": "string",
        "description": "ID of the memory that was used"
      },
      "succeeded": {
        "type": "boolean",
        "description": "Whether the task succeeded after using this memory"
      },
      "session_id": {
        "type": "string",
        "description": "Optional session ID for correlation"
      }
    },
    "required": ["memory_id", "succeeded"]
  }
}
```

**Response**:
```json
{
  "recorded": true,
  "new_confidence": 0.72,
  "message": "Outcome recorded"
}
```

**Agent Usage Pattern**:
```
1. Agent calls memory_search("how to handle timeouts")
   → Returns memory M1: "Use context.WithTimeout"

2. Agent implements solution using M1
   → Task completes successfully

3. Agent calls memory_outcome(memory_id=M1, succeeded=true)
   → System records outcome signal
   → M1's confidence increases
   → Outcome signal's weight potentially increases
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

## Self-Improving Confidence System

### Overview

The confidence system uses a **Bayesian adaptive approach** that learns which signals predict memory usefulness. Instead of fixed weights, the system maintains Beta distributions that evolve based on observed signal accuracy.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Signal Sources                        │
├─────────────────┬─────────────────┬─────────────────────┤
│ Explicit        │ Usage           │ Outcome             │
│ memory_feedback │ memory_search   │ memory_outcome(new) │
└────────┬────────┴────────┬────────┴──────────┬──────────┘
         │                 │                   │
         ▼                 ▼                   ▼
┌─────────────────────────────────────────────────────────┐
│                   Signal Store                           │
│  ┌─────────────────┐    ┌─────────────────────────┐     │
│  │ Event Log       │    │ Aggregates              │     │
│  │ (last 30 days)  │───▶│ (older + running totals)│     │
│  └─────────────────┘    └─────────────────────────┘     │
└────────────────────────────┬────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│              Bayesian Weight Learner                     │
│  Tracks alpha/beta per signal type per project          │
│  Computes adaptive weights from Beta distributions      │
└────────────────────────────┬────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│              Confidence Calculator                       │
│  Combines signals using learned weights                  │
│  Updates Memory.Confidence on each signal               │
└─────────────────────────────────────────────────────────┘
```

### Signal Types

| Signal Type | Source | Trigger |
|-------------|--------|---------|
| `explicit` | `memory_feedback` tool | User rates helpful/unhelpful |
| `usage` | `memory_search` tool | Memory retrieved in search results |
| `outcome` | `memory_outcome` tool | Agent reports task success/failure |

### Data Structures

```go
// SignalType identifies the source of a confidence signal
type SignalType string

const (
    SignalExplicit SignalType = "explicit"  // memory_feedback
    SignalUsage    SignalType = "usage"     // memory_search retrieval
    SignalOutcome  SignalType = "outcome"   // memory_outcome
)

// Signal represents a single confidence event
type Signal struct {
    ID         string     `json:"id"`
    MemoryID   string     `json:"memory_id"`
    ProjectID  string     `json:"project_id"`
    Type       SignalType `json:"type"`
    Positive   bool       `json:"positive"`
    SessionID  string     `json:"session_id"`
    Timestamp  time.Time  `json:"timestamp"`
}

// SignalAggregate stores rolled-up counts for data older than 30 days
type SignalAggregate struct {
    MemoryID     string    `json:"memory_id"`
    ProjectID    string    `json:"project_id"`
    ExplicitPos  int       `json:"explicit_pos"`
    ExplicitNeg  int       `json:"explicit_neg"`
    UsagePos     int       `json:"usage_pos"`
    UsageNeg     int       `json:"usage_neg"`
    OutcomePos   int       `json:"outcome_pos"`
    OutcomeNeg   int       `json:"outcome_neg"`
    LastRollup   time.Time `json:"last_rollup"`
}

// ProjectWeights tracks learned signal weights per project using Beta distributions
type ProjectWeights struct {
    ProjectID     string  `json:"project_id"`

    // Beta distribution params (alpha=successes, beta=failures)
    // Initial priors: Explicit 7:3 (70%), Usage/Outcome 5:5 (50%)
    ExplicitAlpha float64 `json:"explicit_alpha"`  // starts at 7
    ExplicitBeta  float64 `json:"explicit_beta"`   // starts at 3
    UsageAlpha    float64 `json:"usage_alpha"`     // starts at 5
    UsageBeta     float64 `json:"usage_beta"`      // starts at 5
    OutcomeAlpha  float64 `json:"outcome_alpha"`   // starts at 5
    OutcomeBeta   float64 `json:"outcome_beta"`    // starts at 5
}
```

### Bayesian Weight Learning

Each project learns which signals predict memory usefulness via Beta distributions.

**Initial State (new project):**
```
ExplicitAlpha: 7,  ExplicitBeta: 3   → 70% weight (trust user feedback)
UsageAlpha: 5,     UsageBeta: 5      → 50% weight (uncertain)
OutcomeAlpha: 5,   OutcomeBeta: 5    → 50% weight (uncertain)
```

**Learning Process:**

When explicit feedback arrives, check if other signals correctly predicted it:

```go
func (pw *ProjectWeights) LearnFromFeedback(memory *Memory, helpful bool, recentSignals []Signal) {
    // Check if usage signals predicted this feedback
    usagePredictedPositive := hasPositiveSignal(recentSignals, SignalUsage)
    if usagePredictedPositive == helpful {
        pw.UsageAlpha++ // Usage correctly predicted
    } else {
        pw.UsageBeta++  // Usage incorrectly predicted
    }

    // Check if outcome signals predicted this feedback
    outcomePredictedPositive := hasPositiveSignal(recentSignals, SignalOutcome)
    if outcomePredictedPositive == helpful {
        pw.OutcomeAlpha++ // Outcome correctly predicted
    } else {
        pw.OutcomeBeta++  // Outcome incorrectly predicted
    }
}
```

**Computing Normalized Weights:**

```go
func (pw *ProjectWeights) ComputeWeights() (explicit, usage, outcome float64) {
    // Beta distribution mean = alpha / (alpha + beta)
    rawExplicit := pw.ExplicitAlpha / (pw.ExplicitAlpha + pw.ExplicitBeta)
    rawUsage := pw.UsageAlpha / (pw.UsageAlpha + pw.UsageBeta)
    rawOutcome := pw.OutcomeAlpha / (pw.OutcomeAlpha + pw.OutcomeBeta)

    // Normalize to sum to 1.0
    total := rawExplicit + rawUsage + rawOutcome
    return rawExplicit/total, rawUsage/total, rawOutcome/total
}
```

### Confidence Calculation

Each memory maintains confidence as a Beta distribution updated by weighted signals.

```go
type MemoryConfidence struct {
    MemoryID string
    Alpha    float64  // starts at 1
    Beta     float64  // starts at 1
}

// Confidence score = Alpha / (Alpha + Beta)

func (mc *MemoryConfidence) Update(signal Signal, weights ProjectWeights) {
    w := weights.WeightFor(signal.Type)

    if signal.Positive {
        mc.Alpha += w
    } else {
        mc.Beta += w
    }
}

func (mc *MemoryConfidence) Score() float64 {
    return mc.Alpha / (mc.Alpha + mc.Beta)
}
```

**Example Flow:**

```
Memory: "Use context.WithTimeout for database calls"
Initial: Alpha=1, Beta=1 → Confidence=0.50

Day 1: Retrieved in search (usage signal, positive)
       Usage weight = 0.33
       Alpha = 1.33 → Confidence = 0.57

Day 2: Agent reports task succeeded (outcome signal, positive)
       Outcome weight = 0.33
       Alpha = 1.66 → Confidence = 0.62

Day 5: User rates "helpful" (explicit signal, positive)
       Explicit weight = 0.40
       Alpha = 2.06 → Confidence = 0.67
```

### Hybrid Storage

**Event Log (last 30 days):** Full signal detail for recency analysis.

**Aggregates (older data):** Rolled-up counts for storage efficiency.

```go
func (s *Service) ComputeConfidence(memoryID string) float64 {
    agg := s.getAggregate(memoryID)
    recentSignals := s.getRecentSignals(memoryID, 30*24*time.Hour)
    weights := s.getProjectWeights()

    // Start from aggregate
    alpha := 1.0 + float64(agg.ExplicitPos)*weights.Explicit +
                   float64(agg.UsagePos)*weights.Usage +
                   float64(agg.OutcomePos)*weights.Outcome
    beta := 1.0 + float64(agg.ExplicitNeg)*weights.Explicit +
                  float64(agg.UsageNeg)*weights.Usage +
                  float64(agg.OutcomeNeg)*weights.Outcome

    // Apply recent signals
    for _, sig := range recentSignals {
        w := weights.WeightFor(sig.Type)
        if sig.Positive {
            alpha += w
        } else {
            beta += w
        }
    }

    return alpha / (alpha + beta)
}
```

**Rollup Process (daily):**
1. Find signals older than 30 days
2. Group by memory_id, increment aggregate counts
3. Delete rolled-up events
4. Recalculate memory confidence
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
