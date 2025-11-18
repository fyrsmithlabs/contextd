# Multi-Tenant Workflows

**Parent**: [../SPEC.md](../SPEC.md)

This document describes common workflows and usage examples for contextd's multi-tenant architecture.

---

## Database Creation Workflow

### Create Project Database

```go
import "github.com/axyzlabs/contextd/pkg/vectorstore"

// Step 1: Generate database name
projectPath := "/home/user/projects/myapp"
dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)
// Result: "project_a1b2c3d4"

// Step 2: Create database
db := vectorstore.Database{
    Name: dbName,
    Type: vectorstore.DatabaseTypeProject,
    Metadata: map[string]string{
        "project_path": projectPath,
        "created_at":   time.Now().Format(time.RFC3339),
    },
}

err := store.CreateDatabase(ctx, db)
if err != nil {
    log.Fatalf("Failed to create database: %v", err)
}
```

### Create Shared Database

```go
// Shared database has fixed name
db := vectorstore.Database{
    Name: "shared",
    Type: vectorstore.DatabaseTypeShared,
}

err := store.CreateDatabase(ctx, db)
if err != nil {
    log.Fatalf("Failed to create shared database: %v", err)
}
```

---

## Collection Creation Workflow

### Create Project-Specific Collection

```go
// Step 1: Get project database name
projectPath := "/home/user/projects/myapp"
dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

// Step 2: Define collection schema
schema := vectorstore.CollectionSchema{
    Name:           "checkpoints",
    VectorDim:      1536,  // OpenAI text-embedding-3-small
    DistanceMetric: vectorstore.DistanceCosine,
    Fields: map[string]vectorstore.FieldType{
        "summary":    vectorstore.FieldTypeString,
        "content":    vectorstore.FieldTypeString,
        "created_at": vectorstore.FieldTypeInt64,
    },
    Indexed: []string{"created_at"},
}

// Step 3: Create collection
err := store.CreateCollection(ctx, dbName, "checkpoints", schema)
if err != nil {
    log.Fatalf("Failed to create collection: %v", err)
}
```

### Create Shared Collection

```go
// Shared collections use "shared" database
schema := vectorstore.CollectionSchema{
    Name:           "remediations",
    VectorDim:      1536,
    DistanceMetric: vectorstore.DistanceCosine,
}

err := store.CreateCollection(ctx, "shared", "remediations", schema)
if err != nil {
    log.Fatalf("Failed to create shared collection: %v", err)
}
```

---

## Vector Insert Workflow

### Insert Project-Specific Data

```go
// Step 1: Get project database
projectPath := "/home/user/projects/myapp"
dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

// Step 2: Generate embedding
embedding, err := embedder.Embed(ctx, "Completed feature X")
if err != nil {
    log.Fatalf("Failed to generate embedding: %v", err)
}

// Step 3: Create vector
vector := vectorstore.Vector{
    ID:        "checkpoint_001",
    Embedding: embedding,  // []float32
    Payload: map[string]interface{}{
        "summary":    "Completed feature X",
        "content":    "Implemented user authentication with JWT tokens",
        "created_at": time.Now().Unix(),
    },
}

// Step 4: Insert to project database
err = store.Insert(ctx, dbName, "checkpoints", []vectorstore.Vector{vector})
if err != nil {
    log.Fatalf("Failed to insert vector: %v", err)
}
```

### Insert Shared Data

```go
// Insert to shared database (accessible by all projects)
embedding, err := embedder.Embed(ctx, "ImportError: No module named 'flask'")
if err != nil {
    log.Fatalf("Failed to generate embedding: %v", err)
}

vector := vectorstore.Vector{
    ID:        "remediation_001",
    Embedding: embedding,
    Payload: map[string]interface{}{
        "error_message": "ImportError: No module named 'flask'",
        "solution":      "pip install flask",
        "category":      "python",
    },
}

// Use "shared" database
err = store.Insert(ctx, "shared", "remediations", []vectorstore.Vector{vector})
if err != nil {
    log.Fatalf("Failed to insert remediation: %v", err)
}
```

---

## Vector Search Workflow

### Search Project-Specific Data

```go
// Step 1: Get project database
projectPath := "/home/user/projects/myapp"
dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

// Step 2: Generate query embedding
queryEmbedding, err := embedder.Embed(ctx, "feature implementation")
if err != nil {
    log.Fatalf("Failed to generate query embedding: %v", err)
}

// Step 3: Create search query
query := vectorstore.SearchQuery{
    Vector: queryEmbedding,
    TopK:   10,
    Filter: vectorstore.Filter{},  // No project filter needed!
}

// Step 4: Search within project database (physical isolation)
results, err := store.Search(ctx, dbName, "checkpoints", query)
if err != nil {
    log.Fatalf("Failed to search: %v", err)
}

// Step 5: Process results
for _, r := range results {
    fmt.Printf("Checkpoint: %s (score: %.2f)\n", r.Payload["summary"], r.Score)
}
```

### Search Shared Knowledge

```go
// Step 1: Generate query embedding
queryEmbedding, err := embedder.Embed(ctx, "ImportError flask")
if err != nil {
    log.Fatalf("Failed to generate query embedding: %v", err)
}

// Step 2: Search shared database (cross-project)
query := vectorstore.SearchQuery{
    Vector: queryEmbedding,
    TopK:   5,
}

results, err := store.Search(ctx, "shared", "remediations", query)
if err != nil {
    log.Fatalf("Failed to search: %v", err)
}

// Step 3: Process results (from ALL projects)
for _, r := range results {
    fmt.Printf("Solution: %s (score: %.2f)\n", r.Payload["solution"], r.Score)
}
```

---

## Migration Workflow

### Full Migration Process

```bash
# Step 1: Backup legacy data
contextd backup create --output /backup/legacy.tar.gz

# Step 2: Analyze project distribution
contextd migrate analyze --report project_distribution.json
# Output: List of unique project_path values and vector counts

# Step 3: Create project databases (dry-run first)
contextd migrate create-databases --dry-run
contextd migrate create-databases
# Creates: shared, project_<hash> for each unique project_path

# Step 4: Migrate data (dry-run first)
contextd migrate data --dry-run
contextd migrate data --concurrency 4
# Moves vectors from default to appropriate project databases

# Step 5: Validate migration
contextd migrate validate
# Checks: All vectors migrated, no data loss, checksums match

# Step 6: Restart service with v2.0 (multi-tenant always enabled)
systemctl --user restart contextd

# Step 7: Cleanup legacy database (after validation)
contextd migrate cleanup --confirm
# Deletes: default database
```

### Migration Analysis Output

```json
{
  "total_vectors": 1523,
  "unique_projects": 5,
  "project_distribution": {
    "/home/user/projects/contextd": 892,
    "/home/user/projects/myapp": 345,
    "/home/user/projects/website": 156,
    "/home/user/projects/api": 89,
    "/home/user/projects/cli": 41
  },
  "estimated_databases": 6,
  "estimated_migration_time": "45s"
}
```

### Migration Validation Output

```
✓ All 1523 vectors migrated
✓ Checksums match (SHA256)
✓ No data loss detected
✓ 5 project databases created
✓ 1 shared database created
✓ All collections created successfully

Ready for cleanup (contextd migrate cleanup --confirm)
```

---

## Project Hash Generation

### Example Hash Calculations

```go
import (
    "crypto/sha256"
    "fmt"
)

func projectHash(path string) string {
    h := sha256.Sum256([]byte(path))
    return fmt.Sprintf("%x", h)[:8]
}

// Examples
"/home/user/projects/contextd"    → "abc123de"
"/home/user/projects/myapp"       → "def456gh"
"/home/alice/work/project1"       → "789abc01"
"/home/bob/personal/notes"        → "234def56"
```

### Hash Properties

**Deterministic**:
```go
hash1 := projectHash("/home/user/projects/contextd")
hash2 := projectHash("/home/user/projects/contextd")
// hash1 == hash2 (always)
```

**Collision-Resistant**:
```go
hash1 := projectHash("/home/user/projects/contextd")
hash2 := projectHash("/home/user/projects/contextd2")
// hash1 != hash2 (extremely unlikely to collide)
```

**Opaque** (one-way):
```go
hash := projectHash("/home/user/projects/contextd")  // "abc123de"
// Cannot recover "/home/user/projects/contextd" from "abc123de"
```

---

## Database Routing Workflow

### Service Layer Routing

```go
// Checkpoint service automatically routes to project database
type CheckpointService struct {
    store    vectorstore.UniversalVectorStore
    embedder embedding.Service
}

func (s *CheckpointService) Save(ctx context.Context, projectPath string, cp *Checkpoint) error {
    // Automatic routing to project database
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

    // No filter needed - physical isolation!
    return s.store.Insert(ctx, dbName, "checkpoints", []vectorstore.Vector{cp.ToVector()})
}
```

### Remediation Service Routing

```go
// Remediation service always uses shared database
type RemediationService struct {
    store vectorstore.UniversalVectorStore
}

func (s *RemediationService) Save(ctx context.Context, rem *Remediation) error {
    // Always route to shared database
    return s.store.Insert(ctx, "shared", "remediations", []vectorstore.Vector{rem.ToVector()})
}
```

---

## Multi-Project Search Workflow

### Sequential Multi-Project Search

```go
// Search across multiple projects (sequential)
func SearchMultipleProjects(ctx context.Context, store vectorstore.UniversalVectorStore, projects []string, query string) ([]vectorstore.SearchResult, error) {
    // Generate query embedding once
    queryEmbedding, err := embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed: %w", err)
    }

    // Search each project database
    allResults := make([]vectorstore.SearchResult, 0)
    for _, projectPath := range projects {
        dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

        results, err := store.Search(ctx, dbName, "checkpoints", vectorstore.SearchQuery{
            Vector: queryEmbedding,
            TopK:   10,
        })
        if err != nil {
            // Log error, continue with other projects
            log.Printf("Failed to search project %s: %v", projectPath, err)
            continue
        }

        allResults = append(allResults, results...)
    }

    // Sort by score (descending)
    sort.Slice(allResults, func(i, j int) bool {
        return allResults[i].Score > allResults[j].Score
    })

    return allResults, nil
}
```

### Concurrent Multi-Project Search

```go
// Search across multiple projects (concurrent)
func SearchMultipleProjectsConcurrent(ctx context.Context, store vectorstore.UniversalVectorStore, projects []string, query string) ([]vectorstore.SearchResult, error) {
    // Generate query embedding once
    queryEmbedding, err := embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed: %w", err)
    }

    // Concurrent search with goroutines
    type searchResult struct {
        results []vectorstore.SearchResult
        err     error
    }

    resultsChan := make(chan searchResult, len(projects))

    for _, projectPath := range projects {
        go func(path string) {
            dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, path)

            results, err := store.Search(ctx, dbName, "checkpoints", vectorstore.SearchQuery{
                Vector: queryEmbedding,
                TopK:   10,
            })

            resultsChan <- searchResult{results: results, err: err}
        }(projectPath)
    }

    // Collect results
    allResults := make([]vectorstore.SearchResult, 0)
    for i := 0; i < len(projects); i++ {
        result := <-resultsChan
        if result.err != nil {
            log.Printf("Search error: %v", result.err)
            continue
        }
        allResults = append(allResults, result.results...)
    }

    // Sort by score
    sort.Slice(allResults, func(i, j int) bool {
        return allResults[i].Score > allResults[j].Score
    })

    return allResults, nil
}
```

---

## Database Management Workflow

### List Databases

```go
// List all project databases
databases, err := store.ListDatabases(ctx, vectorstore.DatabaseTypeProject)
if err != nil {
    log.Fatalf("Failed to list databases: %v", err)
}

for _, db := range databases {
    fmt.Printf("Database: %s (project: %s)\n", db.Name, db.Metadata["project_path"])
}
```

### Delete Project Database

```go
// Delete project database (and all its data)
projectPath := "/home/user/projects/old-project"
dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)

err := store.DeleteDatabase(ctx, dbName)
if err != nil {
    log.Fatalf("Failed to delete database: %v", err)
}
```

---

## Monitoring Workflow

### Collect Database Metrics

```go
// Count databases by type
func CollectDatabaseMetrics(ctx context.Context, store vectorstore.UniversalVectorStore) error {
    // Shared databases
    shared, err := store.ListDatabases(ctx, vectorstore.DatabaseTypeShared)
    if err != nil {
        return fmt.Errorf("failed to list shared: %w", err)
    }
    metrics.DatabasesTotal.WithLabelValues("shared").Set(float64(len(shared)))

    // Project databases
    projects, err := store.ListDatabases(ctx, vectorstore.DatabaseTypeProject)
    if err != nil {
        return fmt.Errorf("failed to list projects: %w", err)
    }
    metrics.DatabasesTotal.WithLabelValues("project").Set(float64(len(projects)))

    return nil
}
```

### Trace Vector Operations

```go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("contextd")

func (s *Service) Search(ctx context.Context, projectPath, query string) ([]*Checkpoint, error) {
    // Create trace span
    ctx, span := tracer.Start(ctx, "CheckpointService.Search")
    defer span.End()

    // Add trace attributes
    span.SetAttributes(
        attribute.String("contextd.project_path", projectPath),
        attribute.String("contextd.query", query),
    )

    // Perform search
    dbName := vectorstore.GetDatabaseName(vectorstore.DatabaseTypeProject, projectPath)
    results, err := s.store.Search(ctx, dbName, "checkpoints", query)

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Record result count
    span.SetAttributes(attribute.Int("contextd.result_count", len(results)))

    return results, nil
}
```

---

## Summary

**Common Workflows**:

1. **Database Creation**: Project and shared databases
2. **Collection Creation**: Schema definition and creation
3. **Vector Insert**: Project-specific and shared data
4. **Vector Search**: Project-scoped and shared searches
5. **Migration**: Analyze, create, migrate, validate, cleanup
6. **Project Hash**: Deterministic, collision-resistant naming
7. **Database Routing**: Automatic routing in service layer
8. **Multi-Project Search**: Sequential and concurrent
9. **Database Management**: List, delete, monitor
10. **Observability**: Metrics and tracing

**Key Principles**:
- No filters needed (physical isolation)
- Hash-based database naming
- Shared database for cross-project knowledge
- Service layer handles routing
- Comprehensive monitoring
