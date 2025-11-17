# Context Usage Estimates - Multi-Tenant Architecture

**Purpose**: Estimate token/context usage for v2.1 security implementation
**Critical**: Contextd's PRIMARY goal is context optimization - architecture must minimize token usage

---

## Current Context Usage (v2.0 - Broken)

### Search Result Context Bloat

**Problem**: Shared database returns irrelevant results

```
User searches "database error" in personal-blog:
  Returns 50 results:
    - 10 from personal-blog ✅ Relevant
    - 15 from client-project-a ❌ Wrong tech stack
    - 15 from client-project-b ❌ Wrong context
    - 10 from side-projects ⚠️ Maybe relevant

Context sent to Claude: ALL 50 results
Tokens used: ~12,000 tokens (50 results × ~240 tokens each)
Relevant tokens: ~2,400 tokens (10 results)
Wasted: ~9,600 tokens (80% waste)
```

**Impact**: Context pollution, slower responses, higher costs

---

## v2.1 Context Usage (Optimized)

### 4-Tier Search with Scope Ranking

```
User searches "database error" in personal-blog:

  Tier 1 (Project): 5 results (PostgreSQL for this blog)
    Context: ~1,200 tokens
    Relevance: 95%

  Tier 2 (Team - if applicable): 3 results (team patterns)
    Context: ~720 tokens
    Relevance: 75%

  Tier 3 (Org): 2 results (org standards)
    Context: ~480 tokens
    Relevance: 60%

  Tier 4 (Public): 0 results (disabled)
    Context: 0 tokens

Total context: ~2,400 tokens (10 relevant results)
Wasted: ~0 tokens (0% waste)
Improvement: 5x reduction (12,000 → 2,400 tokens)
```

---

## Context Optimization Strategies

### Strategy 1: Early Termination

**Stop searching when enough relevant results found**:

```go
func (s *Service) Search(ctx context.Context, req *SearchRequest) ([]Result, error) {
    results := []Result{}
    targetCount := req.Limit  // e.g., 10

    // Tier 1: Project
    results = append(results, s.searchDB(projectDB, req.Query)...)
    if len(results) >= targetCount {
        return results[:targetCount], nil  // STOP: Enough results
    }

    // Tier 2: Team (only if needed)
    teamResults := s.searchDB(teamDB, req.Query)
    results = append(results, teamResults...)
    if len(results) >= targetCount {
        return results[:targetCount], nil  // STOP
    }

    // Tier 3: Org (only if still need more)
    orgResults := s.searchDB(orgDB, req.Query)
    results = append(results, orgResults...)

    return results[:min(len(results), targetCount)], nil
}
```

**Context Savings**:
- If project has enough results: Skip team/org search entirely
- Avoid loading irrelevant scopes
- Minimize database queries

**Example**:
```
Search for "authentication error" (limit: 10):
  Project DB: Found 12 results → Return 10, STOP
  Context: 2,400 tokens
  Saved: Didn't search team/org (avoided 5,000 tokens)
```

---

### Strategy 2: Scope-Based Truncation

**Truncate lower-priority results more aggressively**:

```go
type ScopeConfig struct {
    MaxResults   int
    MaxTokens    int
    MinRelevance float64
}

var scopeLimits = map[string]ScopeConfig{
    "project": {
        MaxResults:   20,   // Allow more project results
        MaxTokens:    6000,
        MinRelevance: 0.3,  // Lower threshold (show more)
    },
    "team": {
        MaxResults:   10,   // Fewer team results
        MaxTokens:    3000,
        MinRelevance: 0.5,  // Medium threshold
    },
    "org": {
        MaxResults:   5,    // Fewest org results
        MaxTokens:    1500,
        MinRelevance: 0.7,  // High threshold (only best matches)
    },
}
```

**Context Savings**:
- Project results (most relevant): Less truncation
- Org results (less relevant): Aggressive truncation
- Balanced relevance vs context size

---

### Strategy 3: Deduplication

**Remove duplicate results across scopes**:

```go
func deduplicateResults(results []Result) []Result {
    seen := make(map[string]bool)
    unique := []Result{}

    for _, r := range results {
        // Hash based on error message + solution
        hash := hashContent(r.ErrorMessage, r.Solution)

        if !seen[hash] {
            seen[hash] = true
            unique = append(unique, r)
        }
    }

    return unique
}
```

**Context Savings**:
```
Before deduplication:
  Project: "PostgreSQL connection refused" (240 tokens)
  Team: "PostgreSQL connection refused" (240 tokens) ← Duplicate
  Org: "Database connection best practices" (240 tokens)
  Total: 720 tokens

After deduplication:
  Project: "PostgreSQL connection refused" (240 tokens)
  Org: "Database connection best practices" (240 tokens)
  Total: 480 tokens
  Saved: 240 tokens (33%)
```

---

### Strategy 4: Hierarchical Summarization

**Summarize lower-priority results**:

```go
type Result struct {
    ID           string
    ErrorMessage string
    Solution     string  // Full solution (300 tokens)
    Summary      string  // Short summary (50 tokens)
    Scope        string  // "project", "team", "org"
}

func formatResults(results []Result) string {
    output := ""

    for _, r := range results {
        if r.Scope == "project" {
            // Project: Full detail
            output += fmt.Sprintf("Error: %s\nSolution: %s\n", r.ErrorMessage, r.Solution)
        } else if r.Scope == "team" {
            // Team: Summary only
            output += fmt.Sprintf("Error: %s\nSummary: %s\n", r.ErrorMessage, r.Summary)
        } else {
            // Org: Title only
            output += fmt.Sprintf("- %s\n", r.ErrorMessage)
        }
    }

    return output
}
```

**Context Savings**:
```
10 Results:
  3 project (full): 3 × 300 = 900 tokens
  4 team (summary): 4 × 50 = 200 tokens
  3 org (title): 3 × 20 = 60 tokens
  Total: 1,160 tokens

vs All Full:
  10 × 300 = 3,000 tokens

Saved: 1,840 tokens (61% reduction)
```

---

## Estimated Context Usage by Operation

### Checkpoint Save

**v2.0** (shared database):
```
Operation: Save checkpoint
Data sent:
  - Summary: 100 tokens
  - Description: 500 tokens
  - Context: 200 tokens
  - Metadata: 50 tokens
Vector: 1536 dimensions (not counted in context)
Total: 850 tokens

Search existing: ~2,000 tokens (check for duplicates)
Total: ~2,850 tokens
```

**v2.1** (project-scoped):
```
Same as v2.0: 850 tokens (unchanged)
Search: ~500 tokens (project-only, smaller search space)
Total: ~1,350 tokens
Improvement: 53% reduction
```

---

### Remediation Search

**v2.0** (shared, no filtering):
```
Query: "authentication error"
Results: 50 mixed (all projects)
Context per result: ~240 tokens
Total: 12,000 tokens
Relevance: 20% (10/50 relevant)
```

**v2.1** (4-tier search):
```
Query: "authentication error"
Results:
  - Project: 5 results × 240 tokens = 1,200 tokens
  - Team: 3 results × 240 tokens = 720 tokens
  - Org: 2 results × 240 tokens = 480 tokens
Total: 2,400 tokens
Relevance: 90% (9/10 relevant)
Improvement: 5x reduction + better relevance
```

**v2.1 with optimizations**:
```
Early termination (project had enough):
  - Project: 10 results × 240 tokens = 2,400 tokens
  - Team: (skipped)
  - Org: (skipped)
Total: 2,400 tokens

Hierarchical summarization:
  - Project: 5 × 300 tokens = 1,500 tokens (full)
  - Team: 3 × 50 tokens = 150 tokens (summary)
  - Org: 2 × 20 tokens = 40 tokens (title)
Total: 1,690 tokens
Improvement: 7x reduction
```

---

### Skill Application

**v2.0** (shared database):
```
Query: "deployment automation"
Results: 30 skills (all projects, all teams)
Context: 30 × 400 tokens = 12,000 tokens
Applicable: 5 skills (17%)
Wasted: 83%
```

**v2.1** (scoped):
```
Query: "deployment automation"
Results:
  - Team: 8 skills (backend team's deployment)
  - Org: 3 skills (org infrastructure)
Total: 11 × 400 tokens = 4,400 tokens
Applicable: 9 skills (82%)
Improvement: 2.7x reduction + better relevance
```

---

## Context Budget Analysis

### Claude 3.5 Sonnet Context Window

**Total**: 200,000 tokens
**Recommended Usage**:
- Code context: 120,000 tokens (60%)
- Tool results: 40,000 tokens (20%)
- Conversation: 30,000 tokens (15%)
- Buffer: 10,000 tokens (5%)

### Contextd's Context Allocation

**v2.0** (inefficient):
```
Checkpoint search: 2,000 tokens
Remediation search: 12,000 tokens
Skill search: 12,000 tokens
Total: 26,000 tokens (65% of tool budget)
Remaining: 14,000 tokens for actual code
Result: Frequent context overflow
```

**v2.1** (optimized):
```
Checkpoint search: 500 tokens
Remediation search: 2,400 tokens
Skill search: 4,400 tokens
Total: 7,300 tokens (18% of tool budget)
Remaining: 32,700 tokens for actual code
Result: Ample context for code + tools
Improvement: 3.6x more space for code
```

---

## Measuring Context Efficiency

### Metrics to Track

```go
type ContextMetrics struct {
    // Input
    QueryTokens         int     // Tokens in search query

    // Output
    ResultsReturned     int     // Number of results
    TotalTokens         int     // Total context used
    RelevantResults     int     // User-marked as relevant

    // Efficiency
    TokensPerResult     float64 // TotalTokens / ResultsReturned
    RelevanceRatio      float64 // RelevantResults / ResultsReturned
    ContextEfficiency   float64 // RelevantResults / TotalTokens

    // Scopes
    ProjectResults      int
    TeamResults         int
    OrgResults          int
    ScopesSearched      int     // How many tiers searched
}
```

**Target Metrics** (v2.1):
- Relevance ratio: >80% (vs 20% in v2.0)
- Context efficiency: >0.5 relevant results per 1K tokens
- Scopes searched: <3 on average (early termination working)

---

## Configuration for Context Optimization

```yaml
# .contextd/context-optimization.yaml
search:
  # Early termination
  early_termination: true
  target_results: 10

  # Per-scope limits
  project:
    max_results: 20
    max_tokens: 6000
    min_relevance: 0.3

  team:
    max_results: 10
    max_tokens: 3000
    min_relevance: 0.5

  org:
    max_results: 5
    max_tokens: 1500
    min_relevance: 0.7

  # Deduplication
  deduplication: true
  similarity_threshold: 0.9

  # Summarization
  hierarchical_summary: true
  project_format: full      # Full solution
  team_format: summary      # Summary only
  org_format: title         # Title only
```

---

## Expected Context Savings (Overall)

### Typical Session (10 searches)

**v2.0** (shared database):
```
10 searches × 12,000 tokens avg = 120,000 tokens
Result: Exceeds tool budget, context overflow
```

**v2.1** (scoped + optimized):
```
10 searches × 2,400 tokens avg = 24,000 tokens
Result: 60% of tool budget, plenty of room
Improvement: 5x reduction
```

### Real-World Impact

**Scenario**: Developer debugging error
```
v2.0 workflow:
  1. Search remediations: 12,000 tokens
  2. Context overflow warning
  3. /compact or /clear needed
  4. Lost context from earlier conversation
  5. Search again: 12,000 tokens
  Total: 24,000 tokens + lost context

v2.1 workflow:
  1. Search remediations: 2,400 tokens
  2. Search skills: 1,500 tokens
  3. Search troubleshooting: 1,800 tokens
  4. Still have 34,300 tokens left for code
  Total: 5,700 tokens, no context overflow
```

**Result**:
- 4.2x more efficient
- No context overflow
- No need to clear conversation
- Continuous conversation flow

---

## Monitoring & Optimization

### Dashboard Metrics

```
Context Efficiency Dashboard:
  - Avg tokens per search: 2,400 (target: <3,000)
  - Relevance ratio: 85% (target: >80%)
  - Context overflow rate: 0% (target: <5%)
  - Scopes per search: 1.8 (target: <3)
  - Deduplication rate: 15% (shows savings)
```

### Alerts

```yaml
alerts:
  - name: ContextBudgetExceeded
    condition: tool_context > 40000
    severity: warning
    message: "Tool context usage high, consider optimization"

  - name: LowRelevance
    condition: relevance_ratio < 0.5
    severity: warning
    message: "Low relevance ratio, check search scope"

  - name: TooManyScopes
    condition: avg_scopes_searched > 3
    severity: info
    message: "Early termination not effective"
```

---

## Summary

### Context Usage Improvements (v2.0 → v2.1)

| Operation | v2.0 Tokens | v2.1 Tokens | Improvement |
|-----------|-------------|-------------|-------------|
| Checkpoint search | 2,000 | 500 | 4x reduction |
| Remediation search | 12,000 | 2,400 | 5x reduction |
| Skill search | 12,000 | 4,400 | 2.7x reduction |
| **Total session** | **120,000** | **24,000** | **5x reduction** |

### Key Optimizations

1. **Scope-based search**: Only search relevant databases
2. **Early termination**: Stop when enough results
3. **Deduplication**: Remove duplicate results
4. **Hierarchical summarization**: Full detail for project, summaries for team/org
5. **Relevance filtering**: Higher thresholds for lower-priority scopes

### Success Criteria

- ✅ 5x context reduction for typical session
- ✅ >80% relevance ratio (vs 20% in v2.0)
- ✅ Zero context overflow in normal usage
- ✅ Ample room for code + tools (within 200K limit)

---

**Critical for v2.1**: Implement context tracking and optimization from day one. Context efficiency is contextd's primary value proposition.
