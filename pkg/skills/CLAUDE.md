# Package: skills

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides skills management system for storing, searching, and applying reusable workflow templates. Skills represent tested solutions to common problems that can be discovered and applied by AI agents.

## Specification

**Full Spec**: [`docs/specs/skills/SPEC.md`](../../docs/specs/skills/SPEC.md)

**Quick Summary**:
- **Problem**: Repetitive problem-solving wastes time; need reusable templates
- **Solution**: Skills database with semantic search and usage tracking
- **Key Features**:
  - Semantic search for skill discovery
  - Usage tracking (count, success rate)
  - Multi-field skills (problem statement, success criteria, prerequisites)
  - Global knowledge accessible to all projects

## Architecture

**Design Pattern**: Service layer with vector store abstraction

**Dependencies**:
- `pkg/vectorstore` - Vector database operations
- `pkg/validation` - Request validation
- `go.opentelemetry.io/otel` - OpenTelemetry metrics

**Used By**:
- `internal/handlers` - HTTP API handlers
- `pkg/mcp` - MCP skill tools

## Key Components

### Main Types

```go
// Service orchestrates skill operations
type Service struct {
    vectorStore VectorStore
    embedder    EmbeddingGenerator
    tracer      trace.Tracer
    meter       metric.Meter
}

// Skill represents a reusable workflow template
type Skill struct {
    ID                       string
    Name                     string
    Description              string
    Content                  string
    ProblemStatement         string
    SuccessCriteria          []string
    Prerequisites            []string
    ExpectedOutcome          string
    Category                 string
    Tags                     []string
    UsageCount               int
    SuccessRate              float64
    CreatedAt                time.Time
    UpdatedAt                time.Time
}
```

### Main Functions

```go
// NewService creates a new skills service
func NewService(vectorStore VectorStore, embedder EmbeddingGenerator) (*Service, error)

// Create creates a new skill with automatic embedding
func (s *Service) Create(ctx context.Context, req *validation.CreateSkillRequest) (*Skill, error)

// Search performs semantic search for skills
func (s *Service) Search(ctx context.Context, req *validation.SearchSkillsRequest) (*SearchResult, error)

// List retrieves paginated list of skills
func (s *Service) List(ctx context.Context, req *validation.ListSkillsRequest) (*ListResult, error)

// GetByID retrieves a single skill
func (s *Service) GetByID(ctx context.Context, id string) (*Skill, error)

// Update updates an existing skill
func (s *Service) Update(ctx context.Context, id string, fields *UpdateFields) (*Skill, error)

// Delete deletes a skill
func (s *Service) Delete(ctx context.Context, id string) error

// RecordUsage records skill application and updates success rate
func (s *Service) RecordUsage(ctx context.Context, id string, success bool) error
```

## Usage Example

```go
// Create service
svc, err := skills.NewService(vectorStore, embedder)

// Create a new skill
req := &validation.CreateSkillRequest{
    Name:        "Docker Container Restart",
    Description: "Restart a Docker container safely",
    Content:     "docker restart <container-name>",
    ProblemStatement: "Container needs restart without losing data",
    SuccessCriteria: []string{
        "Container restarts successfully",
        "No data loss occurs",
    },
    Category: "devops",
    Tags:     []string{"docker", "containers"},
}
skill, err := svc.Create(ctx, req)

// Search for skills
searchReq := &validation.SearchSkillsRequest{
    Query: "restart docker container",
    TopK:  10,
}
results, err := svc.Search(ctx, searchReq)

// Record usage
err = svc.RecordUsage(ctx, skill.ID, true)  // success = true
```

## Testing

**Test Coverage**: 78% (Target: ≥80%)

**Running Tests**:
```bash
go test ./pkg/skills/
```

## Configuration

**No environment variables** - uses shared vector database

**Database**: Skills stored in `"shared"` database (global knowledge accessible to all projects)

## Security Considerations

1. **Filter Injection Prevention**: All filter values sanitized
2. **Global Knowledge**: Skills accessible to all projects (by design)
3. **Concurrent Updates**: Mutex protection for update operations
4. **Input Validation**: All requests validated before processing

## Performance Notes

- **Create operation**: 100-300ms (embedding generation)
- **Search operation**: 20-100ms (vector search)
- **List operation**: 50-200ms (paginated)
- **Usage update**: 50-150ms (delete + reinsert pattern)

**Optimization Tips**:
- Use caching for frequently accessed skills
- Batch create operations when possible
- Set appropriate `topK` limits for searches

## Related Documentation

- Spec: [`docs/specs/skills/SPEC.md`](../../docs/specs/skills/SPEC.md)
- API Reference: [`internal/handlers/skills.go`](../../internal/handlers/)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
