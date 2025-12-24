# Institutional Knowledge Architecture

**Feature**: Institutional Knowledge (Layer 3)
**Status**: Draft
**Created**: 2025-11-22

## System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                        Organization                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Org Knowledge Base                      │   │
│  │  (policies, coding_standards, org_memories)              │   │
│  └─────────────────────────────────────────────────────────┘   │
│         ▲                    ▲                    ▲             │
│         │ promotion          │ promotion          │             │
│  ┌──────┴──────┐      ┌──────┴──────┐      ┌──────┴──────┐     │
│  │ Team: Plat  │      │ Team: Front │      │ Team: Data  │     │
│  │ knowledge   │      │ knowledge   │      │ knowledge   │     │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘     │
│         │                    │                    │             │
│  ┌──────┴──────┐      ┌──────┴──────┐      ┌──────┴──────┐     │
│  │ Project A   │      │ Project C   │      │ Project E   │     │
│  │ Project B   │      │ Project D   │      │ Project F   │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Knowledge Promoter

**Responsibility**: Detect cross-project patterns and promote to higher scopes.

```go
type KnowledgePromoter interface {
    DetectCandidates(ctx context.Context, scope Scope) ([]PromotionCandidate, error)
    Promote(ctx context.Context, candidate PromotionCandidate) (*PromotedItem, error)
    ManualPromote(ctx context.Context, itemID string, targetScope Scope) (*PromotedItem, error)
}

type PromotionCandidate struct {
    SourceItems   []string  // IDs of similar items across projects
    TargetScope   Scope
    Similarity    float64
    Confidence    float64   // average confidence
    Generalized   string    // proposed generalized content
}

type PromotedItem struct {
    ID            string
    SourceItems   []string
    Scope         Scope
    Title         string
    Description   string
    Content       string
    Confidence    float64
    PromotedAt    time.Time
}
```

### Briefing Generator

**Responsibility**: Create onboarding context from institutional knowledge.

```go
type BriefingGenerator interface {
    Generate(ctx context.Context, req BriefingRequest) (*Briefing, error)
}

type BriefingRequest struct {
    Project string
    Team    string
    Org     string
    Depth   BriefingDepth // minimal, standard, comprehensive
    Budget  int           // max tokens
}

type Briefing struct {
    OrgPatterns     []BriefingItem
    TeamPatterns    []BriefingItem
    ProjectPatterns []BriefingItem
    Policies        []BriefingItem
    Standards       []BriefingItem
    TotalTokens     int
}

type BriefingItem struct {
    Type       string  // memory, policy, standard
    Title      string
    Summary    string
    Confidence float64
    Scope      Scope
}
```

### Scope Resolver

**Responsibility**: Determine knowledge access based on hierarchy.

```go
type ScopeResolver interface {
    GetAccessibleScopes(ctx context.Context, project, team, org string) ([]Scope, error)
    ResolveConflicts(items []KnowledgeItem) []KnowledgeItem
}
```

## Data Flow

### Promotion Detection Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Consolidator │────►│  Promoter    │────►│   Qdrant     │
│ (scheduled)  │     │              │     │   (search)   │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │ Similar Item │
                     │  Detector    │
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │ Generalizer  │
                     │   (LLM)      │
                     └──────────────┘
```

**Sequence**:
1. Scheduled job triggers promotion detection
2. For each team, find memories with similar embeddings across projects
3. If similarity > 0.85 and count >= 2, create promotion candidate
4. LLM generalizes content (strips project-specific details)
5. Create promoted item at team scope
6. Link original items to promoted item

### Briefing Generation Flow

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent   │────►│   Briefing   │────►│   Qdrant     │────►│   Ranker     │
│          │     │  Generator   │     │  (multi-coll)│     │              │
└──────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                                                   │
                                                                   ▼
                                                           ┌──────────────┐
                                                           │   Budget     │
                                                           │   Fitter     │
                                                           └──────────────┘
```

**Sequence**:
1. Agent requests briefing for project
2. Query org policies and standards
3. Query org, team, project memories (high confidence only)
4. Rank by relevance and confidence
5. Fit to token budget
6. Format and return briefing

## Promotion Algorithm

### Detection Phase

```go
func (p *KnowledgePromoter) DetectCandidates(ctx context.Context, team string) ([]PromotionCandidate, error) {
    // Get all projects in team
    projects, err := p.projectStore.ListByTeam(ctx, team)
    if err != nil {
        return nil, err
    }

    // For each project, get high-confidence memories
    var allMemories []Memory
    for _, proj := range projects {
        mems, err := p.memoryStore.List(ctx, MemoryFilter{
            Project:       proj.ID,
            MinConfidence: 0.8,
        })
        if err != nil {
            continue
        }
        allMemories = append(allMemories, mems...)
    }

    // Cluster by embedding similarity
    clusters := p.clusterBySimilarity(allMemories, 0.85)

    // Filter clusters that span multiple projects
    var candidates []PromotionCandidate
    for _, cluster := range clusters {
        projects := uniqueProjects(cluster)
        if len(projects) >= 2 {
            candidates = append(candidates, PromotionCandidate{
                SourceItems: itemIDs(cluster),
                TargetScope: ScopeTeam,
                Similarity:  cluster.AvgSimilarity,
                Confidence:  cluster.AvgConfidence,
            })
        }
    }

    return candidates, nil
}
```

### Generalization Phase

```go
func (p *KnowledgePromoter) Generalize(ctx context.Context, items []Memory) (string, error) {
    prompt := `Given these related patterns from different projects, create a generalized version.

Original patterns:
{{range .Items}}
Project: {{.Project}}
Title: {{.Title}}
Content: {{.Content}}
---
{{end}}

Create a generalized pattern that:
1. Removes project-specific names and paths
2. Keeps the core strategy/approach
3. Uses placeholder language where specifics vary

Output only the generalized content, no explanation.`

    return p.llm.Complete(ctx, prompt, map[string]any{"Items": items})
}
```

## Conflict Resolution

```go
func (r *ScopeResolver) ResolveConflicts(items []KnowledgeItem) []KnowledgeItem {
    // Group by semantic similarity
    clusters := clusterBySimilarity(items, 0.9)

    var resolved []KnowledgeItem
    for _, cluster := range clusters {
        if len(cluster) == 1 {
            resolved = append(resolved, cluster[0])
            continue
        }

        // Multiple items in cluster = potential conflict
        // Prefer more specific scope
        winner := cluster[0]
        for _, item := range cluster[1:] {
            if scopePriority(item.Scope) > scopePriority(winner.Scope) {
                winner = item
            }
        }
        resolved = append(resolved, winner)
    }

    return resolved
}

func scopePriority(s Scope) int {
    switch s {
    case ScopeProject:
        return 3 // highest priority
    case ScopeTeam:
        return 2
    case ScopeOrg:
        return 1
    default:
        return 0
    }
}
```

## Configuration

```yaml
institutional_knowledge:
  promotion:
    enabled: true
    schedule: "0 2 * * *"  # daily at 2 AM
    min_confidence: 0.8
    min_similarity: 0.85
    min_sources: 2
    generalization_model: "claude-3-haiku"

  briefing:
    default_depth: "standard"
    depths:
      minimal:
        max_tokens: 300
        policies: true
        standards: true
        memories: false
      standard:
        max_tokens: 800
        policies: true
        standards: true
        memories: true
        memory_limit: 5
      comprehensive:
        max_tokens: 1500
        policies: true
        standards: true
        memories: true
        memory_limit: 10

  scoping:
    prefer_specific: true
    dedupe_across_scopes: true
```
