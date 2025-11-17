# Package: remediation

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Stores and retrieves error solutions using hybrid matching (semantic + string similarity). Enables learning from past errors and sharing solutions across the team.

## Specification

**Full Spec**: [`docs/specs/remediation/SPEC.md`](../../docs/specs/remediation/SPEC.md)

**Quick Summary**:
- **Problem**: Developers encounter same errors repeatedly, wasting time searching for solutions
- **Solution**: Store error-solution pairs with hybrid search (70% semantic, 30% string matching)
- **Key Features**:
  - Hybrid matching algorithm for high precision
  - Pattern extraction for error normalization
  - Severity classification (critical, high, medium, low)
  - Success rate tracking

## Architecture

**Design Pattern**: Service pattern with custom hybrid matcher

**Dependencies**:
- `pkg/embedding` - Generate vector embeddings for semantic matching
- `pkg/vectorstore` - Store remediation vectors
- `pkg/security` - Redact sensitive data from error messages

**Used By**:
- `pkg/mcp` - MCP server exposes remediation tools
- `pkg/troubleshooting` - AI diagnosis uses remediation history
- `cmd/contextd` - API server endpoints

## Key Components

### Main Types

```go
// Remediation represents an error and its solution
type Remediation struct {
    ID           string                 `json:"id"`
    ErrorMessage string                 `json:"error_message"` // Required
    ErrorType    string                 `json:"error_type"`    // Required
    Solution     string                 `json:"solution"`      // Required
    StackTrace   string                 `json:"stack_trace"`   // Optional
    Context      map[string]interface{} `json:"context"`       // Additional info
    Tags         []string               `json:"tags"`          // Categorization
    Severity     string                 `json:"severity"`      // low, medium, high, critical
    ProjectPath  string                 `json:"project_path"`  // Optional filter
    SuccessCount int                    `json:"success_count"` // Times solution worked
    CreatedAt    time.Time             `json:"created_at"`
}

// Service provides remediation operations
type Service struct {
    store     vectorstore.VectorStore
    embedding *embedding.Service
    matcher   *HybridMatcher
}

// HybridMatcher combines semantic and string similarity
type HybridMatcher struct {
    semanticWeight float64 // Default: 0.7
    stringWeight   float64 // Default: 0.3
}
```

### Main Functions

```go
// Save stores a remediation with pattern extraction
func (s *Service) Save(ctx context.Context, rem *Remediation) error

// Search finds similar errors using hybrid matching
func (s *Service) Search(ctx context.Context, errorMsg string, opts *SearchOptions) ([]*Remediation, error)

// IncrementSuccess tracks successful solutions
func (s *Service) IncrementSuccess(ctx context.Context, id string) error

// ExtractPatterns normalizes error messages for better matching
func ExtractPatterns(errorMsg string) []string
```

## Usage Example

```go
// Create service
svc := remediation.NewService(vectorStore, embeddingSvc)

// Save remediation
rem := &remediation.Remediation{
    ErrorMessage: "dial tcp 127.0.0.1:6333: connect: connection refused",
    ErrorType:    "ConnectionError",
    Solution:     "Start Qdrant: docker-compose up -d qdrant",
    Context: map[string]interface{}{
        "service": "qdrant",
        "port":    6333,
    },
    Tags:     []string{"qdrant", "docker", "connection"},
    Severity: "high",
}
if err := svc.Save(ctx, rem); err != nil {
    return err
}

// Search for similar errors
results, err := svc.Search(ctx, "connection refused port 6333", &remediation.SearchOptions{
    Limit:    5,
    MinScore: 0.6,
})
if err != nil {
    return err
}

for _, rem := range results {
    fmt.Printf("Similar error (%.2f): %s\n", rem.Score, rem.ErrorMessage)
    fmt.Printf("Solution: %s\n", rem.Solution)
    fmt.Printf("Success rate: %d times\n\n", rem.SuccessCount)
}

// Mark solution as successful
if err := svc.IncrementSuccess(ctx, results[0].ID); err != nil {
    return err
}
```

## Testing

**Test Coverage**: 82% (Target: ≥80%)

**Key Test Files**:
- `remediation_test.go` - Unit tests for service methods
- `matcher_test.go` - Hybrid matching algorithm tests
- `patterns_test.go` - Pattern extraction tests

**Running Tests**:
```bash
go test ./pkg/remediation/
go test -cover ./pkg/remediation/
go test -race ./pkg/remediation/
```

## Configuration

**Environment Variables**:
- Inherits embedding config from `pkg/embedding`

**Hybrid Matcher Tuning**:
```go
// Default weights (70% semantic, 30% string)
matcher := remediation.NewHybridMatcher(0.7, 0.3)

// Adjust for more exact matching
matcher := remediation.NewHybridMatcher(0.5, 0.5)

// Adjust for more semantic matching
matcher := remediation.NewHybridMatcher(0.9, 0.1)
```

## Security Considerations

- **Redaction**: Automatically redact API keys, tokens from error messages using `pkg/security`
- **Shared knowledge**: Remediations stored in shared database (not project-specific)
- **Sanitization**: Clean error messages before storage to prevent injection

## Performance Notes

- **Pattern extraction**: Pre-computed patterns for fast string matching
- **Hybrid search**: Combined score calculated in single pass
- **Cache candidates**: Top 100 semantic matches cached for string comparison
- **Batch operations**: Use `SaveBatch` for importing multiple remediations

**Hybrid Matching Algorithm**:
1. Vector search retrieves top 100 candidates (semantic)
2. Levenshtein distance computed for each candidate (string)
3. Scores combined: `0.7 * semantic + 0.3 * string`
4. Results sorted by combined score, filtered by threshold (0.6)

## Related Documentation

- Spec: [`docs/specs/remediation/SPEC.md`](../../docs/specs/remediation/SPEC.md)
- Research: [`docs/specs/remediation/research/`](../../docs/specs/remediation/research/)
- Troubleshooting Integration: [`docs/specs/troubleshooting/SPEC.md`](../../docs/specs/troubleshooting/SPEC.md)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
