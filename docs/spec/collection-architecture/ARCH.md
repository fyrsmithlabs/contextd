# Collection Architecture

**Feature**: Collection Architecture
**Status**: Draft
**Created**: 2025-11-22

## System Context

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         contextd MCP Server                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Collection Manager                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │  Database   │  │ Collection  │  │   Schema    │              │   │
│  │  │  Resolver   │  │  Factory    │  │  Validator  │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Qdrant Cluster                                 │
│                                                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │ DB: acme_corp   │  │ DB: globex      │  │ DB: initech     │         │
│  │                 │  │                 │  │                 │         │
│  │ org_memories    │  │ org_memories    │  │ org_memories    │         │
│  │ org_policies    │  │ org_policies    │  │ org_policies    │         │
│  │ platform_*      │  │ engineering_*   │  │ dev_*           │         │
│  │ frontend_*      │  │ sales_*         │  │ ops_*           │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Database Resolver

**Responsibility**: Map org/tenant to correct Qdrant database.

```go
type DatabaseResolver interface {
    Resolve(ctx context.Context, orgID string) (string, error)
    Exists(ctx context.Context, orgID string) (bool, error)
}

// Implementation routes to org's dedicated database
func (r *resolver) Resolve(ctx context.Context, orgID string) (string, error) {
    // Database name = org ID (sanitized)
    return sanitizeDBName(orgID), nil
}
```

### Collection Factory

**Responsibility**: Create and manage collections with correct configuration.

```go
type CollectionFactory interface {
    CreateOrgCollections(ctx context.Context, db string) error
    CreateTeamCollections(ctx context.Context, db, team string) error
    CreateProjectCollections(ctx context.Context, db, team, project string) error
    DeleteTeamCollections(ctx context.Context, db, team string) error
    DeleteProjectCollections(ctx context.Context, db, team, project string) error
}

type CollectionConfig struct {
    Name           string
    VectorSize     int
    Distance       Distance // Cosine, Euclid, Dot
    PayloadIndexes []PayloadIndex
}
```

### Schema Validator

**Responsibility**: Validate payloads against collection schemas.

```go
type SchemaValidator interface {
    Validate(collectionType string, payload map[string]any) error
    GetSchema(collectionType string) (*Schema, error)
}
```

## Database Lifecycle

### Organization Provisioning

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Provisioner  │────►│   Qdrant     │────►│  Collection  │
│              │     │ CreateDB()   │     │   Factory    │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                     ┌───────────────────────────┼───────────────────────────┐
                     ▼                           ▼                           ▼
              ┌─────────────┐            ┌─────────────┐            ┌─────────────┐
              │org_memories │            │org_policies │            │ org_skills  │
              └─────────────┘            └─────────────┘            └─────────────┘
```

**Sequence**:
1. New org signs up
2. Create Qdrant database with org ID
3. Create all org-level collections with indexes
4. Initialize default policies/standards (if templates exist)

### Team Provisioning

```go
func (f *CollectionFactory) CreateTeamCollections(ctx context.Context, db, team string) error {
    collections := []CollectionConfig{
        {Name: fmt.Sprintf("%s_memories", team), ...},
        {Name: fmt.Sprintf("%s_remediations", team), ...},
        {Name: fmt.Sprintf("%s_coding_standards", team), ...},
    }

    for _, cfg := range collections {
        if err := f.qdrant.CreateCollection(ctx, db, cfg); err != nil {
            return fmt.Errorf("create %s: %w", cfg.Name, err)
        }
    }
    return nil
}
```

### Project Provisioning

```go
func (f *CollectionFactory) CreateProjectCollections(ctx context.Context, db, team, project string) error {
    prefix := fmt.Sprintf("%s_%s", team, project)

    collections := []CollectionConfig{
        {Name: fmt.Sprintf("%s_memories", prefix), ...},
        {Name: fmt.Sprintf("%s_remediations", prefix), ...},
        {Name: fmt.Sprintf("%s_codebase", prefix), ...},
        {Name: fmt.Sprintf("%s_sessions", prefix), ...},
        {Name: fmt.Sprintf("%s_checkpoints", prefix), ...},
    }

    for _, cfg := range collections {
        if err := f.qdrant.CreateCollection(ctx, db, cfg); err != nil {
            return fmt.Errorf("create %s: %w", cfg.Name, err)
        }
    }
    return nil
}
```

## Query Routing

### Scope Resolution

```go
type ScopeResolver struct {
    dbResolver DatabaseResolver
}

func (r *ScopeResolver) ResolveCollections(ctx context.Context, orgID, team, project, collType string) []string {
    db := r.dbResolver.Resolve(ctx, orgID)

    // Return collections from specific to general
    return []string{
        fmt.Sprintf("%s/%s_%s_%s", db, team, project, collType),  // project
        fmt.Sprintf("%s/%s_%s", db, team, collType),               // team
        fmt.Sprintf("%s/org_%s", db, collType),                    // org
    }
}
```

### Hierarchical Query

```go
func (m *MemoryManager) SearchHierarchical(ctx context.Context, req SearchRequest) ([]Memory, error) {
    collections := m.scopeResolver.ResolveCollections(
        ctx, req.OrgID, req.Team, req.Project, "memories",
    )

    var allResults []Memory
    seen := make(map[string]bool)

    for _, collection := range collections {
        results, err := m.qdrant.Search(ctx, collection, req.Query)
        if err != nil {
            continue // skip unavailable collections
        }

        // Deduplicate (promoted items may appear at multiple levels)
        for _, r := range results {
            if !seen[r.ID] {
                seen[r.ID] = true
                allResults = append(allResults, r)
            }
        }

        if len(allResults) >= req.Limit {
            break
        }
    }

    return m.rankResults(allResults, req.Limit), nil
}
```

## Vector Configuration

```yaml
vector_config:
  # OpenAI embeddings
  openai:
    model: "text-embedding-3-small"
    dimensions: 1536
    distance: "Cosine"

  # Local embeddings (e.g., sentence-transformers)
  local:
    model: "all-MiniLM-L6-v2"
    dimensions: 384
    distance: "Cosine"

  # Default for new orgs
  default: "openai"
```

## Index Configuration

```go
var StandardIndexes = map[string][]PayloadIndex{
    "memories": {
        {Field: "confidence", Type: Float},
        {Field: "outcome", Type: Keyword},
        {Field: "tags", Type: Keyword},
        {Field: "created_at", Type: DateTime},
        {Field: "last_used", Type: DateTime},
    },
    "remediations": {
        {Field: "status", Type: Keyword},
        {Field: "confidence", Type: Float},
        {Field: "verified", Type: Bool},
        {Field: "created_at", Type: DateTime},
    },
    "policies": {
        {Field: "category", Type: Keyword},
        {Field: "severity", Type: Keyword},
        {Field: "effective_date", Type: DateTime},
    },
    // ... other collection types
}
```

## Backup and Recovery

### Per-Database Snapshots

```go
type BackupManager interface {
    CreateSnapshot(ctx context.Context, db string) (*Snapshot, error)
    RestoreSnapshot(ctx context.Context, db string, snapshotID string) error
    ListSnapshots(ctx context.Context, db string) ([]*Snapshot, error)
    DeleteSnapshot(ctx context.Context, db, snapshotID string) error
}
```

### GDPR Compliance

```go
// Complete org deletion
func (m *BackupManager) DeleteOrganization(ctx context.Context, orgID string) error {
    db := m.dbResolver.Resolve(ctx, orgID)

    // Create final snapshot for audit
    snapshot, err := m.CreateSnapshot(ctx, db)
    if err != nil {
        return fmt.Errorf("create final snapshot: %w", err)
    }

    // Store snapshot reference in audit log
    m.auditLog.Record(ctx, "org_deletion", orgID, snapshot.ID)

    // Delete entire database
    return m.qdrant.DeleteDatabase(ctx, db)
}
```

## Configuration

```yaml
collection_architecture:
  qdrant:
    host: "localhost"
    port: 6334
    grpc_port: 6333
    api_key: "${QDRANT_API_KEY}"

  defaults:
    vector_size: 1536
    distance: "Cosine"
    on_disk: false

  provisioning:
    org_collections:
      - org_memories
      - org_remediations
      - org_policies
      - org_coding_standards
      - org_repo_standards
      - org_skills
      - org_agents
      - org_anti_patterns
      - org_feedback

    team_collections:
      - "{team}_memories"
      - "{team}_remediations"
      - "{team}_coding_standards"

    project_collections:
      - "{team}_{project}_memories"
      - "{team}_{project}_remediations"
      - "{team}_{project}_codebase"
      - "{team}_{project}_sessions"
      - "{team}_{project}_checkpoints"
```
