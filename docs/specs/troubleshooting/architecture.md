# Troubleshooting Architecture

**Parent**: [../SPEC.md](../SPEC.md)

## Service Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Troubleshooting Service                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌──────────────────┐ │
│  │   Diagnosis    │  │    Pattern     │  │     Session      │ │
│  │    Engine      │  │   Retrieval    │  │   Management     │ │
│  └────────────────┘  └────────────────┘  └──────────────────┘ │
│          │                   │                      │           │
│          └───────────────────┴──────────────────────┘           │
│                              │                                   │
│  ┌───────────────────────────┴────────────────────────────────┐│
│  │              Service Core (business logic)                  ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  ┌───────────────────────────┴────────────────────────────────┐│
│  │         Universal VectorStore Interface                     ││
│  │  - CreateDatabase, Insert, Search, Delete                   ││
│  │  - Multi-tenant isolation (shared database)                 ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
└──────────────────────────────┼───────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
┌───────▼────────┐   ┌─────────▼────────┐   ┌────────▼────────┐
│ (Native DBs)   │   │ (Collection      │   │ (Weaviate,      │
│                │   │  Prefixes)       │   │  Pinecone, etc.)│
└────────────────┘   └──────────────────┘   └─────────────────┘
        │                      │                      │
        └──────────────────────┴──────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Vector Database     │
                    │  - Shared DB         │
                    │  - Collection:       │
                    │    troubleshooting_  │
                    │    knowledge         │
                    └─────────────────────┘
```

## Component Responsibilities

### Diagnosis Engine

- Orchestrates 5-step troubleshooting process
- Generates hypotheses from similar issues
- Ranks hypotheses by probability
- Creates recommended actions
- Detects destructive operations

### Pattern Retrieval

- Semantic search using vector embeddings
- Hybrid scoring (semantic + success rate + usage)
- Metadata filtering (category, severity, tags)
- Result reranking

### Session Management

- Track complete diagnostic sessions
- Record actions performed
- Store resolutions and feedback
- Enable learning from outcomes

## Data Flow

```
┌──────────────┐
│ User Request │ (Error message, stack trace, context)
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ 1. Generate Embedding                        │
│    - Embedder.Embed(error_message)           │
│    - Returns: []float32 (1536 dimensions)    │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ 2. Search Similar Issues                     │
│    - VectorStore.Search(embedding, filters)  │
│    - Database: "shared"                      │
│    - Collection: "troubleshooting_knowledge" │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ 3. Calculate Hybrid Scores                   │
│    - Semantic: 60%                           │
│    - Success Rate: 30%                       │
│    - Usage Frequency: 10%                    │
│    - Rerank results                          │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ 4. Generate Hypotheses                       │
│    - Extract root causes                     │
│    - Calculate probabilities                 │
│    - Aggregate evidence                      │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ 5. Create Actions & Session                  │
│    - Verification steps                      │
│    - Solution steps                          │
│    - Safety warnings                         │
│    - Session tracking                        │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────┐
│   Response   │ (Diagnosis, similar issues, actions)
└──────────────┘
```

## Storage Architecture

### Storage Strategy

Troubleshooting patterns are stored in the **shared database** for universal access:

```
shared/                          # Global troubleshooting knowledge
├── troubleshooting_knowledge    # Collection for all patterns
│   ├── pattern_1 (vector + metadata)
│   ├── pattern_2 (vector + metadata)
│   └── ...
```

**Benefits**:
- All projects can access global troubleshooting knowledge
- No project-specific filtering needed (patterns are universal)
- Better performance through database isolation
- Easier pattern sharing and collaboration

### Indexing Strategy

**Vector Indexing**:
- **Algorithm**: IVF_FLAT (Inverted File with Flat Index)
- **Clusters**: 128 (IVFSearchNProbe constant)
- **Distance Metric**: Cosine similarity
- **Dimension**: 1536 (standard embedding size)

**Metadata Indexing**:
- **category**: String field for filtering
- **severity**: String field for filtering
- **tags**: String field (comma-separated) for pattern matching
- **success_rate**: Float field for hybrid scoring
- **usage_count**: Int32 field for hybrid scoring

## Data Models

### TroubleshootingKnowledge

Core knowledge record stored in vector database.

```go
type TroubleshootingKnowledge struct {
    ID              string            `json:"id"`
    ErrorPattern    string            `json:"error_pattern"`
    Context         string            `json:"context"`     // kubernetes, docker, git, general
    RootCause       string            `json:"root_cause"`
    Solution        string            `json:"solution"`
    DiagnosticSteps string            `json:"diagnostic_steps"`
    SuccessRate     float32           `json:"success_rate"` // 0.0 - 1.0
    Severity        Severity          `json:"severity"`     // critical, high, medium, low
    Category        string            `json:"category"`     // configuration, resource, etc.
    Tags            []string          `json:"tags"`
    Metadata        map[string]string `json:"metadata,omitempty"`
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
    LastUsed        time.Time         `json:"last_used"`
    UsageCount      int32             `json:"usage_count"`
}
```

### Hypothesis

Potential root cause with supporting evidence.

```go
type Hypothesis struct {
    Description       string   `json:"description"`
    Probability       float64  `json:"probability"`    // 0.0 - 1.0
    Evidence          []string `json:"evidence"`
    Category          string   `json:"category"`
    VerificationSteps []string `json:"verification_steps"`
}
```

### Action

Recommended troubleshooting action.

```go
type Action struct {
    Step        int      `json:"step"`
    Description string   `json:"description"`
    Commands    []string `json:"commands,omitempty"`
    Expected    string   `json:"expected_outcome"`
    Destructive bool     `json:"destructive"`  // Requires confirmation
    Safety      string   `json:"safety_notes,omitempty"`
}
```

**Destructive Operation Detection**:

Keywords that trigger destructive flag:
- `delete`, `remove`, `drop`, `destroy`
- `restart`, `kill`, `terminate`
- `wipe`, `format`, `reset`

Safety note automatically added: "CAUTION: This action may cause service disruption. Confirm before proceeding."

## Hybrid Matching Algorithm

### Scoring Components

#### 1. Semantic Similarity (60% weight)

Measures vector similarity between query and stored patterns.

```go
// Calculate semantic score from vector distance
semanticScore = 1.0 / (1.0 + distance)

// Example distances:
// distance = 0.0  → semanticScore = 1.0 (perfect match)
// distance = 0.5  → semanticScore = 0.67
// distance = 1.0  → semanticScore = 0.5
// distance = 2.0  → semanticScore = 0.33
```

#### 2. Success Rate (30% weight)

Historical success rate of the solution.

```go
// Success rate stored directly (0.0 - 1.0)
// Tracked through feedback loop:
// - Initial: 0.0 (no data)
// - After 10 successful applications: 1.0
// - After 8/10 successful: 0.8
```

#### 3. Usage Frequency (10% weight)

How often the pattern has been used.

```go
// Normalize usage count to 0-1 range
usageScore = min(usageCount / 100.0, 1.0)

// Example:
// usageCount = 0   → usageScore = 0.0
// usageCount = 50  → usageScore = 0.5
// usageCount = 100 → usageScore = 1.0
// usageCount = 200 → usageScore = 1.0 (capped)
```

### Final Hybrid Score

```go
const (
    WeightSemanticScore  = 0.6
    WeightSuccessRate    = 0.3
    WeightUsageFrequency = 0.1
)

hybridScore = (semanticScore * 0.6) +
              (successRate * 0.3) +
              (usageScore * 0.1)
```

### Example Calculation

```
Pattern A:
- Semantic: 0.9 (very similar)
- Success: 0.5 (moderate success rate)
- Usage: 0.2 (used 20 times)
- Hybrid: (0.9 * 0.6) + (0.5 * 0.3) + (0.2 * 0.1) = 0.54 + 0.15 + 0.02 = 0.71

Pattern B:
- Semantic: 0.7 (somewhat similar)
- Success: 0.95 (very high success rate)
- Usage: 0.8 (used 80 times)
- Hybrid: (0.7 * 0.6) + (0.95 * 0.3) + (0.8 * 0.1) = 0.42 + 0.285 + 0.08 = 0.785

Result: Pattern B ranks higher (0.785 > 0.71)
Rationale: High success rate and usage compensate for lower semantic similarity
```

## Integration Points

### Internal Dependencies

1. **pkg/vectorstore**: Universal vector store interface
2. **pkg/embedding**: Embedding generation service
3. **pkg/telemetry**: OpenTelemetry instrumentation
4. **pkg/validation**: Request validation

### External Dependencies

1. **Vector Database**: Qdrant (local instance)
2. **Embedding Service**: OpenAI API or TEI (local)
3. **Monitoring**: OpenTelemetry collector (optional)

### Service Integration

**MCP Server**:
- Registers `troubleshoot` and `list_patterns` tools
- Handles JSON-RPC requests from Claude Code
- Translates between MCP protocol and service API

**HTTP Handlers**:
- Exposes REST API for troubleshooting operations
- HTTP transport on port 8080 (no authentication)
- Returns standardized JSON responses

**Observability**:
- OpenTelemetry traces for all operations
- Metrics for diagnosis performance
- Pattern match tracking
- Success rate monitoring
