# Remediation Workflows

**Parent**: [../SPEC.md](../SPEC.md)

This document describes usage patterns and example workflows for the remediation system.

---

## MCP Tools

### remediation_save

Store an error solution for future reference.

**Description**:
Saves error message, type, solution, stack trace, and metadata with vector embeddings for intelligent matching.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "error_message": {
      "type": "string",
      "description": "Error message or exception text"
    },
    "error_type": {
      "type": "string",
      "description": "Error type or exception class"
    },
    "solution": {
      "type": "string",
      "description": "Solution or fix for the error"
    },
    "project_path": {
      "type": "string",
      "description": "Project path where error occurred (optional)"
    },
    "context": {
      "type": "object",
      "description": "Additional context about the error"
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Tags for categorization"
    },
    "severity": {
      "type": "string",
      "description": "Severity level (low, medium, high, critical)"
    },
    "stack_trace": {
      "type": "string",
      "description": "Stack trace if available"
    }
  },
  "required": ["error_message", "error_type", "solution"]
}
```

**Output**:
```json
{
  "id": "uuid",
  "error_message": "ImportError: No module named 'requests'",
  "error_type": "ImportError",
  "solution": "Install requests: pip install requests",
  "created_at": "2025-11-04T10:30:00Z"
}
```

### remediation_search

Find similar error solutions using hybrid matching.

**Description**:
Returns ranked results with match scores (70% semantic + 30% string similarity), similar errors, and their solutions.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "error_message": {
      "type": "string",
      "description": "Error message to search for similar errors"
    },
    "stack_trace": {
      "type": "string",
      "description": "Stack trace for better matching"
    },
    "limit": {
      "type": "integer",
      "description": "Number of results (default: 5, max: 100)"
    },
    "min_score": {
      "type": "number",
      "description": "Minimum match score (0-1, default: 0.5)"
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Filter by tags"
    }
  },
  "required": ["error_message"]
}
```

**Output**:
```json
{
  "results": [
    {
      "remediation": {
        "id": "uuid",
        "error_message": "ImportError: No module named 'requests'",
        "error_type": "ImportError",
        "solution": "Install requests: pip install requests",
        "tags": ["python", "import"],
        "timestamp": 1699012800
      },
      "match_score": 0.87,
      "match_details": {
        "semantic_score": 0.92,
        "string_score": 0.78,
        "hybrid_score": 0.87,
        "error_type_match": true,
        "stack_trace_match": false
      }
    }
  ],
  "count": 1
}
```

---

## Usage Examples

### Example 1: Save Python Import Error

```go
req := &remediation.CreateRemediationRequest{
    ErrorMessage: "ImportError: No module named 'requests'",
    ErrorType:    "ImportError",
    Solution:     "Install the requests module: pip install requests",
    Tags:         []string{"python", "import", "dependencies"},
    Severity:     "medium",
    Context: map[string]string{
        "language": "python",
        "version":  "3.11",
    },
}

rem, err := service.Create(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created remediation: %s\n", rem.ID)
```

### Example 2: Search for Similar Error

```go
searchReq := &remediation.SearchRequest{
    ErrorMessage: "ModuleNotFoundError: No module named 'django'",
    Limit:        5,
    MinScore:     0.6,
    Tags:         []string{"python"},
}

results, err := service.FindSimilarErrors(ctx, searchReq)
if err != nil {
    log.Fatal(err)
}

for _, match := range results {
    fmt.Printf("Match Score: %.2f\n", match.MatchScore)
    fmt.Printf("Error: %s\n", match.Remediation.ErrorMessage)
    fmt.Printf("Solution: %s\n", match.Remediation.Solution)
    fmt.Printf("Semantic: %.2f, String: %.2f\n",
        match.MatchDetails.SemanticScore,
        match.MatchDetails.StringScore)
    fmt.Println("---")
}
```

### Example 3: Custom Matcher Weights

```go
// Prioritize semantic similarity (90% semantic, 10% string)
matcher := remediation.NewMatcherWithWeights(
    0.9,  // semantic weight
    0.1,  // string weight
    0.6,  // min semantic score
    0.2,  // min string score
    0.7,  // min hybrid score
)

// Use custom matcher in service
service := remediation.NewServiceWithMatcher(vectorStore, embedder, matcher)
```

### Example 4: With Stack Trace

```go
req := &remediation.CreateRemediationRequest{
    ErrorMessage: "NullPointerException: Cannot invoke method on null object",
    ErrorType:    "NullPointerException",
    Solution:     "Add null check before method invocation",
    StackTrace: `at com.example.Service.process(Service.java:42)
at com.example.Controller.handle(Controller.java:123)
at com.example.Main.main(Main.java:15)`,
    Tags:     []string{"java", "npe"},
    Severity: "high",
}

rem, err := service.Create(ctx, req)

// Search with stack trace for better matching
searchReq := &remediation.SearchRequest{
    ErrorMessage: "NullPointerException at Service.process",
    StackTrace: `at com.example.Service.process(Service.java:50)
at com.example.Controller.handle(Controller.java:130)`,
    Limit: 3,
}

// Stack trace match will boost score by 15%
results, err := service.FindSimilarErrors(ctx, searchReq)
```

---

## Error Handling

### Input Validation Errors

**Error Code**: `VALIDATION_ERROR`

**Scenarios**:
- Missing required fields
- Invalid severity level
- Tags exceed limits
- Invalid project path

**Response**:
```json
{
  "error": "VALIDATION_ERROR",
  "message": "invalid error_message",
  "details": {
    "field": "error_message",
    "error": "error_message is required"
  }
}
```

### Service Errors

**Error Code**: `INTERNAL_ERROR`

**Scenarios**:
- Embedding generation failed
- Vector store unavailable
- Database write failed

**Response**:
```json
{
  "error": "INTERNAL_ERROR",
  "message": "failed to create remediation",
  "details": {
    "cause": "embedding service timeout"
  }
}
```

### Timeout Errors

**Error Code**: `TIMEOUT_ERROR`

**Default Timeouts**:
- Create: 30 seconds
- Search: 60 seconds

**Response**:
```json
{
  "error": "TIMEOUT_ERROR",
  "message": "remediation search timed out",
  "details": {
    "timeout": "60s"
  }
}
```

### Retry Strategy

**Transient Errors (Retry)**:
- Network timeouts
- Rate limit errors
- Temporary vector store unavailability

**Permanent Errors (No Retry)**:
- Validation errors
- Invalid authentication
- Corrupted data

**Retry Configuration**:
- Max attempts: 3
- Backoff: Exponential (1s, 2s, 4s)
- Jitter: ±20%
