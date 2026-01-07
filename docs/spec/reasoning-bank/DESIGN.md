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

## Memory Consolidation

### Overview

Memory consolidation prevents knowledge rot by automatically merging similar memories into synthesized, higher-value knowledge. As memories accumulate over time, duplicates and near-duplicates emerge from similar problem-solving patterns. Consolidation uses vector similarity detection and LLM-powered synthesis to:

1. **Detect similarity clusters** - Find groups of memories with >0.8 cosine similarity
2. **Synthesize consolidated knowledge** - Use LLM to merge memories into comprehensive patterns
3. **Preserve source attribution** - Archive original memories with back-links for traceability
4. **Boost search relevance** - Consolidated memories rank 20% higher in search results

**Why Consolidation Matters:**
- **Reduces redundancy** - 10 similar "timeout handling" memories become 1 comprehensive strategy
- **Improves search quality** - Consolidated memories represent consensus from multiple experiences
- **Maintains knowledge lineage** - Original memories preserved for attribution and auditing
- **Prevents storage bloat** - Archived memories excluded from normal searches but retained for reference

### Architecture

#### 1. Similarity Detection Engine

Uses vector embeddings to find clusters of related memories:

```
Memory Pool (Project: web-app)
├─ "Use context.WithTimeout for database calls" (confidence: 0.75, usage: 10)
├─ "Always set DB query timeouts" (confidence: 0.82, usage: 8)
└─ "Database timeout best practices" (confidence: 0.70, usage: 15)

   ↓ (Embedding vectors computed)

Similarity Matrix (cosine similarity)
├─ Memory1 ↔ Memory2: 0.87
├─ Memory1 ↔ Memory3: 0.84
└─ Memory2 ↔ Memory3: 0.89

   ↓ (Greedy clustering with threshold 0.8)

SimilarityCluster {
  Members: [Memory1, Memory2, Memory3],
  AverageSimilarity: 0.87,
  MinSimilarity: 0.84
}
```

**Algorithm:**
1. Retrieve all active memories for a project
2. Compute embedding vectors for each memory (title + content)
3. Calculate pairwise cosine similarity between all memories
4. Group memories with similarity > threshold (default 0.8) into clusters
5. Only form clusters with ≥2 members

**Implementation:** `Distiller.FindSimilarClusters(ctx, projectID, threshold)`

#### 2. LLM-Powered Synthesis

Each similarity cluster is sent to an LLM for synthesis into consolidated knowledge:

**Consolidation Prompt Template:**

```
You are consolidating similar memories into a single synthesized memory.

Source Memories:
---
Memory 1:
Title: Use context.WithTimeout for database calls
Content: Always wrap database operations with context.WithTimeout to prevent hanging queries...
Tags: go, database, timeout
Confidence: 0.75, Usage: 10

Memory 2:
Title: Always set DB query timeouts
Content: Database queries should have explicit timeouts to avoid resource exhaustion...
Tags: database, best-practice
Confidence: 0.82, Usage: 8

Memory 3:
Title: Database timeout best practices
Content: Implement timeouts on all database operations using context package...
Tags: go, database, patterns
Confidence: 0.70, Usage: 15
---

Your Task:
1. Identify the common theme across these memories
2. Synthesize the key insights into a coherent, comprehensive strategy
3. Preserve important details that shouldn't be lost
4. Note when and how to apply this consolidated knowledge

Output Format:
TITLE: <concise title>
CONTENT: <synthesized strategy with specific steps>
TAGS: <comma-separated tags>
OUTCOME: success|failure
SOURCE_ATTRIBUTION: <brief description of sources>
```

**LLM Response:**

```
TITLE: Database Timeout Management in Go
CONTENT: Always use context.WithTimeout for database operations to prevent resource exhaustion and hanging queries. Best practices: 1) Set timeout based on expected query duration (typically 5-30s for OLTP). 2) Wrap all DB calls (queries, transactions, connections) with timeout context. 3) Handle context.DeadlineExceeded errors gracefully. 4) Log timeout events for monitoring. This prevents cascading failures when database performance degrades.
TAGS: go, database, timeout, best-practice, patterns
OUTCOME: success
SOURCE_ATTRIBUTION: Synthesized from 3 source memories about database timeout handling with combined usage count of 33
```

**Implementation:** `Distiller.MergeCluster(ctx, cluster)`

#### 3. Confidence & Attribution System

**Confidence Calculation:**

Consolidated memories use a weighted average of source confidences:

```go
confidence = Σ(source_confidence_i * (usage_count_i + 1)) / Σ(usage_count_i + 1)
```

- High-usage memories contribute more weight to final confidence
- Frequently-used high-confidence memories dominate the score
- Consensus bonus (up to +0.1) for low variance across sources

**Example:**
```
Source 1: confidence=0.75, usage=10 → weight=11, contribution=8.25
Source 2: confidence=0.82, usage=8  → weight=9,  contribution=7.38
Source 3: confidence=0.70, usage=15 → weight=16, contribution=11.2

Final confidence = (8.25 + 7.38 + 11.2) / (11 + 9 + 16) = 0.745
```

**Source Attribution:**

- **Consolidated Memory**: Description field contains LLM-generated attribution text
- **Source Memories**: State changed to "archived", ConsolidationID set to consolidated memory ID
- **Bidirectional links**: Navigate from consolidated → sources (via ConsolidationID back-references) and sources → consolidated (via ConsolidationID field)

**Search Behavior:**
- Archived memories are filtered from normal search results
- Consolidated memories receive a 20% relevance boost
- Attribution preserved for traceability and auditing

#### 4. Consolidation Tracking

To prevent redundant processing, the system tracks last consolidation time per project:

```go
// Consolidation window: 24 hours (default)
lastConsolidation[projectID] = time.Now()

// Skip if consolidated within window
if time.Since(lastConsolidation[projectID]) < 24*time.Hour {
  return EmptyResult // Unless ForceAll=true
}
```

**Behavior:**
- Prevents re-processing recently consolidated memories
- Configurable window (default: 24h)
- ForceAll option bypasses window check
- DryRun mode doesn't update timestamp

### MCP Tool Usage

#### memory_consolidate

Manually trigger consolidation for a specific project.

**Tool Definition:**

```json
{
  "name": "memory_consolidate",
  "description": "Consolidate similar memories to reduce redundancy and improve knowledge quality. Merges memories with similarity above threshold into synthesized consolidated memories.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_id": {
        "type": "string",
        "description": "Project ID to consolidate memories for",
        "required": true
      },
      "similarity_threshold": {
        "type": "number",
        "description": "Minimum cosine similarity (0.0-1.0) for consolidation",
        "default": 0.8
      },
      "dry_run": {
        "type": "boolean",
        "description": "Preview consolidation without making changes",
        "default": false
      },
      "max_clusters": {
        "type": "integer",
        "description": "Maximum number of clusters to process (0=no limit)",
        "default": 0
      }
    }
  }
}
```

**Response:**

```json
{
  "created_memories": ["mem_abc123", "mem_def456"],
  "archived_memories": ["mem_old001", "mem_old002", "mem_old003", "mem_old004"],
  "skipped_count": 5,
  "total_processed": 9,
  "duration_seconds": 12.45
}
```

**Usage Example:**

```javascript
// Basic consolidation with defaults
await mcp.callTool("memory_consolidate", {
  project_id: "web-app-backend"
});

// Custom threshold and dry run preview
await mcp.callTool("memory_consolidate", {
  project_id: "web-app-backend",
  similarity_threshold: 0.85,
  dry_run: true
});

// Limit processing for resource control
await mcp.callTool("memory_consolidate", {
  project_id: "web-app-backend",
  max_clusters: 10
});
```

**Typical Workflow:**

1. **Preview with dry run** - See what would be consolidated without making changes
2. **Review proposed consolidations** - Check created/archived counts
3. **Execute consolidation** - Run without dry_run to apply changes
4. **Verify results** - Search for consolidated memories to validate synthesis quality

### Automatic Consolidation Scheduler

Background scheduler for hands-free memory consolidation.

#### Configuration

**config.yaml:**

```yaml
consolidation_scheduler:
  enabled: true                # Enable automatic consolidation
  interval: 24h                # Time between consolidation runs
  similarity_threshold: 0.8    # Similarity threshold for clustering
```

**Environment Variables:**

```bash
export CONSOLIDATION_SCHEDULER_ENABLED=true
export CONSOLIDATION_SCHEDULER_INTERVAL=12h
export CONSOLIDATION_SCHEDULER_SIMILARITY_THRESHOLD=0.85
```

#### Behavior

- **Scheduled Runs**: Consolidation executes every `interval` (default: 24h)
- **Project Scope**: Processes all configured projects in a single run
- **Graceful Errors**: Individual project failures don't stop the scheduler
- **Resource Limits**: 10-minute timeout per consolidation run
- **Lifecycle**: Starts with contextd, stops gracefully on shutdown

#### Implementation Details

```go
// Created at startup if enabled
scheduler := NewConsolidationScheduler(distiller, logger,
  WithInterval(cfg.ConsolidationScheduler.Interval),
  WithProjectIDs([]string{"project-1", "project-2"}),
  WithConsolidationOptions(ConsolidationOptions{
    SimilarityThreshold: cfg.ConsolidationScheduler.SimilarityThreshold,
    DryRun:              false,
    ForceAll:            false,
    MaxClustersPerRun:   0,
  }),
)

scheduler.Start() // Begins background goroutine

// On shutdown
scheduler.Stop()  // Gracefully stops ticker
```

**Logging:**

```
INFO  Starting consolidation scheduler  interval=24h threshold=0.8
DEBUG Consolidation run started          projects=2
INFO  Consolidation complete             created=3 archived=9 duration=8.2s
WARN  Project consolidation failed       project=web-app error="timeout exceeded"
```

### Configuration Options Reference

#### ConsolidationOptions

```go
type ConsolidationOptions struct {
  // SimilarityThreshold: Minimum cosine similarity (0.0-1.0) for clustering.
  // Higher = stricter matching, lower = looser grouping.
  // Default: 0.8 (recommended for most use cases)
  SimilarityThreshold float64

  // MaxClustersPerRun: Limit clusters processed per run (0 = no limit).
  // Use to control resource usage and runtime duration.
  // Default: 0 (process all clusters)
  MaxClustersPerRun int

  // DryRun: Preview consolidation without making changes.
  // Returns what would be created/archived without executing.
  // Default: false
  DryRun bool

  // ForceAll: Bypass consolidation window check.
  // Forces consolidation even if run recently.
  // Default: false
  ForceAll bool
}
```

#### ConsolidationResult

```go
type ConsolidationResult struct {
  // CreatedMemories: IDs of newly created consolidated memories
  CreatedMemories []string

  // ArchivedMemories: IDs of source memories that were archived
  ArchivedMemories []string

  // SkippedCount: Memories evaluated but not consolidated
  SkippedCount int

  // TotalProcessed: Total memories evaluated
  TotalProcessed int

  // Duration: Time taken for consolidation operation
  Duration time.Duration
}
```

#### Memory State Transitions

```
Active Memory (state=active, consolidation_id=nil)
  │
  │ [Consolidation runs, memory part of cluster]
  ▼
Archived Memory (state=archived, consolidation_id=<consolidated_id>)
  │
  │ [Original content preserved, excluded from search]
  │
  └─► Consolidated Memory (state=active, consolidation_id=nil)
       │
       └─► Description contains source attribution
```

### Consolidation Flow

#### Manual Trigger via MCP Tool

```
Agent                MCP Server           Distiller           LLM           VectorStore
  │                      │                    │                │                │
  │─ memory_consolidate ►│                    │                │                │
  │  (project_id, opts)  │                    │                │                │
  │                      │─── Consolidate() ─►│                │                │
  │                      │                    │─ ListMemories ────────────────►│
  │                      │                    │◄── memories ───────────────────│
  │                      │                    │                │                │
  │                      │                    │─ GetMemoryVector (for each) ──►│
  │                      │                    │◄── vectors ────────────────────│
  │                      │                    │                │                │
  │                      │                    │ [compute similarity matrix]    │
  │                      │                    │ [form clusters]                │
  │                      │                    │                │                │
  │                      │                    │─ synthesize ──►│                │
  │                      │                    │◄── consolidated │               │
  │                      │                    │    response     │               │
  │                      │                    │                │                │
  │                      │                    │─ Record(consolidated) ─────────►│
  │                      │                    │─ Update(sources, archived) ────►│
  │                      │◄── result ─────────│                │                │
  │◄── statistics ───────│                    │                │                │
```

#### Automatic Scheduler Trigger

```
Scheduler            Distiller           LLM           VectorStore
  │                      │                │                │
  │ [timer fires]        │                │                │
  │                      │                │                │
  │─ ConsolidateAll() ──►│                │                │
  │  (projectIDs, opts)  │                │                │
  │                      │                │                │
  │                      │ [for each project]             │
  │                      │─ Consolidate() ────────────────►│
  │                      │                │                │
  │                      │ [same flow as manual trigger]  │
  │                      │                │                │
  │◄── aggregated ───────│                │                │
  │    results           │                │                │
  │                      │                │                │
  │ [log statistics]     │                │                │
  │ [wait for next interval]             │                │
```

### Testing Strategy

#### Unit Tests

- **Similarity Detection**
  - CosineSimilarity function with known vectors
  - FindSimilarClusters with mock embeddings
  - Cluster statistics calculation (centroid, avg/min similarity)
  - Threshold boundary conditions

- **LLM Synthesis**
  - buildConsolidationPrompt formatting
  - parseConsolidatedMemory response parsing
  - MergeCluster with mock LLM
  - Confidence calculation (weighted average)
  - Source attribution storage

- **Memory Linking**
  - ConsolidationID back-reference setting
  - State transition (active → archived)
  - Original content preservation
  - Bidirectional navigation

- **Consolidate Orchestration**
  - Full consolidation workflow
  - DryRun mode (no changes)
  - MaxClustersPerRun limit enforcement
  - Error handling (LLM failures, partial failures)
  - Statistics accuracy (created/archived/skipped counts)

- **Scheduler**
  - Start/Stop lifecycle
  - Interval-based triggering
  - Project list configuration
  - Graceful shutdown
  - Error resilience (continues after failures)

#### Integration Tests

- **End-to-End Consolidation**
  - Create similar memories with varying confidence/usage
  - Run consolidation with threshold 0.8
  - Verify consolidated memory properties
  - Check ConsolidationID back-links on source memories
  - Validate search filters archived memories
  - Confirm consolidated memories rank higher

- **Similarity Threshold Validation**
  - Memories >0.8 similarity are consolidated
  - Memories <0.8 similarity remain separate
  - Only similar memories archived, dissimilar stay active

- **Original Content Preservation**
  - Source memories retain all original fields
  - No data loss during consolidation
  - Consolidated memory has different synthesized content

- **Confidence Calculation**
  - Equal confidence/usage → simple average
  - High usage dominates → weighted toward frequent memories
  - Mixed scenarios → correct weighted formula
  - Edge cases (zero usage, same confidence)

- **Manual vs Automatic Triggers**
  - MCP tool manual trigger executes consolidation
  - Scheduler automatic trigger fires on interval
  - Both use same infrastructure and produce valid results
  - Dry run works with both trigger types

- **Source Attribution**
  - Description contains meaningful attribution text
  - Attribution references source memory count/content
  - Source IDs retrievable via ConsolidationResult
  - Bidirectional relationship (consolidated ↔ sources)

#### Performance Tests

- **Consolidation latency** - <30s for 1000 memories
- **LLM synthesis throughput** - Handle 10 clusters concurrently
- **Memory scaling** - Consolidate projects with 10K+ memories
- **Search boost impact** - Verify 20% ranking improvement
