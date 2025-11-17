# Skills Management Package

The `pkg/skills` package provides a comprehensive skills management system for contextd, enabling users to create, search, and apply reusable workflow templates with semantic search capabilities.

## Overview

Skills are reusable workflow templates that can be semantically searched and applied to similar situations. Each skill includes:

- **Name and Description**: Clear identification and purpose
- **Content**: The actual workflow steps (markdown format)
- **Version**: Semantic versioning for tracking changes
- **Prerequisites**: Required tools, skills, or conditions
- **Expected Outcome**: What the skill should accomplish
- **Category**: Classification (debugging, deployment, analysis, etc.)
- **Tags**: Additional categorization
- **Usage Tracking**: Usage count and success rate metrics

## Architecture

```
┌─────────────────┐
│  MCP Tools      │  ← skill_create, skill_search, skill_list, skill_apply, etc.
└────────┬────────┘
         │
┌────────▼────────┐
│  Service Layer  │  ← Business logic, validation, embedding generation
└────────┬────────┘
         │
┌────────▼────────┐
└─────────────────┘
```

## Key Components

### Models (`models.go`)

- **Skill**: Core domain model with all skill fields
- **SkillSearchResult**: Search result with similarity score
- **SearchResult**: Collection of search results
- **ListResult**: Paginated list results
- **UpdateFields**: Partial update structure
- **SkillUsageStats**: Usage tracking statistics

### Service (`service.go`)

The `Service` struct orchestrates all skill operations:

```go
type Service struct {
    embeddingClient *embedding.Service
    tracer          trace.Tracer
    meter           metric.Meter
    // Metrics...
}
```

**Core Operations**:

- `Create()` - Create new skill with embedding generation
- `Search()` - Semantic search with filters
- `List()` - Paginated listing with filters
- `Update()` - Update existing skill (with re-embedding if content changes)
- `Delete()` - Delete skill by ID
- `GetByID()` - Retrieve single skill
- `RecordUsage()` - Track usage and update success rate

## Usage Examples

### Creating a Skill

```go
import (
    "context"
    "github.com/axyzlabs/contextd/pkg/skills"
    "github.com/axyzlabs/contextd/pkg/validation"
)

func createDeploymentSkill(svc *skills.Service) error {
    req := &validation.CreateSkillRequest{
        Name:        "Docker Deployment Workflow",
        Description: "Complete workflow for deploying applications using Docker",
        Content: `# Docker Deployment Steps

1. Build the Docker image:
   \`\`\`bash
   docker build -t myapp:latest .
   \`\`\`

2. Tag for registry:
   \`\`\`bash
   docker tag myapp:latest registry.example.com/myapp:latest
   \`\`\`

3. Push to registry:
   \`\`\`bash
   docker push registry.example.com/myapp:latest
   \`\`\`

4. Deploy to production:
   \`\`\`bash
   kubectl set image deployment/myapp myapp=registry.example.com/myapp:latest
   \`\`\`
`,
        Version:         "1.0.0",
        Author:          "Platform Team",
        Category:        "deployment",
        Prerequisites:   []string{"docker", "kubectl", "registry access"},
        ExpectedOutcome: "Application successfully deployed to production",
        Tags:            []string{"docker", "kubernetes", "deployment"},
        Metadata: map[string]string{
            "team":        "platform",
            "environment": "production",
        },
    }

    skill, err := svc.Create(context.Background(), req)
    if err != nil {
        return err
    }

    fmt.Printf("Created skill: %s (ID: %s)\n", skill.Name, skill.ID)
    return nil
}
```

### Searching Skills

```go
func searchDeploymentSkills(svc *skills.Service) error {
    req := &validation.SearchSkillsRequest{
        Query:    "how to deploy with kubernetes",
        TopK:     5,
        Category: "deployment",
        Tags:     []string{"kubernetes"},
    }

    results, err := svc.Search(context.Background(), req)
    if err != nil {
        return err
    }

    for _, result := range results.Results {
        fmt.Printf("Match: %s (Score: %.2f)\n", result.Skill.Name, result.Score)
        fmt.Printf("  Prerequisites: %v\n", result.Skill.Prerequisites)
        fmt.Printf("  Success Rate: %.1f%%\n", result.Skill.SuccessRate*100)
    }

    return nil
}
```

### Applying a Skill

```go
func applySkill(svc *skills.Service, skillID string, success bool) error {
    // Get skill by ID
    skill, err := svc.GetByID(context.Background(), skillID)
    if err != nil {
        return err
    }

    // Display skill content
    fmt.Printf("Applying: %s\n", skill.Name)
    fmt.Printf("Prerequisites: %v\n", skill.Prerequisites)
    fmt.Println(skill.Content)

    // Record usage
    err = svc.RecordUsage(context.Background(), skillID, success)
    if err != nil {
        return err
    }

    return nil
}
```

### Listing Skills

```go
func listSkillsByUsage(svc *skills.Service) error {
    req := &validation.ListSkillsRequest{
        Limit:  10,
        Offset: 0,
        SortBy: "usage_count", // Sort by most used
    }

    results, err := svc.List(context.Background(), req)
    if err != nil {
        return err
    }

    fmt.Printf("Found %d skills (showing %d)\n", results.Total, len(results.Skills))

    for _, skill := range results.Skills {
        fmt.Printf("- %s (Used: %d times, Success: %.1f%%)\n",
            skill.Name, skill.UsageCount, skill.SuccessRate*100)
    }

    return nil
}
```

### Updating a Skill

```go
func updateSkillVersion(svc *skills.Service, skillID string) error {
    newVersion := "2.0.0"
    newContent := "# Updated Content..."

    fields := &skills.UpdateFields{
        Version: &newVersion,
        Content: &newContent,
    }

    skill, err := svc.Update(context.Background(), skillID, fields)
    if err != nil {
        return err
    }

    fmt.Printf("Updated skill to version %s\n", skill.Version)
    return nil
}
```

## OpenTelemetry Instrumentation

The service automatically instruments all operations with:

- **Traces**: Every operation creates a span with relevant attributes
- **Metrics**:
  - `skills.create.total` - Counter for skill creations
  - `skills.search.total` - Counter for searches
  - `skills.update.total` - Counter for updates
  - `skills.delete.total` - Counter for deletions
  - `skills.apply.total` - Counter for skill applications
  - `skills.operation.duration` - Histogram of operation durations
  - `skills.embedding.duration` - Histogram of embedding generation times

## Performance Considerations

### Embedding Generation

- Embeddings are generated automatically on create/update
- Only regenerated when name, description, or content changes
- Uses the configured embedding service (OpenAI or TEI)
- Cached by embedding service to reduce API calls

### Search Performance

- Filter expressions optimize result sets before similarity calculation
- TopK parameter controls result set size (max: 100)
- Default nlist parameter: 128 for good speed/accuracy balance

### Update Operations

- Embeddings regenerated only if content changes
- Atomic operation ensures consistency

### Usage Tracking

- RecordUsage updates statistics via delete + reinsert
- Success rate calculated with float64 precision

## Security

### Input Sanitization

All filter values are sanitized to prevent injection attacks:

```go
func sanitizeFilterValue(value string) string {
    value = strings.ReplaceAll(value, "\\", "\\\\")
    value = strings.ReplaceAll(value, "\"", "\\\"")
    return value
}
```

### Validation

All requests validated using `pkg/validation` before processing:

- Name: 1-200 characters
- Description: 1-2000 characters
- Content: 1-50,000 characters
- Version: 1-50 characters
- Category: 1-100 characters
- Tags: alphanumeric, 1-50 characters each

## Error Handling

The service returns wrapped errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to create skill: %w", err)
}
```

Common error scenarios:

- **Embedding client nil**: Service initialization fails
- **Embedding generation failed**: Returns wrapped error
- **Skill not found**: Returns "not found" error on Get/Update/Delete

## Testing

The package includes comprehensive tests:

- **Unit tests** (`service_test.go`): Model validation, data structures
- **Edge case tests** (`edge_cases_test.go`): Boundary conditions, special characters, concurrency

Run tests:

```bash
# Unit tests only
go test ./pkg/skills/

# With coverage
go test -cover ./pkg/skills/

# With race detection
go test -race ./pkg/skills/

INTEGRATION=true go test ./pkg/skills/
```

## Dependencies

- `github.com/axyzlabs/contextd/pkg/embedding` - Embedding generation
- `github.com/axyzlabs/contextd/pkg/validation` - Request validation
- `github.com/google/uuid` - ID generation
- `go.opentelemetry.io/otel` - Observability

## Related Documentation

- **User Guide**: See [../../docs/SKILLS.md](../../docs/SKILLS.md)
- **MCP Integration**: See [../mcp/skills_tools.go](../mcp/skills_tools.go)
- **Validation Models**: See [../validation/models.go](../validation/models.go)

## Future Enhancements

- [ ] Skill versioning history
- [ ] Skill dependencies graph
- [ ] Skill templates with variables
- [ ] Export/import skills (JSON/YAML)
- [ ] Skill collections/bundles
- [ ] Skill recommendations based on context
- [ ] Multi-language skill support
- [ ] Skill execution automation
