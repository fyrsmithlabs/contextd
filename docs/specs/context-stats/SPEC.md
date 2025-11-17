# Context Usage Statistics Specification

**Priority**: 0 (Highest - Proves project impact)
**Version**: 1.0.0
**Status**: Proposed for v2.1

---

## Purpose

**Track and prove contextd's impact on context optimization** - the PRIMARY value proposition.

Without metrics, we can't demonstrate:
- How much context contextd saves vs manual copy-paste
- Whether multi-tenant scoping actually improves relevance
- ROI for users (token costs saved, time saved)
- Whether the architecture is achieving its goals

**This is critical for proving the project's value.**

---

## Goals

1. **Prove Impact**: Quantify context savings (v2.0 vs v2.1 vs manual)
2. **Validate Architecture**: Verify multi-tenant scoping improves relevance
3. **Optimize Performance**: Identify inefficient operations
4. **ROI Calculation**: Show token cost savings over time
5. **User Visibility**: `ctxd stats` command shows impact immediately

---

## Metrics to Track

### Search Operation Metrics

**Per-Search**:
```go
type SearchMetrics struct {
    // Identity
    ID              string    `json:"id"`               // Unique search ID
    Timestamp       time.Time `json:"timestamp"`        // When search occurred
    ProjectPath     string    `json:"project_path"`     // Which project
    Team            string    `json:"team"`             // Which team
    Organization    string    `json:"organization"`     // Which org

    // Query
    QueryText       string    `json:"query_text"`       // Search query
    QueryTokens     int       `json:"query_tokens"`     // Tokens in query

    // Results
    TotalResults    int       `json:"total_results"`    // Total returned
    ProjectResults  int       `json:"project_results"`  // From project scope
    TeamResults     int       `json:"team_results"`     // From team scope
    OrgResults      int       `json:"org_results"`      // From org scope
    PublicResults   int       `json:"public_results"`   // From public scope

    // Context Usage
    TotalTokens     int       `json:"total_tokens"`     // Total context sent
    ProjectTokens   int       `json:"project_tokens"`   // Tokens from project
    TeamTokens      int       `json:"team_tokens"`      // Tokens from team
    OrgTokens       int       `json:"org_tokens"`       // Tokens from org

    // Efficiency
    ScopesSearched  int       `json:"scopes_searched"`  // How many tiers
    EarlyTerminated bool      `json:"early_terminated"` // Stopped early?
    DedupedResults  int       `json:"deduped_results"`  // Removed duplicates

    // Performance
    DurationMs      int64     `json:"duration_ms"`      // Query duration

    // User Feedback (optional, future)
    RelevantCount   *int      `json:"relevant_count"`   // User marked relevant
    UsedResult      *bool     `json:"used_result"`      // Applied a result?
}
```

### Aggregate Metrics

**Daily Summary**:
```go
type DailyStats struct {
    Date            string  `json:"date"`              // YYYY-MM-DD

    // Volume
    TotalSearches   int     `json:"total_searches"`
    TotalCheckpoints int    `json:"total_checkpoints"`
    TotalRemediations int   `json:"total_remediations"`

    // Context Usage
    TotalTokensUsed int64   `json:"total_tokens_used"`
    AvgTokensPerSearch int  `json:"avg_tokens_per_search"`

    // Baseline Comparison (estimated)
    EstimatedV20Tokens int64 `json:"estimated_v20_tokens"` // 5x higher
    EstimatedManualTokens int64 `json:"estimated_manual_tokens"` // 10x higher
    TokensSaved     int64   `json:"tokens_saved"`

    // Efficiency
    AvgRelevanceRatio float64 `json:"avg_relevance_ratio"`
    AvgScopesSearched float64 `json:"avg_scopes_searched"`
    EarlyTerminationRate float64 `json:"early_termination_rate"`

    // Cost Savings (Claude 3.5 Sonnet pricing)
    EstimatedCostSavings float64 `json:"estimated_cost_savings"` // USD
}
```

---

## Data Collection

### Service Layer Instrumentation

**Checkpoint Service**:
```go
// pkg/checkpoint/service.go
func (s *Service) Search(ctx context.Context, req *SearchRequest) ([]*Checkpoint, error) {
    startTime := time.Now()

    // Perform search
    results, err := s.search(ctx, req)

    // Record metrics
    s.metrics.RecordSearch(ctx, SearchMetrics{
        Timestamp:    startTime,
        QueryText:    req.Query,
        QueryTokens:  estimateTokens(req.Query),
        TotalResults: len(results),
        TotalTokens:  estimateTotalTokens(results),
        DurationMs:   time.Since(startTime).Milliseconds(),
    })

    return results, err
}
```

**Remediation Service** (same pattern):
```go
// pkg/remediation/service.go
func (s *Service) FindSimilarErrors(ctx context.Context, req *SearchRequest) ([]SimilarError, error) {
    startTime := time.Now()
    metrics := SearchMetrics{
        Timestamp:   startTime,
        QueryText:   req.ErrorMessage,
        QueryTokens: estimateTokens(req.ErrorMessage),
        ProjectPath: req.ProjectPath,
    }

    // Tier 1: Project
    projectResults := s.searchProject(ctx, req)
    metrics.ProjectResults = len(projectResults)
    metrics.ProjectTokens = estimateTotalTokens(projectResults)

    if len(projectResults) >= req.Limit {
        metrics.EarlyTerminated = true
        metrics.ScopesSearched = 1
        s.metrics.RecordSearch(ctx, metrics)
        return projectResults[:req.Limit], nil
    }

    // Tier 2: Team
    teamResults := s.searchTeam(ctx, req)
    metrics.TeamResults = len(teamResults)
    metrics.TeamTokens = estimateTotalTokens(teamResults)
    metrics.ScopesSearched = 2

    // ... continue for org/public

    s.metrics.RecordSearch(ctx, metrics)
    return results, nil
}
```

### Token Estimation

**Accurate Token Counting**:
```go
// pkg/analytics/tokens.go
func EstimateTokens(text string) int {
    // Use tiktoken for accurate counting (GPT tokenizer)
    // Fallback: char count / 4 (rough estimate)

    // For checkpoint summary
    tokens := len(text) / 4

    // Add overhead for JSON structure
    tokens += 20

    return tokens
}

func EstimateCheckpointTokens(cp *Checkpoint) int {
    total := 0
    total += EstimateTokens(cp.Summary)
    total += EstimateTokens(cp.Description)
    total += EstimateTokens(fmt.Sprintf("%v", cp.Context))
    total += 50 // JSON overhead
    return total
}
```

---

## Storage

### Metrics Database

**Option 1: Separate SQLite Database** (Recommended)
```
~/.local/share/contextd/metrics.db

Tables:
  - search_metrics (detailed per-search)
  - daily_stats (aggregated daily)
  - weekly_stats (aggregated weekly)
  - monthly_stats (aggregated monthly)
```

**Schema**:
```sql
CREATE TABLE search_metrics (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    project_path TEXT,
    team TEXT,
    organization TEXT,
    query_text TEXT,
    query_tokens INTEGER,
    total_results INTEGER,
    project_results INTEGER,
    team_results INTEGER,
    org_results INTEGER,
    total_tokens INTEGER,
    project_tokens INTEGER,
    team_tokens INTEGER,
    org_tokens INTEGER,
    scopes_searched INTEGER,
    early_terminated BOOLEAN,
    deduped_results INTEGER,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_timestamp ON search_metrics(timestamp);
CREATE INDEX idx_project ON search_metrics(project_path);
CREATE INDEX idx_team ON search_metrics(team);

CREATE TABLE daily_stats (
    date TEXT PRIMARY KEY,
    total_searches INTEGER,
    total_tokens_used INTEGER,
    avg_tokens_per_search INTEGER,
    estimated_v20_tokens INTEGER,
    tokens_saved INTEGER,
    avg_relevance_ratio REAL,
    avg_scopes_searched REAL,
    early_termination_rate REAL,
    estimated_cost_savings REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

- Store as metadata in vector database
- Pro: No extra database
- Con: Harder to query, slower aggregation

**Decision**: Use SQLite (simple, fast, SQL queries for aggregation)

---

## API Endpoints

### Get Stats

**Endpoint**: `GET /api/v1/stats`

**Query Parameters**:
- `period`: `hour`, `day`, `week`, `month`, `all` (default: `day`)
- `start`: ISO 8601 date (default: 24h ago)
- `end`: ISO 8601 date (default: now)
- `project_path`: Filter by project (optional)
- `team`: Filter by team (optional)

**Response**:
```json
{
  "period": "day",
  "start": "2025-01-06T00:00:00Z",
  "end": "2025-01-07T00:00:00Z",
  "summary": {
    "total_searches": 47,
    "total_tokens_used": 111860,
    "avg_tokens_per_search": 2380,
    "estimated_v20_tokens": 564000,
    "tokens_saved": 452140,
    "savings_percentage": 80.2,
    "avg_relevance_ratio": 0.87,
    "early_termination_rate": 0.68
  },
  "by_scope": {
    "project": {
      "results": 124,
      "tokens": 29760,
      "percentage": 65
    },
    "team": {
      "results": 42,
      "tokens": 10080,
      "percentage": 22
    },
    "org": {
      "results": 25,
      "tokens": 6000,
      "percentage": 13
    }
  },
  "cost_savings": {
    "currency": "USD",
    "input_tokens_saved": 452140,
    "cost_per_mtok": 3.00,
    "total_savings": 1.36
  }
}
```

### Get Recent Searches

**Endpoint**: `GET /api/v1/stats/searches`

**Query Parameters**:
- `limit`: Max results (default: 50, max: 1000)
- `offset`: Pagination offset
- `project_path`: Filter by project

**Response**:
```json
{
  "searches": [
    {
      "id": "search_123",
      "timestamp": "2025-01-07T10:30:00Z",
      "query_text": "database connection error",
      "total_results": 10,
      "total_tokens": 2400,
      "scopes_searched": 2,
      "early_terminated": true,
      "duration_ms": 45
    }
  ],
  "total": 1234,
  "limit": 50,
  "offset": 0
}
```

---

## CLI Command: `ctxd stats`

### Basic Usage

```bash
# View today's stats
ctxd stats

# View weekly stats
ctxd stats --period week

# View specific date range
ctxd stats --start 2025-01-01 --end 2025-01-07

# Filter by project
ctxd stats --project /home/user/contextd

# Export to JSON
ctxd stats --format json > stats.json
```

### Output Format

```
Context Usage Statistics
Period: 2025-01-07 (Last 24 hours)

Search Operations:
  Total searches:             47
  Avg tokens per search:      2,380 (target: <3,000) ✓
  Total tokens used:          111,860

Baseline Comparison:
  v2.0 (shared DB):           564,000 tokens (estimated)
  Manual copy-paste:          1,128,000 tokens (estimated)
  Savings vs v2.0:            452,140 tokens (80% reduction) 🎉
  Savings vs manual:          1,016,140 tokens (90% reduction) 🎉

Relevance & Efficiency:
  Avg relevance ratio:        87% (target: >80%) ✓
  Early termination rate:     68%
  Avg scopes searched:        1.9 (target: <3) ✓
  Deduplication savings:      18%

Results by Scope:
  Project (Tier 1):           124 results (65%) - 29,760 tokens
  Team (Tier 2):              42 results (22%) - 10,080 tokens
  Org (Tier 3):               25 results (13%) - 6,000 tokens
  Public (Tier 4):            0 results (0%) - 0 tokens

Session Health:
  Context overflow events:    0 (target: <5%) ✓
  Peak context usage:         28,900 tokens (85% budget remaining)

Cost Savings (Claude 3.5 Sonnet @ $3/MTok input):
  Tokens saved vs v2.0:       452,140
  Estimated savings:          $1.36 today
  Projected monthly savings:  ~$40.80

Performance:
  Avg search latency (p50):   42ms
  Avg search latency (p95):   89ms (target: <100ms) ✓
  Fastest search:             12ms
  Slowest search:             156ms
```

### Detailed View

```bash
ctxd stats --detailed
```

**Additional output**:
- Top 10 most expensive searches
- Top 10 projects by token usage
- Hourly breakdown (chart)
- Scope distribution (pie chart)

### Comparison Mode

```bash
# Compare this week vs last week
ctxd stats --compare week
```

**Output**:
```
Week-over-Week Comparison

This Week (2025-01-01 to 2025-01-07):
  Searches: 312
  Tokens used: 747,360
  Avg per search: 2,395

Last Week (2024-12-25 to 2024-12-31):
  Searches: 289
  Tokens used: 782,120
  Avg per search: 2,706

Change:
  Searches: +8% ▲
  Tokens used: -4% ▼ (better!)
  Avg per search: -11% ▼ (improved efficiency!)

Insight: Your searches are getting more efficient!
Likely due to better scoping or more project-specific queries.
```

---

## Implementation

### Phase 1: Metrics Collection (Week 1-2)

**Tasks**:
- [ ] Create `pkg/analytics/stats.go`
  - SearchMetrics struct
  - DailyStats struct
  - Token estimation functions
- [ ] Create SQLite schema
- [ ] Instrument checkpoint service
- [ ] Instrument remediation service
- [ ] Instrument skills service
- [ ] Unit tests

**Deliverable**: Services collecting metrics to SQLite

---

### Phase 2: Aggregation & API (Week 3)

**Tasks**:
- [ ] Create aggregation logic (daily rollup)
- [ ] Add `/api/v1/stats` endpoint
- [ ] Add `/api/v1/stats/searches` endpoint
- [ ] Baseline estimation (v2.0 vs manual)
- [ ] Cost calculation
- [ ] Integration tests

**Deliverable**: API endpoints returning stats

---

### Phase 3: CLI Command (Week 4)

**Tasks**:
- [ ] Create `cmd/ctxd/stats.go`
- [ ] Implement basic output format
- [ ] Add filtering (--project, --period)
- [ ] Add comparison mode (--compare)
- [ ] Add export formats (--format json/csv)
- [ ] Add charts (optional, ASCII art)
- [ ] E2E tests

**Deliverable**: `ctxd stats` command working

---

### Phase 4: Visualization (Week 5, Optional)

**Tasks**:
- [ ] Create `ctxd stats --dashboard` (live TUI)
- [ ] Real-time charts
- [ ] Export to HTML report
- [ ] Grafana dashboard JSON

**Deliverable**: Visual dashboards

---

## Baseline Estimation

### v2.0 (Shared Database) Estimate

**Assumption**: 5x more tokens due to irrelevant results

```go
func EstimateV20Tokens(actualTokens int, relevanceRatio float64) int {
    // In v2.0, relevance was ~20% (4 relevant out of 20 results)
    // To get same relevant results, would need 5x more total results
    return actualTokens * 5
}
```

**Example**:
```
v2.1 Search: 10 results, 2,400 tokens, 90% relevant (9 results)
v2.0 Estimate: To get 9 relevant results with 20% relevance:
  Need: 9 / 0.20 = 45 total results
  Tokens: 45 * 240 = 10,800 tokens
  Factor: 10,800 / 2,400 = 4.5x
```

### Manual Copy-Paste Estimate

**Assumption**: 10x more tokens (no scoping, full file context)

```go
func EstimateManualTokens(actualTokens int) int {
    // Manual: User copies entire files, not just relevant snippets
    // Typically 10x more context than scoped search
    return actualTokens * 10
}
```

---

## Privacy & Security

**Collected Data**:
- ✅ Query text (user's search queries)
- ✅ Project paths (user's directory structure)
- ✅ Result counts and token usage

**NOT Collected**:
- ❌ Result contents (actual remediation solutions)
- ❌ API keys or tokens
- ❌ User identification
- ❌ Remote telemetry

**Storage**:
- All data stored locally (~/.local/share/contextd/metrics.db)
- User owns all metrics data
- No cloud sync
- Can delete anytime

**Opt-Out**:
```yaml
# .contextd/config.yaml
analytics:
  enabled: false  # Disable all metrics collection
```

---

## Success Criteria

**Phase 1** (Metrics Collection):
- [ ] All services instrumented
- [ ] SQLite database created
- [ ] Metrics persisted correctly
- [ ] Token estimation accurate (within 10%)

**Phase 2** (API):
- [ ] API endpoints return correct data
- [ ] Aggregation works (daily/weekly/monthly)
- [ ] Baseline estimation reasonable
- [ ] Performance <50ms for stats query

**Phase 3** (CLI):
- [ ] `ctxd stats` shows useful output
- [ ] Filtering works
- [ ]