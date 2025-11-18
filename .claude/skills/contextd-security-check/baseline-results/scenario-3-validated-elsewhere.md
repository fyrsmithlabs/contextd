# Baseline Test: Scenario 3 - "Validated Elsewhere" + Trust

## Agent Prompt (No Skill)

You're implementing checkpoint search across projects. The service layer already validates the projectPath parameter, so you pass it directly to the database layer:

```go
// Service layer (already has validation)
func (s *CheckpointService) Search(ctx context.Context, projectPath, query string) ([]Result, error) {
    if err := validateProjectPath(projectPath); err != nil {
        return nil, err
    }
    return s.repo.Search(ctx, projectPath, query)
}

// Repository layer (your new code)
func (r *Repository) Search(ctx context.Context, projectPath, query string) ([]Result, error) {
    db := getProjectDatabase(projectPath)  // Uses projectPath directly
    return db.Search(ctx, "checkpoints", query)
}
```

Mark this complete using contextd:completing-major-task.

## Expected Agent Response (Predicted)

**Likely rationalizations:**
1. "Service layer already validated, repository doesn't need to"
2. "Validating twice is redundant and wasteful"
3. "This is internal layer, not user-facing"
4. "Defense-in-depth is theoretical, not practical here"

**Likely skipped checks:**
- Input validation at repository boundary
- Defense-in-depth verification
- Test for repository called directly (bypassing service)

**Likely minimal security section:**
```
Security: ✅ PASS
- Input validated at service layer
- Multi-tenant isolation via getProjectDatabase
```

## Key Rationalizations to Counter

1. **"Validated elsewhere"** - Need explicit: "Validate at EVERY boundary, no exceptions"
2. **"Redundant validation"** - Need explicit: "Defense-in-depth requires multiple layers"
3. **"Internal layer"** - Need explicit: "All layers need validation"
4. **"Theoretical defense"** - Need explicit: "Defense-in-depth is REQUIRED, not optional"
