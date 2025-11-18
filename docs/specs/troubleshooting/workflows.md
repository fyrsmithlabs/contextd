# Troubleshooting Workflows

**Parent**: [../SPEC.md](../SPEC.md)

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

## HTTP API

### POST /api/v1/troubleshoot

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

### GET /api/v1/troubleshoot/patterns

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

## MCP Tools

### troubleshoot

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

### list_patterns

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
curl -X POST http://localhost:8080/api/v1/troubleshoot \
  -H "Content-Type: application/json" \
  -d '{
    "error_message": "panic: runtime error: invalid memory address",
    "context": {"file": "main.go", "line": "42"},
    "mode": "auto"
  }'

# List patterns
curl "http://localhost:8080/api/v1/troubleshoot/patterns?category=network&severity=high"
```

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
