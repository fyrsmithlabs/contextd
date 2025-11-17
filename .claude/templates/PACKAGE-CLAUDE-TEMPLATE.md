# Package: [PACKAGE_NAME]

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

[Brief 1-2 sentence description of what this package does]

## Specification

**Full Spec**: [`docs/specs/[FEATURE]/SPEC.md`](../../docs/specs/[FEATURE]/SPEC.md)

**Quick Summary**:
- **Problem**: [What problem does this solve?]
- **Solution**: [How does it solve it?]
- **Key Features**: [Bullet points of main features]

## Architecture

**Design Pattern**: [Service pattern / Factory / Singleton / etc.]

**Dependencies**:
- [List key dependencies with purpose]
- Example: `pkg/embedding` - Generate vector embeddings

**Used By**:
- [List packages/components that use this]
- Example: `cmd/contextd` - API server

## Key Components

### Main Types

```go
// [Brief description]
type [MainType] struct {
    // Key fields
}
```

### Main Functions

```go
// [Brief description]
func [KeyFunction](ctx context.Context, ...) ([ReturnType], error)
```

## Usage Example

```go
// Typical usage pattern
svc := [package].New([dependencies])
result, err := svc.[MainMethod](ctx, input)
if err != nil {
    return err
}
```

## Testing

**Test Coverage**: [Current %] (Target: ≥80%)

**Key Test Files**:
- `[package]_test.go` - Unit tests
- `[package]_integration_test.go` - Integration tests (if applicable)

**Running Tests**:
```bash
go test ./pkg/[package]/
go test -cover ./pkg/[package]/
go test -race ./pkg/[package]/
```

## Configuration

**Environment Variables** (if applicable):
- `[VAR_NAME]` - [Description] (default: [value])

## Security Considerations

[Any security-specific notes, or "None" if not applicable]

## Performance Notes

[Any performance considerations, or "Standard performance expectations" if not applicable]

## Related Documentation

- Spec: [`docs/specs/[FEATURE]/SPEC.md`](../../docs/specs/[FEATURE]/SPEC.md)
- Research: [`docs/specs/[FEATURE]/research/`](../../docs/specs/[FEATURE]/research/)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
