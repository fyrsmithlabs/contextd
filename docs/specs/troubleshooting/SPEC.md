# Troubleshooting Service Specification

**Package**: `pkg/troubleshooting`
**Status**: Implemented
**Version**: 1.0.0
**Last Updated**: 2025-11-04

## Table of Contents

1. [Overview](#overview)
2. [Features and Capabilities](#features-and-capabilities)
3. [AI-Powered Diagnosis Workflow](#ai-powered-diagnosis-workflow)
4. [Architecture and Design](#architecture-and-design)
5. [API Specifications](#api-specifications)
6. [Data Models and Schemas](#data-models-and-schemas)
7. [Pattern Storage and Retrieval](#pattern-storage-and-retrieval)
8. [Hybrid Matching Algorithm](#hybrid-matching-algorithm)
9. [Performance Characteristics](#performance-characteristics)
10. [Error Handling](#error-handling)
11. [Security Considerations](#security-considerations)
12. [Testing Requirements](#testing-requirements)
13. [Usage Examples](#usage-examples)
14. [Integration Points](#integration-points)

## Overview

### Purpose

The troubleshooting service provides AI-powered error diagnosis and troubleshooting capabilities for contextd. It analyzes error messages and stack traces, identifies root causes, generates hypotheses, and recommends diagnostic steps and solutions based on historical knowledge and semantic pattern matching.

### Core Philosophy

**Primary Goals**:
1. **Automated Learning**: Continuously learn from resolved errors to build knowledge base
2. **Intelligent Diagnosis**: Use AI to identify root causes and generate actionable recommendations
3. **Context Efficiency**: Reduce trial-and-error debugging time through semantic pattern matching
4. **Safety-First**: Detect and warn about destructive operations in recommended solutions

### Key Differentiators

- **Hybrid Matching**: Combines semantic similarity (60%), success rate (30%), and usage frequency (10%) for intelligent ranking
- **Progressive Disclosure**: Returns information based on confidence level - high confidence includes detailed timeline and affected resources
- **Safety Detection**: Automatically identifies destructive operations (delete, remove, drop, kill, restart) and adds warnings
- **Multi-Tenant Isolation**: Stores global troubleshooting knowledge in shared database accessible to all projects
- **Feedback Loop**: Tracks success rates and usage patterns to improve recommendations over time

## Features and Capabilities

### Core Features

1. **AI-Powered Error Diagnosis**
   - Semantic error pattern matching
   - Root cause identification
   - Hypothesis generation with probability scoring
   - Verification step recommendations
   - Solution generation based on similar issues

2. **Troubleshooting Knowledge Base**
   - Store error patterns with solutions
   - Organize by category (configuration, resource, dependency, permission, logic, network, storage)
   - Classify by severity (critical, high, medium, low)
   - Track success rates and usage patterns
   - Tag for improved searchability

3. **Intelligent Pattern Recognition**
   - Semantic search using vector embeddings
   - Hybrid scoring combining similarity, success rate, and usage
   - Metadata filtering (category, severity, tags)
   - Reranking for optimal result ordering

4. **Interactive Troubleshooting Sessions**
   - Track complete diagnostic sessions
   - Record actions performed
   - Capture resolution outcomes
   - Store feedback for continuous learning

5. **Observability and Monitoring**
   - OpenTelemetry instrumentation
   - Metrics for diagnosis performance
   - Pattern match tracking
   - Success rate monitoring
   - Hypothesis generation tracking

### MCP Tool Integration

The troubleshooting service exposes two MCP tools:

1. **`troubleshoot`**: AI-powered error diagnosis
   - Analyzes error messages and stack traces
   - Searches similar issues in knowledge base
   - Generates hypotheses with evidence
   - Recommends diagnostic steps and solutions
   - Returns session ID for tracking

2. **`list_patterns`**: Browse troubleshooting patterns
   - Filter by category, severity, success rate
   - Paginated results
   - Useful for learning from past solutions

## AI-Powered Diagnosis Workflow

### 5-Step Troubleshooting Process

The service implements a comprehensive 5-step process based on industry best practices:

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: Symptom Collection                                      │
│ - Error message, stack trace, context                           │
│ - Environment metadata (file, line, language)                   │
│ - Mode selection (auto, interactive, guided)                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 2: Pattern Recognition (Semantic Search)                   │
│ - Generate embedding for error message                          │
│ - Vector similarity search in knowledge base                    │
│ - Filter by category, tags (optional)                           │
│ - Rerank by hybrid score (semantic + success + usage)           │
│ - Return top N similar issues (default: 5)                      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 3: Hypothesis Formation                                    │
│ - Extract root causes from similar issues                       │
│ - Calculate probability: match_score * success_rate             │
│ - Aggregate evidence for recurring patterns                     │
│ - Generate verification steps for each hypothesis               │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 4: Hypothesis Ranking                                      │
│ - Normalize probabilities (sum to 1.0)                          │
│ - Sort by probability (descending)                              │
│ - Select top hypothesis as most likely root cause               │
│ - Determine confidence level (high: ≥0.8, medium: ≥0.5)         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 5: Action Generation & Resolution                          │
│ - Extract verification steps from top hypothesis                │
│ - Extract solution steps from best matching issue               │
│ - Detect destructive operations (delete, remove, kill, etc.)    │
│ - Add safety warnings for destructive steps                     │
│ - Return recommended actions with expected outcomes             │
└─────────────────────────────────────────────────────────────────┘
```

### Confidence Levels

The diagnosis assigns confidence levels based on match quality:

| Confidence | Probability | Meaning | Behavior |
|-----------|-------------|---------|----------|
| **High** | ≥ 0.8 | Strong match, high success rate | Include detailed timeline, affected resources, solution directly actionable |
| **Medium** | 0.5 - 0.79 | Moderate match, some uncertainty | Include general recommendations, suggest verification steps |
| **Low** | < 0.5 | Weak match or novel issue | Recommend manual investigation, external documentation search |

### Progressive Disclosure

The service returns information based on confidence level to avoid overwhelming users with low-quality data:

**High Confidence (≥0.8)**:
- Root cause with evidence
- Detailed timeline of events
- Affected resources list
- Direct solution steps
- Expected outcomes for each step

**Medium Confidence (0.5-0.79)**:
- Root cause with caveats
- Verification steps to confirm hypothesis
- General diagnostic guidance
- Similar issues for reference

**Low Confidence (<0.5)**:
- Generic troubleshooting guidance
- Recommendation to search external docs
- Suggestion to create new knowledge entry after resolution

## Architecture and Design

### Service Architecture

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

### Component Responsibilities

**Diagnosis Engine**:
- Orchestrates 5-step troubleshooting process
- Generates hypotheses from similar issues
- Ranks hypotheses by probability
- Creates recommended actions
- Detects destructive operations

**Pattern Retrieval**:
- Semantic search using vector embeddings
- Hybrid scoring (semantic + success rate + usage)
- Metadata filtering (category, severity, tags)
- Result reranking

**Session Management**:
- Track complete diagnostic sessions
- Record actions performed
- Store resolutions and feedback
- Enable learning from outcomes

### Data Flow

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

## API Specifications

### Service API (Internal)

#### Diagnose

Performs complete 5-step troubleshooting diagnosis.

```go
func (s *Service) Diagnose(
    ctx context.Context,
    req *DiagnosisRequest,
) (*Session, error)
```

**Request**:
```go
type DiagnosisRequest struct {
    ErrorMessage string            // Required: Error message or exception
    StackTrace   string            // Optional: Stack trace if available
    Context      map[string]string // Optional: Additional context (file, line, etc.)
    Mode         Mode              // auto, interactive, guided (default: auto)
    Category     string            // Optional: Filter by category
    Tags         []string          // Optional: Filter by tags
    TopK         int               // Number of similar issues (default: 5, max: 50)
    MinScore     float64           // Minimum match score (default: 0.5, range: 0-1)
}
```

**Response**:
```go
type Session struct {
    ID               string         // Unique session ID
    Status           string         // in_progress, completed, failed
    Diagnosis        Diagnosis      // Complete diagnosis result
    SimilarIssues    []SimilarIssue // Top matching similar issues
    RecommendedSteps []Action       // Recommended troubleshooting actions
    StartedAt        time.Time      // Session start time
    CompletedAt      *time.Time     // Session completion time (if completed)
    Outcome          string         // success, failure, escalated
}

type Diagnosis struct {
    ErrorMessage       string
    StackTrace         string
    Context            map[string]string
    RootCause          string       // Most likely root cause
    Hypotheses         []Hypothesis // All hypotheses with probabilities
    Confidence         string       // high, medium, low
    ConfidenceScore    float64      // 0.0 - 1.0
    Category           string
    Severity           Severity
    SimilarIssues      []SimilarIssue
    RecommendedActions []Action
    DiagnosticSteps    []string
    AffectedResources  []string                 // High confidence only
    Timeline           []map[string]interface{} // High confidence only
    DiagnosisID        string
    TimeTaken          float64 // milliseconds
    DiagnosedAt        time.Time
}
```

#### SearchSimilarIssues

Performs semantic search for similar troubleshooting issues.

```go
func (s *Service) SearchSimilarIssues(
    ctx context.Context,
    errorMessage string,
    contextMap map[string]string,
    category string,
    tags []string,
    limit int,
) ([]SimilarIssue, error)
```

**Returns**:
```go
type SimilarIssue struct {
    ID             string
    Knowledge      TroubleshootingKnowledge
    MatchScore     float64 // Hybrid score (0.0 - 1.0)
    SemanticScore  float64 // Vector similarity (0.0 - 1.0)
    MetadataMatch  bool    // Category/tag match
    SuccessRate    float32
    Confidence     string
    Solution       string
    Tags           []string
    IsDestructive  bool
    SafetyWarnings []string
}
```

#### StoreResolution

Stores a new troubleshooting resolution in knowledge base.

```go
func (s *Service) StoreResolution(
    ctx context.Context,
    req *StoreKnowledgeRequest,
) (*TroubleshootingKnowledge, error)
```

**Request**:
```go
type StoreKnowledgeRequest struct {
    ErrorPattern    string            // Required: Error pattern
    Context         string            // Required: Context (kubernetes, docker, git, etc.)
    RootCause       string            // Required: Identified root cause
    Solution        string            // Required: Solution that worked
    DiagnosticSteps string            // Optional: Steps to verify
    Severity        string            // Required: critical, high, medium, low
    Category        string            // Required: Error category
    Tags            []string          // Optional: Tags for searchability
    Metadata        map[string]string // Optional: Additional metadata
}
```

#### ListPatterns

Lists troubleshooting patterns with filtering.

```go
func (s *Service) ListPatterns(
    ctx context.Context,
    category string,
    severity Severity,
    minSuccessRate float64,
) ([]Pattern, error)
```

**Returns**:
```go
type Pattern struct {
    ID           string
    ErrorPattern string
    Category     string
    Severity     Severity
    RootCause    string
    Solution     string
    SuccessRate  float32
    Tags         []string
    UsageCount   int32
    LastUsed     time.Time
}
```

### HTTP API (via handlers)

#### POST /api/v1/troubleshoot

Diagnoses an error and provides remediation guidance.

**Request**:
```json
{
  "error_message": "panic: runtime error: invalid memory address",
  "stack_trace": "goroutine 1 [running]:\nmain.main()\n  /path/to/main.go:42",
  "context": {
    "file": "main.go",
    "line": "42",
    "language": "go"
  },
  "mode": "auto"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "in_progress",
    "diagnosis": {
      "root_cause": "Nil pointer dereference",
      "category": "logic",
      "severity": "high",
      "confidence": {
        "level": "high",
        "score": 0.85
      }
    },
    "similar_issues": [
      {
        "id": "abc-123",
        "match_score": 0.92,
        "confidence": "high",
        "solution": "Check for nil before dereferencing pointer",
        "tags": ["go", "nil-pointer", "runtime"],
        "destructive": false
      }
    ],
    "recommended_actions": [
      {
        "step": 1,
        "description": "Add nil check before accessing pointer",
        "expected_outcome": "Prevent panic at runtime",
        "destructive": false
      }
    ]
  },
  "meta": {
    "confidence": "high",
    "count": 3
  }
}
```

#### GET /api/v1/troubleshoot/patterns

Lists troubleshooting patterns with filtering.

**Query Parameters**:
- `category` (optional): Filter by category
- `severity` (optional): Filter by severity (critical, high, medium, low)
- `min_success_rate` (optional): Minimum success rate (0.0 - 1.0)
- `limit` (optional): Results per page (default: 20, max: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response**:
```json
{
  "success": true,
  "data": {
    "patterns": [
      {
        "id": "pattern-123",
        "error_pattern": "connection refused",
        "category": "network",
        "severity": "high",
        "root_cause": "Service not running",
        "solution": "Start the service",
        "success_rate": 0.95,
        "tags": ["network", "connection"],
        "usage_count": 42,
        "last_used": "2025-11-04T10:30:00Z"
      }
    ]
  },
  "meta": {
    "count": 20,
    "limit": 20,
    "offset": 0
  }
}
```

### MCP Tools

#### troubleshoot

AI-powered error diagnosis and troubleshooting.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "error_message": {
      "type": "string",
      "description": "Error message or exception text (required)"
    },
    "stack_trace": {
      "type": "string",
      "description": "Stack trace if available"
    },
    "context": {
      "type": "object",
      "description": "Additional context (environment, versions, etc.)"
    },
    "mode": {
      "type": "string",
      "enum": ["auto", "interactive", "guided"],
      "description": "Troubleshooting mode (default: auto)"
    },
    "category": {
      "type": "string",
      "description": "Error category filter (configuration, resource, etc.)"
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Tags for filtering similar issues"
    },
    "top_k": {
      "type": "number",
      "description": "Number of similar issues to return (default: 5)"
    }
  },
  "required": ["error_message"]
}
```

#### list_patterns

Browse troubleshooting patterns from knowledge base.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "category": {
      "type": "string",
      "description": "Filter by category"
    },
    "severity": {
      "type": "string",
      "enum": ["critical", "high", "medium", "low"],
      "description": "Filter by severity"
    },
    "min_success_rate": {
      "type": "number",
      "description": "Minimum success rate (0-1)"
    },
    "limit": {
      "type": "number",
      "description": "Number of results (default: 10, max: 100)"
    }
  }
}
```

## Data Models and Schemas

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

**Vector Database Schema**:
- **Database**: `shared` (global knowledge accessible to all projects)
- **Collection**: `troubleshooting_knowledge`
- **Vector Dimension**: 1536 (standard embedding dimension)
- **Metadata Fields**: All fields stored as payload for filtering and retrieval

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

### Severity Levels

```go
const (
    SeverityCritical = Severity("critical") // Service crash, data loss, security breach
    SeverityHigh     = Severity("high")     // Major feature broken, workaround exists
    SeverityMedium   = Severity("medium")   // Minor feature broken, inconvenient
    SeverityLow      = Severity("low")      // Cosmetic, edge case, minor annoyance
)
```

### Categories

```go
const (
    CategoryConfiguration = "configuration" // Config errors, missing env vars
    CategoryResource      = "resource"      // Out of memory, disk full
    CategoryDependency    = "dependency"    // Missing library, version mismatch
    CategoryPermission    = "permission"    // Access denied, file permissions
    CategoryLogic         = "logic"         // Nil pointer, index out of bounds
    CategoryNetwork       = "network"       // Connection refused, timeout
    CategoryStorage       = "storage"       // Database errors, file I/O
    CategoryGeneral       = "general"       // Uncategorized
)
```

## Pattern Storage and Retrieval

### Storage Strategy

Troubleshooting patterns are stored in the **shared database** for universal access:

**Multi-Tenant Architecture**:
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

### Indexing

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

### Retrieval Process

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Generate Query Embedding                                 │
│    - Input: error_message                                   │
│    - Output: []float32 (1536 dimensions)                    │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 2. Build Search Query                                       │
│    - Vector: query embedding                                │
│    - TopK: limit * 2 (for reranking)                       │
│    - Filter: category AND tags (if provided)               │
│    - Example: `category == "network" and tags like "%dns%"`│
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 3. Execute Vector Search                                    │
│    - Database: "shared"                                     │
│    - Collection: "troubleshooting_knowledge"                │
│    - Search with IVF index (nprobe=128)                     │
│    - Returns: Top 2N results by vector similarity           │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 4. Calculate Hybrid Scores                                  │
│    - Semantic: 1.0 / (1.0 + distance) * 0.6                │
│    - Success Rate: success_rate * 0.3                       │
│    - Usage: (usage_count / 100) * 0.1                       │
│    - Total: semantic + success_rate + usage                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 5. Rerank and Filter                                        │
│    - Sort by hybrid score (descending)                      │
│    - Take top N results                                     │
│    - Enrich with safety information                         │
│    - Add confidence levels                                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
               [Ranked Results]
```

## Hybrid Matching Algorithm

### Scoring Components

The hybrid matching algorithm combines three signals:

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

### Confidence Determination

```go
func determineConfidence(matchScore float64) string {
    if matchScore >= 0.8 {
        return ConfidenceHigh
    } else if matchScore >= 0.5 {
        return ConfidenceMedium
    }
    return ConfidenceLow
}
```

## Performance Characteristics

### Target Response Times

| Operation | Target | Typical | Notes |
|-----------|--------|---------|-------|
| Diagnose (full) | < 2s | 1.5s | Includes embedding generation, search, hypothesis generation |
| Search Similar Issues | < 300ms | 200ms | Vector search with hybrid scoring |
| Store Resolution | < 500ms | 300ms | Embedding generation + database insert |
| List Patterns | < 100ms | 50ms | Metadata query (no vector search) |

### Scalability Characteristics

**Vector Search Performance**:
- **Small KB (<1000 patterns)**: <50ms search time
- **Medium KB (1000-10000 patterns)**: 50-200ms search time
- **Large KB (>10000 patterns)**: 200-500ms search time

**Factors Affecting Performance**:
1. **Number of Patterns**: Linear growth with IVF index
2. **Vector Dimension**: Fixed at 1536 (standard)
3. **TopK Parameter**: Minimal impact up to 50 results
4. **Filter Complexity**: Metadata filters add 10-30ms
5. **Reranking**: Negligible (<5ms) for typical result sets

### Resource Requirements

**Memory**:
- Service base: ~50MB
- Per 1000 patterns: ~20MB (vectors + metadata)
- Embedding cache: Configurable (future feature)

**CPU**:
- Diagnosis: Single core, ~100ms peak
- Vector search: Distributed across IVF clusters
- Hypothesis generation: Minimal (<10ms)

**Storage**:
- Per pattern: ~5KB (vector + metadata)
- 10,000 patterns: ~50MB
- Includes redundancy for vector index

### Optimization Strategies

1. **Vector Index Tuning**:
   - IVF clusters: Balanced at 128 for typical workloads
   - Increase for very large knowledge bases (>50k patterns)
   - Decrease for very small knowledge bases (<500 patterns)

2. **Batch Operations**:
   - Store multiple resolutions in single transaction
   - Bulk import for initial knowledge base seeding

3. **Caching** (Future):
   - Cache embeddings for frequently diagnosed errors
   - Cache search results with TTL
   - Cache pattern metadata for list operations

4. **Query Optimization**:
   - Use specific filters to reduce search space
   - Limit TopK to actual needed results
   - Avoid overly broad categories

## Error Handling

### Error Categories

1. **Validation Errors**: Invalid input parameters
2. **Database Errors**: Vector store connection/operation failures
3. **Embedding Errors**: Embedding generation failures
4. **Not Found Errors**: Pattern or session not found
5. **Timeout Errors**: Operations exceeding context deadline

### Error Response Format

All errors follow standard format:

```go
type APIError struct {
    Code    string `json:"code"`    // Machine-readable error code
    Message string `json:"message"` // Human-readable message
    Details string `json:"details"` // Technical details
}
```

### Error Codes

| Code | HTTP Status | Meaning | User Action |
|------|-------------|---------|-------------|
| `INVALID_REQUEST` | 400 | Invalid input parameters | Fix request parameters |
| `DIAGNOSIS_FAILED` | 500 | Diagnosis process failed | Retry or report issue |
| `DATABASE_ERROR` | 500 | Vector store operation failed | Check database status |
| `EMBEDDING_ERROR` | 500 | Embedding generation failed | Check embedding service |
| `SESSION_NOT_FOUND` | 404 | Session ID not found | Verify session ID |
| `LIST_PATTERNS_FAILED` | 500 | Pattern listing failed | Retry or report issue |
| `UPDATE_FEEDBACK_FAILED` | 500 | Feedback update failed | Retry update |

### Error Wrapping

All errors are wrapped with context using `fmt.Errorf`:

```go
if err := s.embedder.Embed(ctx, text); err != nil {
    return nil, fmt.Errorf("failed to generate embedding: %w", err)
}
```

### Context Propagation

All operations respect context deadlines:

```go
ctx, span := s.tracer.Start(ctx, "troubleshooting.diagnose")
defer span.End()

select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
    // Continue operation
}
```

## Security Considerations

### Input Validation

1. **Error Message**: Max 10,000 characters, prevent injection
2. **Stack Trace**: Max 50,000 characters, sanitize paths
3. **Context**: Max 100 key-value pairs, sanitize values
4. **Category**: Enum validation (only predefined categories)
5. **Severity**: Enum validation (only critical/high/medium/low)
6. **Tags**: Max 20 tags, max 50 chars each

### Sensitive Data Handling

**Automatic Redaction**:
- API keys (pattern: `sk-...`, `key-...`)
- Tokens (pattern: `Bearer ...`, `token: ...`)
- Passwords (pattern: `password=...`, `pwd=...`)
- File paths with usernames (`/home/username/` → `/home/***/`)

**Sanitization**:
```go
func sanitizeFilterValue(value string) string {
    // Escape special characters for filter expressions
    value = strings.ReplaceAll(value, "\\", "\\\\")
    value = strings.ReplaceAll(value, "\"", "\\\"")
    return value
}
```

### Filter Injection Prevention

**Multi-Tenant Isolation**:
- Troubleshooting patterns stored in isolated shared database
- No cross-project data access possible
- Database-level isolation prevents filter injection

**Query Sanitization**:
- All filter values sanitized before building expressions
- Special characters escaped
- No raw user input in filter strings

### Access Control

**Authentication**:
- Bearer token required for all API endpoints
- Token validation with constant-time comparison
- Token stored with 0600 permissions

**Authorization**:
- All users can access global troubleshooting knowledge
- Session access limited to session creator (future)
- No PII stored in patterns

## Testing Requirements

### Coverage Requirements

| Component | Minimum Coverage | Critical Paths |
|-----------|------------------|----------------|
| Service Core | 80% | 100% |
| Diagnosis Engine | 80% | 100% |
| Pattern Retrieval | 80% | 100% |
| Hybrid Scoring | 100% | 100% |
| Safety Detection | 100% | 100% |
| Handlers | 80% | - |

### Test Categories

#### 1. Unit Tests

**Service Tests**:
- `TestNewService`: Constructor validation
- `TestDiagnose`: Full diagnosis workflow
- `TestSearchSimilarIssues`: Pattern retrieval
- `TestGenerateHypotheses`: Hypothesis generation
- `TestRankHypotheses`: Probability ranking
- `TestGenerateActions`: Action generation
- `TestDetectDestructive`: Safety detection

**Hybrid Scoring Tests**:
- `TestCalculateHybridScore`: Score calculation
- `TestSemanticScoring`: Vector similarity
- `TestSuccessRateWeighting`: Success rate impact
- `TestUsageFrequencyWeighting`: Usage count impact

**Safety Tests**:
- `TestIsDestructive`: Destructive keyword detection
- `TestContainsDestructive`: Solution analysis
- `TestSafetyWarnings`: Warning generation

#### 2. Integration Tests

**Vector Store Integration**:
- Store and retrieve patterns
- Search with filters
- Hybrid ranking validation
- Multi-tenant isolation

**End-to-End Diagnosis**:
- Complete workflow from request to response
- Multiple similar issues handling
- Confidence level determination
- Progressive disclosure validation

#### 3. Handler Tests

**HTTP Handler Tests**:
- Valid request handling
- Invalid request handling
- Error response formatting
- Safety warning inclusion

**MCP Tool Tests**:
- Tool schema validation
- Parameter parsing
- Response formatting
- Error handling

#### 4. Performance Tests

**Benchmark Tests**:
- `BenchmarkDiagnose`: Full diagnosis performance
- `BenchmarkSearch`: Vector search performance
- `BenchmarkHybridScoring`: Scoring calculation performance

**Load Tests**:
- Concurrent diagnosis requests
- Large knowledge base searches
- High-frequency pattern storage

#### 5. Edge Case Tests

**Error Conditions**:
- Empty error message
- Missing required fields
- Invalid category/severity
- Database connection failure
- Embedding service failure

**Boundary Conditions**:
- Very long error messages
- Very long stack traces
- Max context size
- No similar issues found
- Zero usage count patterns

### Test Data

**Fixtures**:
```go
func newTestKnowledge() *TroubleshootingKnowledge {
    return &TroubleshootingKnowledge{
        ID:              "test-123",
        ErrorPattern:    "connection refused",
        Context:         "network",
        RootCause:       "Service not running",
        Solution:        "Start the service",
        DiagnosticSteps: "1. Check service status\n2. Start service",
        SuccessRate:     0.95,
        Severity:        SeverityHigh,
        Category:        CategoryNetwork,
        Tags:            []string{"network", "connection"},
        CreatedAt:       time.Now(),
        UpdatedAt:       time.Now(),
        LastUsed:        time.Now(),
        UsageCount:      42,
    }
}
```

**Mocks**:
```go
type mockEmbedder struct {
    embedFunc func(ctx context.Context, text string) (*embedding.EmbeddingResult, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) (*embedding.EmbeddingResult, error) {
    if m.embedFunc != nil {
        return m.embedFunc(ctx, text)
    }
    return &embedding.EmbeddingResult{
        Embedding: make([]float32, 1536),
    }, nil
}
```

## Usage Examples

### Example 1: Basic Error Diagnosis

```go
package main

import (
    "context"
    "fmt"
    "github.com/axyzlabs/contextd/pkg/troubleshooting"
)

func main() {
    // Create service
    service, err := troubleshooting.NewService(vectorStore, embedder)
    if err != nil {
        panic(err)
    }

    // Diagnose error
    req := &troubleshooting.DiagnosisRequest{
        ErrorMessage: "panic: runtime error: invalid memory address or nil pointer dereference",
        Context: map[string]string{
            "file":     "main.go",
            "line":     "42",
            "language": "go",
        },
        Mode: troubleshooting.ModeAuto,
    }

    session, err := service.Diagnose(context.Background(), req)
    if err != nil {
        panic(err)
    }

    // Print results
    fmt.Printf("Session ID: %s\n", session.ID)
    fmt.Printf("Root Cause: %s\n", session.Diagnosis.RootCause)
    fmt.Printf("Confidence: %s (%.2f)\n",
        session.Diagnosis.Confidence,
        session.Diagnosis.ConfidenceScore)

    // Print similar issues
    fmt.Printf("\nSimilar Issues:\n")
    for i, issue := range session.SimilarIssues {
        fmt.Printf("%d. %s (score: %.2f)\n",
            i+1, issue.Knowledge.ErrorPattern, issue.MatchScore)
        fmt.Printf("   Solution: %s\n", issue.Solution)
        if issue.IsDestructive {
            fmt.Printf("   ⚠️  WARNING: Destructive operation\n")
        }
    }

    // Print recommended actions
    fmt.Printf("\nRecommended Actions:\n")
    for _, action := range session.RecommendedSteps {
        fmt.Printf("%d. %s\n", action.Step, action.Description)
        if action.Destructive {
            fmt.Printf("   ⚠️  %s\n", action.Safety)
        }
    }
}
```

### Example 2: Store Resolution After Fixing

```go
// After resolving an error, store the solution
req := &troubleshooting.StoreKnowledgeRequest{
    ErrorPattern:    "connection refused localhost:5432",
    Context:         "postgresql",
    RootCause:       "PostgreSQL service not running",
    Solution:        "sudo systemctl start postgresql",
    DiagnosticSteps: "1. Check service status: systemctl status postgresql\n2. Check logs: journalctl -u postgresql",
    Severity:        "high",
    Category:        troubleshooting.CategoryNetwork,
    Tags:            []string{"postgresql", "database", "connection"},
}

knowledge, err := service.StoreResolution(context.Background(), req)
if err != nil {
    panic(err)
}

fmt.Printf("Stored knowledge: %s\n", knowledge.ID)
```

### Example 3: Browse Patterns

```go
// List high-severity network issues with good success rate
patterns, err := service.ListPatterns(
    context.Background(),
    troubleshooting.CategoryNetwork,
    troubleshooting.SeverityHigh,
    0.8, // min success rate
)
if err != nil {
    panic(err)
}

fmt.Printf("Found %d patterns:\n", len(patterns))
for _, p := range patterns {
    fmt.Printf("- %s (success: %.1f%%, used: %d times)\n",
        p.ErrorPattern, p.SuccessRate*100, p.UsageCount)
}
```

### Example 4: MCP Tool Usage

From Claude Code:

```bash
# Diagnose an error
/troubleshoot "Failed to connect to database: SQLSTATE[HY000] [2002]"

# Browse patterns
/list_patterns category=database severity=high

# Filter by success rate
/list_patterns min_success_rate=0.9
```

### Example 5: HTTP API Usage

```bash
# Diagnose via API
TOKEN=$(cat ~/.config/contextd/token)

curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://localhost/api/v1/troubleshoot \
  -d '{
    "error_message": "panic: runtime error: invalid memory address",
    "context": {"file": "main.go", "line": "42"},
    "mode": "auto"
  }'

# List patterns
curl --unix-socket ~/.config/contextd/api.sock \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost/api/v1/troubleshoot/patterns?category=network&severity=high"
```

## Integration Points

### Internal Dependencies

1. **pkg/vectorstore**: Universal vector store interface
2. **pkg/embedding**: Embedding generation service
3. **pkg/telemetry**: OpenTelemetry instrumentation
4. **pkg/validation**: Request validation

### External Dependencies

2. **Embedding Service**: OpenAI API or TEI (local)
3. **Monitoring**: OpenTelemetry collector (optional)

### Service Integration

**MCP Server**:
- Registers `troubleshoot` and `list_patterns` tools
- Handles JSON-RPC requests from Claude Code
- Translates between MCP protocol and service API

**HTTP Handlers**:
- Exposes REST API for troubleshooting operations
- Handles authentication via Bearer token
- Returns standardized JSON responses

**Observability**:
- OpenTelemetry traces for all operations
- Metrics for diagnosis performance
- Pattern match tracking
- Success rate monitoring

## Future Enhancements

### Planned Features

1. **Session Persistence**: Store and retrieve complete diagnostic sessions
2. **Feedback Loop**: Automatic success rate updates based on feedback
3. **Pattern Evolution**: Merge similar patterns, archive outdated ones
4. **Interactive Mode**: Step-by-step guided troubleshooting with user input
5. **Guided Mode**: Wizard-style troubleshooting workflow
6. **Pattern Templates**: Predefined templates for common error types
7. **Multi-Language Support**: Error message translation for international users
8. **AI Enhancement**: GPT-4 integration for novel error analysis

### Research Areas

1. **Embedding Optimization**: Fine-tune embeddings for error messages
2. **Causal Analysis**: Build causal graphs for complex error chains
3. **Automated Testing**: Generate tests based on error patterns
4. **Predictive Diagnosis**: Predict errors before they occur
5. **Cross-Project Learning**: Share patterns across organizations (privacy-preserving)

## Related Documentation

- **Architecture**: `docs/standards/architecture.md`
- **Testing Standards**: `docs/standards/testing-standards.md`
- **Coding Standards**: `docs/standards/coding-standards.md`
- **Vector Store**: `docs/specs/vectorstore/SPEC.md`
- **Remediation**: `docs/specs/remediation/SPEC.md`
- **User Guide**: `docs/contextd/troubleshooting.md`

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-04 | Initial specification |

---

**Maintained by**: contextd team
**Last Review**: 2025-11-04
**Next Review**: 2026-02-04
