# Package: troubleshoot

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides AI-powered error diagnosis and pattern recognition. Analyzes error messages using semantic search against known patterns and AI hypothesis generation to provide root cause analysis and remediation suggestions.

## Specification

**Full Spec**: [`docs/specs/mcp/IMPLEMENTATION-PLAN.md`](../../docs/specs/mcp/IMPLEMENTATION-PLAN.md) (Phase 1.3)

**Quick Summary**:
- **Problem**: Errors are cryptic; need AI-powered diagnosis with context-aware suggestions
- **Solution**: Hybrid pattern matching + AI hypothesis generation with confidence scoring
- **Key Features**:
  - Semantic search for similar known patterns
  - AI-powered hypothesis generation (OpenAI)
  - Pattern storage for team knowledge sharing
  - Integration with remediation service

## Architecture

**Design Pattern**: Service layer with hybrid diagnosis (patterns + AI)

**Dependencies**:
- `pkg/vectorstore` - Pattern storage and semantic search
- `go.opentelemetry.io/otel` - OpenTelemetry tracing
- `go.uber.org/zap` - Structured logging
- External: OpenAI API (optional, for AI diagnosis)

**Used By**:
- `pkg/mcp` - MCP troubleshoot and list_patterns tools
- Future: `pkg/handlers` - REST API endpoints

## Key Components

### Main Types

```go
// Service provides error diagnosis operations
type Service struct {
    store    VectorStore
    logger   *zap.Logger
    aiClient AIClient  // Optional, nil = pattern-only mode
    tracer   trace.Tracer
}

// Diagnosis represents AI-powered error analysis
type Diagnosis struct {
    ErrorMessage    string
    RootCause       string
    Hypotheses      []Hypothesis
    Recommendations []string
    RelatedPatterns []Pattern
    Confidence      float64
}

// Pattern represents a known error pattern
type Pattern struct {
    ID          string
    ErrorType   string
    Description string
    Solution    string
    Frequency   int
    Confidence  float64
    CreatedAt   time.Time
}
```

### Main Functions

```go
// NewService creates a new troubleshoot service
func NewService(store VectorStore, logger *zap.Logger, aiClient AIClient) *Service

// Diagnose analyzes an error and provides diagnosis
func (s *Service) Diagnose(ctx context.Context, errorMsg, errorContext string) (*Diagnosis, error)

// SavePattern stores a known error pattern
func (s *Service) SavePattern(ctx context.Context, pattern *Pattern) error

// GetPatterns retrieves all known patterns
func (s *Service) GetPatterns(ctx context.Context) ([]Pattern, error)
```

## Usage Example

```go
// Create service (with AI)
svc := troubleshoot.NewService(vectorStore, logger, aiClient)

// Diagnose an error
diagnosis, err := svc.Diagnose(ctx,
    "connection refused: dial tcp 127.0.0.1:6333: connect: connection refused",
    "Qdrant startup during contextd initialization")

if err != nil {
    return err
}

// Check diagnosis
fmt.Printf("Root Cause: %s\n", diagnosis.RootCause)
fmt.Printf("Confidence: %.2f\n", diagnosis.Confidence)
for _, rec := range diagnosis.Recommendations {
    fmt.Printf("- %s\n", rec)
}

// Save a pattern for future reference
pattern := &troubleshoot.Pattern{
    ErrorType:   "ConnectionError",
    Description: "Qdrant connection refused on port 6333",
    Solution:    "Start Qdrant: docker-compose up -d qdrant",
    Confidence:  0.95,
}
err = svc.SavePattern(ctx, pattern)
```

## Diagnosis Workflow

1. **Pattern Search**: Query vector store for semantically similar patterns
2. **High-Confidence Match**: If pattern score >0.8, return pattern-based diagnosis
3. **AI Hypothesis**: If no high-confidence match, query AI for analysis
4. **Combine Results**: Merge pattern matches with AI hypotheses
5. **Generate Recommendations**: Combine pattern solutions + AI suggestions
6. **Calculate Confidence**: Average of pattern scores + hypothesis likelihoods

## Testing

**Test Coverage**: Expected e80%

**Key Test Files**:
- `service_test.go` - Service methods, pattern matching, AI integration
- `types_test.go` (create) - Validation tests for Diagnosis and Pattern

**Running Tests**:
```bash
go test ./pkg/troubleshoot/
go test -cover ./pkg/troubleshoot/
go test -race ./pkg/troubleshoot/
```

## Configuration

**Environment Variables**:
- `OPENAI_API_KEY` - OpenAI API key (optional, for AI diagnosis)
- No AI client = pattern-only mode (graceful degradation)

**Database**: Patterns stored in `"shared"` database (team-scoped knowledge)

**Collection**: `"troubleshoot_patterns"` collection

## Security Considerations

1. **Multi-Tenant Isolation**: Patterns can be team-scoped via TeamID filter
2. **Input Validation**: Error messages sanitized before AI query
3. **AI API Keys**: Loaded from environment, never logged
4. **Pattern Access**: Team-scoped patterns filtered by TeamID
5. **No PII in Logs**: Error messages redacted in logs if sensitive

## Performance Notes

- **Pattern Search**: 20-100ms (semantic search)
- **AI Diagnosis**: 1-3s (OpenAI API call, network-dependent)
- **High-Confidence Match**: 20-100ms (no AI call, pattern only)
- **Graceful Degradation**: Falls back to patterns if AI fails

**Optimization Tips**:
- Use pattern-only mode for latency-sensitive operations
- Cache frequent error patterns
- Set appropriate confidence thresholds (>0.8 for auto-suggest)

## AI Integration

**Provider**: OpenAI (abstract interface allows future providers)

**Prompt Structure**:
- Error message + context
- Top 3 similar patterns (if any)
- Request: root_cause, hypotheses[], recommendations[]

**Response Format**: JSON with structured diagnosis

**Fallback**: If AI fails, return pattern-based diagnosis or error

## Related Documentation

- Implementation Plan: [`docs/specs/mcp/IMPLEMENTATION-PLAN.md`](../../docs/specs/mcp/IMPLEMENTATION-PLAN.md)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Remediation Package: [`pkg/remediation/CLAUDE.md`](../remediation/CLAUDE.md)
