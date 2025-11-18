---
name: contextd-pkg-storage
description: Use when working with checkpoint, remediation, cache, or skills packages - enforces database-per-project isolation, Qdrant patterns, and query security to prevent data leakage across projects and teams
---

# Storage Package Guidelines

## Overview

Storage packages (checkpoint, remediation, cache, skills) handle vector data with strict multi-tenant isolation. Database-per-project architecture is MANDATORY for security and performance. This skill enforces Qdrant patterns and prevents filter injection attacks.

## When to Use This Skill

**Use when:**
- Implementing checkpoint storage or search
- Working with remediation/skills shared storage
- Adding caching layer to storage packages
- Designing queries for vector data
- Creating new collections in Qdrant
- Testing multi-tenant isolation

**Symptoms requiring this skill:**
- "Should I use one database or database-per-project?"
- "Filters are simpler than separate databases"
- "User provides database name for flexibility"
- "Performance is slow with database switching"
- "How do I prevent cross-project data leakage?"

## Critical Rules (No Exceptions)

### 1. Database-Per-Project for Checkpoints (MANDATORY)

**Never use shared database for checkpoints. Never use filters for isolation.**

```go
// ✅ GOOD - Database-per-project isolation
projectHash := hashProjectPath(projectPath) // SHA256[:8]
database := fmt.Sprintf("project_%s", projectHash)
results, err := qdrant.Search(ctx, database, "checkpoints", vector, limit)

// ❌ BAD - Shared database with filters (FILTER INJECTION RISK)
database := "checkpoints"
filter := map[string]string{"project": projectPath}
results, err := qdrant.SearchWithFilter(ctx, database, "checkpoints", vector, filter, limit)

// ❌ BAD - Single shared database (DATA LEAKAGE)
database := "contextd"
results, err := qdrant.Search(ctx, database, "checkpoints", vector, limit)
```

**Why database-per-project:**
- Prevents filter injection attacks (no user-controlled filters)
- 10-16x faster queries (partition pruning at database level)
- Physical isolation (impossible to query wrong project)
- Simpler security model (database boundary = trust boundary)

### 2. Never Accept User-Provided Database Names

**Always derive database from projectPath. Never trust user input for security boundaries.**

```go
// ✅ GOOD - Derive database from projectPath
func SearchCheckpoints(ctx context.Context, projectPath, query string, limit int) ([]Result, error) {
    database := getDatabaseForProject(projectPath) // Derives project_<hash>
    vector, err := embedder.Embed(query)
    if err != nil {
        return nil, fmt.Errorf("embedding failed: %w", err)
    }
    return store.Search(ctx, database, "checkpoints", vector, limit)
}

// ❌ BAD - User-provided database (even with validation)
func SearchCheckpoints(ctx context.Context, database, query string, limit int) ([]Result, error) {
    // Validation is NOT enough - must derive, not accept
    if !strings.HasPrefix(database, "project_") {
        return nil, fmt.Errorf("invalid database")
    }
    vector, err := embedder.Embed(query)
    if err != nil {
        return nil, fmt.Errorf("embedding failed: %w", err)
    }
    return store.Search(ctx, database, "checkpoints", vector, limit)
}
```

**Why derive, never accept:**
- Validation ≠ security (bypass possible)
- Derivation creates cryptographic guarantee (SHA256 hash)
- No ambiguity about which project
- Prevents privilege escalation

**Attack example (why validation is insufficient):**

```go
// ❌ VULNERABLE - Validation only
func SearchCheckpoints(database string) {
    if !strings.HasPrefix(database, "project_") {
        return errors.New("invalid database")
    }
    // Attack: user provides "project_00000000" (common hash collision attempt)
    // Attack: user guesses "project_abc123de" (another user's project)
    qdrant.Search(database, ...)
}

// ✅ SECURE - Cryptographic derivation
func SearchCheckpoints(projectPath string) {
    // User cannot guess hash without knowing exact projectPath
    // Even if they know projectPath, hash derivation is deterministic and verifiable
    hash := sha256.Sum256([]byte(projectPath))
    database := fmt.Sprintf("project_%s", hex.EncodeToString(hash[:])[:8])
    qdrant.Search(database, ...)
}
```

**The attack:** With validation-only, attacker can:
1. Try common hashes (00000000, 11111111, abc123de)
2. Brute-force 8-character hex space (16^8 = 4 billion attempts)
3. Guess other users' database names

With cryptographic derivation, attacker needs:
1. Exact projectPath (filesystem path)
2. Cannot bypass without filesystem access
3. Hash collision computationally infeasible

### 3. Separate Project and Shared Data

**Checkpoints = project-specific. Remediations/Skills = shared within team.**

```go
// ✅ GOOD - Correct database selection
func (s *Service) SaveData(ctx context.Context, dataType string, projectPath string, data Data) error {
    var database string

    switch dataType {
    case "checkpoint":
        // Project-specific database
        database = fmt.Sprintf("project_%s", hashProjectPath(projectPath))
    case "remediation", "skill":
        // Shared database (or team-specific)
        database = "shared"
    default:
        return fmt.Errorf("unknown data type: %s", dataType)
    }

    return s.store.Upsert(ctx, database, dataType, data)
}

// ❌ BAD - Wrong database for data type
func (s *Service) SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
    // Checkpoint in shared database = DATA LEAKAGE
    return s.store.Upsert(ctx, "shared", "checkpoints", checkpoint)
}
```

**Database selection rules:**
- Checkpoints: ALWAYS `project_<hash>`
- Remediations: `shared` or `team_<name>`
- Skills: `shared` or `team_<name>`
- Research: `project_<hash>` (project-specific)
- Cache: Depends on cache scope (usually project-specific)

### 4. Validate Collection Names

**Whitelist allowed collections. Never trust user input.**

```go
// ✅ GOOD - Collection whitelist
var validCollections = map[string]bool{
    "checkpoints": true,
    "research":    true,
    "notes":       true,
}

func (s *Service) Search(ctx context.Context, collection, query string) ([]Result, error) {
    if !validCollections[collection] {
        return nil, fmt.Errorf("invalid collection: %s", collection)
    }
    // Proceed with search
}

// ❌ BAD - No validation
func (s *Service) Search(ctx context.Context, collection, query string) ([]Result, error) {
    // Direct use of user input
    return s.store.Search(ctx, s.database, collection, query, 10)
}
```

## Qdrant Best Practices

### Vector Dimensions

**Use consistent dimensions for all collections:**

```go
const (
    DimensionsBGESmall    = 384   // BAAI/bge-small-en-v1.5
    DimensionsOpenAISmall = 1536  // text-embedding-3-small
)

// ✅ GOOD - Explicit dimension validation
func (s *Service) CreateCollection(ctx context.Context, name string, dimensions int) error {
    if dimensions != DimensionsBGESmall && dimensions != DimensionsOpenAISmall {
        return fmt.Errorf("unsupported dimensions: %d", dimensions)
    }
    return s.store.CreateCollection(ctx, name, dimensions)
}
```

### Distance Metric

**Use Cosine similarity for semantic search:**

```go
// ✅ GOOD - Cosine distance for embeddings
config := qdrant.VectorParams{
    Size:     384,
    Distance: qdrant.DistanceCosine, // Best for normalized embeddings
}
```

### HNSW Indexing

**Enable HNSW for performance:**

```go
// ✅ GOOD - HNSW configuration
hnswConfig := qdrant.HnswConfig{
    M:              16,  // Number of edges per node
    EfConstruct:    100, // Construction time search depth
    FullScanThreshold: 10000, // Switch to full scan for small collections
}
```

## Testing Requirements

### Mandatory Multi-Tenant Isolation Test

**Every storage package MUST have this test:**

```go
func TestMultiTenantIsolation(t *testing.T) {
    project1 := "/home/user/project1"
    project2 := "/home/user/project2"

    service := setupTestService(t)

    // Save checkpoint to project1
    checkpoint1 := &Checkpoint{
        Summary: "Project 1 checkpoint",
        Project: project1,
        Content: "Sensitive project 1 data",
    }
    err := service.SaveCheckpoint(context.Background(), project1, checkpoint1)
    require.NoError(t, err)

    // Search from project2 should NOT find project1's checkpoint
    results, err := service.SearchCheckpoints(context.Background(), project2, "Project 1", 10)
    require.NoError(t, err)
    assert.Empty(results, "Project 2 should not see Project 1 checkpoints")

    // Search from project1 SHOULD find its own checkpoint
    results, err = service.SearchCheckpoints(context.Background(), project1, "Project 1", 10)
    require.NoError(t, err)
    assert.NotEmpty(results, "Project 1 should find its own checkpoints")
}
```

**This test is MANDATORY. No exceptions.**

### Collection Validation Test

```go
func TestCollectionValidation(t *testing.T) {
    service := setupTestService(t)

    tests := []struct {
        name       string
        collection string
        wantErr    bool
    }{
        {"valid checkpoint", "checkpoints", false},
        {"valid research", "research", false},
        {"invalid collection", "drop_table", true},
        {"sql injection attempt", "checkpoints; DROP TABLE", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := service.Search(context.Background(), tt.collection, "test", 10)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Common Mistakes and Fixes

| Mistake | Why It's Wrong | Fix |
|---------|----------------|-----|
| Using shared database for checkpoints | Data leakage across projects | Use `project_<hash>` database |
| Filter-based isolation | Filter injection attacks, slow queries | Database-per-project isolation |
| Accepting user database names | Security bypass via input manipulation | Derive from projectPath hash |
| Inconsistent vector dimensions | Search fails, corrupt results | Validate dimensions match model |
| No collection validation | Collection injection attacks | Whitelist allowed collections |
| Mixing project and shared data | Confusing security model | Clear separation by data type |

## Rationalization Table

**These excuses all mean: Follow database-per-project isolation.**

| Excuse | Reality |
|--------|---------|
| "Single database is simpler" | Violates multi-tenant isolation (PRIMARY security goal) |
| "Filters achieve same isolation" | Filter injection attacks + 10-16x slower queries |
| "We can refactor later" | Technical debt, harder to migrate with data in production |
| "User knows their database name" | Never trust user input for security boundaries |
| "Format validation is enough" | Validation ≠ derivation; must derive from projectPath |
| "50ms performance matters" | Security first, always. Optimize within secure model |
| "This is localhost, low risk" | Multi-tenant isolation prevents privilege escalation |
| "Shared database with filters is standard" | Not for security-critical multi-tenant systems |
| "I'll use database-per-project for new code, keep filters for existing" | No mixed approaches. Migrate existing code immediately |
| "Validation + sanitization is as good as derivation" | No. Cryptographic derivation is the ONLY acceptable approach |
| "I'll let user choose projectPath instead of database" | Indirect bypass. projectPath must come from filesystem, not user input |
| "Collection validation is optional for internal APIs" | MANDATORY for ALL code paths. No exceptions for "internal" |

## Red Flags - STOP and Review

**If you're thinking any of these, you're about to violate isolation:**

- "One database is simpler"
- "Filters work just as well"
- "User provides database name for flexibility"
- "Performance is more important than isolation"
- "We can add isolation later"
- "Format validation is sufficient"
- "New code uses isolation, existing code keeps filters"
- "User chooses projectPath (not database directly)"
- "Internal APIs don't need collection validation"
- "Sanitization makes user input safe"

**All of these mean: Re-read this skill. Use database-per-project.**

## No Mixed Approaches (MANDATORY)

**You cannot mix database-per-project and filter-based isolation.**

If you find existing code using filters:
1. Mark as technical debt
2. Create migration task
3. Migrate BEFORE adding new features
4. No "gradual transition" - migrate immediately

**Why no mixing:**
- Confusing security model (which code path is secure?)
- Filter injection still possible (attack surface remains)
- Maintenance nightmare (two patterns to maintain)
- False sense of security ("mostly isolated")

**Migration is MANDATORY. No exceptions.**

## Integration with Other Skills

**Before marking task complete:**
- Use `contextd:completing-major-task` with security verification
- Include multi-tenant isolation test results
- Verify database-per-project pattern used

**Before creating PR:**
- Use `contextd:code-review` skill
- Reviewer must verify isolation tests present and passing
- No shared database for checkpoints (blocking issue)

**For security-critical changes:**
- Use `contextd:security-check` skill
- Deep validation of database selection logic
- Attack scenario testing (filter injection, privilege escalation)

## Quick Reference

### Database Selection Pattern

```go
func getDatabaseForProject(projectPath string) string {
    hash := sha256.Sum256([]byte(projectPath))
    return fmt.Sprintf("project_%s", hex.EncodeToString(hash[:])[:8])
}

func getDatabaseForDataType(dataType, projectPath string) string {
    switch dataType {
    case "checkpoint", "research", "notes":
        return getDatabaseForProject(projectPath)
    case "remediation", "skill", "troubleshooting":
        return "shared" // or team-specific
    default:
        panic(fmt.Sprintf("unknown data type: %s", dataType))
    }
}
```

### Collection Creation Pattern

```go
func (s *Service) ensureCollection(ctx context.Context, database, collection string, dimensions int) error {
    exists, err := s.store.CollectionExists(ctx, database, collection)
    if err != nil {
        return fmt.Errorf("failed to check collection: %w", err)
    }

    if !exists {
        config := qdrant.CollectionConfig{
            VectorSize: dimensions,
            Distance:   qdrant.DistanceCosine,
            HnswConfig: qdrant.HnswConfig{
                M:           16,
                EfConstruct: 100,
            },
        }
        return s.store.CreateCollection(ctx, database, collection, config)
    }

    return nil
}
```

### Search Pattern

```go
func (s *Service) SearchCheckpoints(ctx context.Context, projectPath, query string, limit int) ([]Result, error) {
    // 1. Derive database (never accept from user)
    database := getDatabaseForProject(projectPath)

    // 2. Validate collection
    collection := "checkpoints"
    if !validCollections[collection] {
        return nil, fmt.Errorf("invalid collection")
    }

    // 3. Generate embedding
    vector, err := s.embedder.Embed(query)
    if err != nil {
        return nil, fmt.Errorf("embedding failed: %w", err)
    }

    // 4. Search with database-level isolation
    return s.store.Search(ctx, database, collection, vector, limit)
}
```

## Bottom Line

**Database-per-project isolation is MANDATORY for checkpoints. No filters. No shared databases. No user-provided database names.**

If you're not deriving the database from projectPath hash, you're violating multi-tenant isolation.

Test isolation. Derive databases. Validate collections. Security first, always.
