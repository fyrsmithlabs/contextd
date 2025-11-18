# Package Guidelines

**See**: [root CLAUDE.md](../CLAUDE.md) for project-wide policies

## Package Philosophy

**Public packages** (`pkg/`) - Reusable, well-documented APIs for external use
**Internal packages** (`internal/`) - Application-specific, not exported

## Package-Skill Mapping

| Package | Category | Skill to Invoke | Security Level |
|---------|----------|-----------------|----------------|
| pkg/auth | Security | contextd:pkg-security | Critical |
| pkg/checkpoint | Storage | contextd:pkg-storage | High |
| pkg/remediation | Storage | contextd:pkg-storage | Medium |
| pkg/config | Core | contextd:pkg-core | Medium |
| pkg/telemetry | Core | contextd:pkg-core | Low |
| pkg/logging | Core | contextd:pkg-core | High (secret redaction) |
| pkg/embedding | AI | contextd:pkg-ai | Medium |

## When Working in a Package

**Mandatory workflow**:
1. Read this file (quick orientation)
2. **Invoke category skill** from mapping table above
3. Follow skill's patterns and testing requirements
4. Before completion: Invoke appropriate completion skill

## Adding New Package

**MANDATORY**: Invoke `contextd:creating-package` skill before creating package.

This skill will:
- Guide package structure
- Assign to category
- Update this mapping table
- Update/create category skill if needed

## Standards Reference

@docs/standards/coding-standards.md
@docs/standards/testing-standards.md
@docs/standards/package-guidelines.md

## Quick Pattern Reference

**Service Pattern**:
```go
type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}
```

**Interface Design**:
```go
// Minimal, focused interfaces
type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    Save(ctx context.Context, item *Item) error
}
```

**Error Handling**:
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```
