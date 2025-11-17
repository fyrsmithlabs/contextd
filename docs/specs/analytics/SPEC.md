# Analytics System Specification

## Overview

The Analytics System provides comprehensive usage tracking and metrics collection for contextd, enabling users to measure context optimization effectiveness, feature adoption, performance characteristics, and business impact. The system tracks token reduction metrics, feature usage patterns, performance data, and generates actionable insights about context efficiency.

**Primary Goal**: Quantify context optimization value and demonstrate ROI through measurable metrics.

**Key Capabilities**:
- Session tracking with token reduction metrics
- Feature adoption monitoring
- Performance metrics collection
- Business impact calculation
- Time-series analysis with multiple period aggregations
- Project-level filtering and multi-tenant support

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      MCP Layer (API)                         │
│  - analytics_get: Retrieve aggregated metrics                │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                  Analytics Service                           │
│  - Session Management: Start/End session tracking            │
│  - Feature Recording: Track feature usage                    │
│  - Performance Recording: Capture operation metrics          │
│  - Metric Aggregation: Calculate business metrics            │
│  - OpenTelemetry Integration: Emit metrics instruments       │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  Collections:                                                │
│  - session_metrics: Per-session token tracking               │
│  - feature_adoption: Feature usage statistics                │
│  - performance_metrics: Operation performance data           │
│  - claude_md_snapshots: CLAUDE.md size tracking              │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow

**Session Lifecycle**:
```
1. StartSession → Create session_metrics record
2. [Operations occur, features used]
3. RecordFeatureUsage → Update feature_adoption
4. RecordPerformance → Insert performance_metrics
5. EndSession → Update session_metrics with final tokens
6. [Calculate token reduction, update aggregates]
```

**Analytics Retrieval**:
```
1. analytics_get(period, dates, project_path)
2. → Service.GetBusinessMetrics()
3. → Store queries: session_metrics, feature_adoption, performance_metrics
4. → Aggregate calculations: avg/median/max, rate calculations
5. → Return formatted output with insights
```

## Metrics Tracked

### 1. Token Reduction Metrics

**Purpose**: Quantify context optimization effectiveness

| Metric | Description | Calculation | Target |
|--------|-------------|-------------|--------|
| TokensBefore | Tokens in context before optimization | Measured at session start | N/A |
| TokensAfter | Tokens after checkpoint/search | Measured at session end | N/A |
| TokensSaved | Tokens eliminated from context | TokensBefore - TokensAfter | >0 |
| ReductionPct | Percentage reduction | (TokensSaved / TokensBefore) * 100 | >30% |
| AvgTokenReduction | Average reduction across sessions | Sum(ReductionPct) / SessionCount | >40% |
| MedianTokenReduction | Median reduction (robust to outliers) | Median(ReductionPct values) | >35% |
| MaxTokenReduction | Best observed reduction | Max(ReductionPct) | >80% |

**Data Sources**:
- `session_metrics.tokens_before`
- `session_metrics.tokens_after`
- `session_metrics.tokens_saved`
- `session_metrics.reduction_pct`

### 2. Feature Adoption Metrics

**Purpose**: Track which contextd features are most valuable

| Metric | Description | Target |
|--------|-------------|--------|
| Feature Count | Number of times feature used | Increasing trend |
| First Used | When feature first adopted | Early adoption |
| Last Used | Most recent usage | Recent activity |
| Avg Latency | Average operation latency | <500ms |
| Success Rate | Percentage of successful operations | >95% |

**Tracked Features**:
- `checkpoint_save` - Session checkpoint creation
- `checkpoint_search` - Semantic checkpoint search
- `checkpoint_list` - Checkpoint browsing
- `remediation_save` - Error solution storage
- `remediation_search` - Error solution lookup
- `troubleshoot` - AI-powered diagnosis
- `index_repository` - Repository indexing
- `skill_apply` - Skill application

**Data Sources**:
- `feature_adoption.count`
- `feature_adoption.avg_latency_ms`
- `feature_adoption.success_rate`

### 3. Performance Metrics

**Purpose**: Monitor system performance and identify bottlenecks

| Metric | Description | Threshold | Alert |
|--------|-------------|-----------|-------|
| LatencyMs | Operation duration (milliseconds) | <500ms | >2000ms |
| Success | Operation completed successfully | 100% | <95% |
| EmbeddingCacheHit | Embedding retrieved from cache | High | <50% |
| TokensProcessed | Tokens in operation input | N/A | >50K |
| ResultCount | Number of results returned | N/A | N/A |

**Aggregated Performance Metrics**:
- `AvgSearchLatency` - Average checkpoint_search latency
- `AvgCheckpointLatency` - Average checkpoint_save latency
- `CacheHitRate` - Percentage of cache hits
- `OverallSuccessRate` - Success rate across all operations

**Data Sources**:
- `performance_metrics.latency_ms`
- `performance_metrics.success`
- `performance_metrics.embedding_cache_hit`

### 4. Business Impact Metrics

**Purpose**: Demonstrate ROI and business value

| Metric | Description | Calculation | Business Value |
|--------|-------------|-------------|----------------|
| TotalSessions | Number of tracked sessions | Count(sessions) | Usage volume |
| TotalTimeSaved | Estimated minutes saved | Sum(ReductionPct / 10) | Productivity gain |
| EstimatedCostSave | Estimated $ saved on API costs | (AvgTokensSaved / 1K) * $0.015 * Sessions | Cost savings |
| SearchPrecision | Search success rate | SuccessCount / TotalSearches | Quality metric |
| MTTRReduction | Mean time to resolution reduction | (BaselineMTTR - CurrentMTTR) / BaselineMTTR | Efficiency gain |
| ActiveProjects | Unique projects using contextd | Count(distinct project_paths) | Adoption metric |

**Assumptions**:
- 1 minute saved per 10% token reduction
- Claude Sonnet pricing: $0.015 per 1K input tokens
- Search precision measured by result relevance

**Data Sources**:
- `business_metrics` (calculated from sessions + performance)

### 5. CLAUDE.md Size Tracking

**Purpose**: Monitor context configuration file growth

| Metric | Description | Target |
|--------|-------------|--------|
| Size (bytes) | CLAUDE.md file size | <50KB |
| Line Count | Number of lines | <1000 |
| Optimized | Whether file was optimized | After optimization |

**Data Sources**:
- `claude_md_snapshots.size`
- `claude_md_snapshots.line_count`
- `claude_md_snapshots.optimized`

## Time Period Support

### Supported Periods

| Period | Description | Date Range | Use Case |
|--------|-------------|------------|----------|
| `daily` | Last 24 hours | now - 1 day | Spot-checking recent performance |
| `weekly` | Last 7 days | now - 7 days | Default view for regular monitoring |
| `monthly` | Last 30 days | now - 30 days | Trend analysis |
| `all-time` | Since 2020-01-01 | 2020-01-01 to now | Historical overview |

### Custom Date Ranges

Users can override default periods with explicit dates:

```json
{
  "start_date": "2025-10-01",
  "end_date": "2025-11-01"
}
```

**Date Format**: `YYYY-MM-DD` (ISO 8601)

### Period Calculation Logic

```go
switch period {
case "daily":
    startDate = endDate.AddDate(0, 0, -1)
case "weekly":
    startDate = endDate.AddDate(0, 0, -7)
case "monthly":
    startDate = endDate.AddDate(0, -1, 0)
case "all-time":
    startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
default:
    startDate = endDate.AddDate(0, 0, -7) // Default to weekly
}
```

## Project-Level Filtering

### Multi-Tenant Support

The analytics system supports filtering by project path for multi-tenant scenarios:

```json
{
  "project_path": "/home/user/projects/my-project"
}
```

**Behavior**:
- If `project_path` provided: Only metrics for that project
- If `project_path` empty: Aggregate across all projects

**Implementation**:
- `session_metrics.project_path` filter
- `feature_adoption.project_path` filter
- `performance_metrics.project_path` filter

**Use Cases**:
1. **Single Project Analysis**: Focus on one codebase's optimization
2. **Cross-Project Comparison**: Compare metrics between projects
3. **Global Dashboard**: View all projects in aggregate

## API Specifications

### MCP Tool: analytics_get

**Tool Name**: `analytics_get`

**Description**: Get context usage analytics and metrics including token reduction, feature adoption, performance metrics, and business impact

**Input Schema**:

```json
{
  "type": "object",
  "properties": {
    "period": {
      "type": "string",
      "description": "Time period (daily, weekly, monthly, all-time) - default: weekly",
      "enum": ["daily", "weekly", "monthly", "all-time"],
      "default": "weekly"
    },
    "project_path": {
      "type": "string",
      "description": "Filter by project path"
    },
    "start_date": {
      "type": "string",
      "description": "Start date (YYYY-MM-DD) - defaults to period",
      "pattern": "^\\d{4}-\\d{2}-\\d{2}$"
    },
    "end_date": {
      "type": "string",
      "description": "End date (YYYY-MM-DD) - defaults to now",
      "pattern": "^\\d{4}-\\d{2}-\\d{2}$"
    }
  }
}
```

**Output Schema**:

```json
{
  "type": "object",
  "properties": {
    "period": {
      "type": "string",
      "description": "Time period analyzed"
    },
    "start_date": {
      "type": "string",
      "format": "date-time",
      "description": "Analysis start date"
    },
    "end_date": {
      "type": "string",
      "format": "date-time",
      "description": "Analysis end date"
    },
    "total_sessions": {
      "type": "integer",
      "description": "Number of tracked sessions"
    },
    "avg_token_reduction_pct": {
      "type": "number",
      "description": "Average token reduction percentage"
    },
    "total_time_saved_min": {
      "type": "number",
      "description": "Total minutes saved"
    },
    "search_precision": {
      "type": "number",
      "description": "Search success rate (0-1)"
    },
    "estimated_cost_save_usd": {
      "type": "number",
      "description": "Estimated cost savings in USD"
    },
    "top_features": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "feature": {"type": "string"},
          "count": {"type": "integer"},
          "avg_latency_ms": {"type": "number"},
          "success_rate": {"type": "number"}
        }
      },
      "description": "Top 5 most used features"
    },
    "performance": {
      "type": "object",
      "properties": {
        "avg_search_latency_ms": {"type": "number"},
        "avg_checkpoint_latency_ms": {"type": "number"},
        "cache_hit_rate": {"type": "number"},
        "overall_success_rate": {"type": "number"}
      }
    }
  }
}
```

**Example Request**:

```json
{
  "period": "weekly",
  "project_path": "/home/user/projects/contextd"
}
```

**Example Response**:

```json
{
  "period": "weekly",
  "start_date": "2025-10-28T00:00:00Z",
  "end_date": "2025-11-04T00:00:00Z",
  "total_sessions": 25,
  "avg_token_reduction_pct": 45.2,
  "total_time_saved_min": 113.0,
  "search_precision": 0.89,
  "estimated_cost_save_usd": 2.35,
  "top_features": [
    {
      "feature": "checkpoint_search",
      "count": 78,
      "avg_latency_ms": 235.5,
      "success_rate": 0.92
    },
    {
      "feature": "checkpoint_save",
      "count": 45,
      "avg_latency_ms": 180.2,
      "success_rate": 0.98
    },
    {
      "feature": "remediation_search",
      "count": 23,
      "avg_latency_ms": 198.7,
      "success_rate": 0.87
    }
  ],
  "performance": {
    "avg_search_latency_ms": 235.5,
    "avg_checkpoint_latency_ms": 180.2,
    "cache_hit_rate": 0.65,
    "overall_success_rate": 0.94
  }
}
```

**Formatted Output** (displayed to user):

```markdown
# Context Usage Analytics (weekly)

**Period:** 2025-10-28 to 2025-11-04

## Token Optimization
- **Total Sessions:** 25
- **Avg Token Reduction:** 45.2%
- **Estimated Cost Savings:** $2.35

## Time Savings
- **Total Time Saved:** 113.0 minutes
- **Search Precision:** 89.0%

## Top Features
1. **checkpoint_search**: 78 uses (235.5 ms avg, 92.0% success)
2. **checkpoint_save**: 45 uses (180.2 ms avg, 98.0% success)
3. **remediation_search**: 23 uses (198.7 ms avg, 87.0% success)

## Performance
- **Avg Search Latency:** 235.5 ms
- **Avg Checkpoint Latency:** 180.2 ms
- **Cache Hit Rate:** 65.0%
- **Overall Success Rate:** 94.0%

---
*Generated at 2025-11-04 12:00:00*
```

## Internal Service API

### Analytics Service Interface

```go
type Service struct {
    store  Store
    tracer trace.Tracer
    meter  metric.Meter

    // OpenTelemetry metric instruments
    sessionCounter       metric.Int64Counter
    tokenReductionHist   metric.Float64Histogram
    featureUsageCounter  metric.Int64Counter
    operationLatencyHist metric.Float64Histogram
    cacheHitCounter      metric.Int64Counter
}
```

### Key Methods

#### Session Management

```go
// StartSession creates a new session tracking entry
func (s *Service) StartSession(
    ctx context.Context,
    projectPath string,
    tokensBefore int,
) (*SessionMetrics, error)

// EndSession updates a session with final metrics
func (s *Service) EndSession(
    ctx context.Context,
    sessionID string,
    tokensAfter int,
) error
```

**Behavior**:
- Generates unique session ID
- Records CLAUDE.md size if present
- Initializes empty feature/metadata maps
- Records OpenTelemetry metrics

#### Feature Tracking

```go
// RecordFeatureUsage tracks usage of a contextd feature
func (s *Service) RecordFeatureUsage(
    ctx context.Context,
    feature string,
    projectPath string,
    latencyMs float64,
    success bool,
) error
```

**Behavior**:
- Updates running average of latency
- Updates success rate calculation
- Updates first/last used timestamps
- Emits OpenTelemetry counter

#### Performance Recording

```go
// RecordPerformance tracks performance metrics for an operation
func (s *Service) RecordPerformance(
    ctx context.Context,
    operation string,
    latencyMs float64,
    success bool,
    cacheHit bool,
    tokensProcessed int,
    resultCount int,
    projectPath string,
) error
```

**Behavior**:
- Inserts timestamped performance record
- Emits OpenTelemetry histogram for latency
- Tracks cache hit rate

#### CLAUDE.md Tracking

```go
// TrackClaudeMDSize records a snapshot of CLAUDE.md size
func (s *Service) TrackClaudeMDSize(
    ctx context.Context,
    projectPath string,
    optimized bool,
) error
```

**Behavior**:
- Reads CLAUDE.md from project path
- Counts lines and bytes
- Creates snapshot record
- Skips if file doesn't exist

#### Metric Retrieval

```go
// GetBusinessMetrics retrieves business impact metrics for a period
func (s *Service) GetBusinessMetrics(
    ctx context.Context,
    period string,
    start, end time.Time,
) (*BusinessMetrics, error)

// GetAggregatedMetrics retrieves aggregated analytics for a period
func (s *Service) GetAggregatedMetrics(
    ctx context.Context,
    period string,
    start, end time.Time,
) (*AggregatedMetrics, error)

// GetFeatureAdoption retrieves feature adoption metrics
func (s *Service) GetFeatureAdoption(
    ctx context.Context,
    projectPath string,
) ([]FeatureAdoption, error)
```

## Data Models

### SessionMetrics

```go
type SessionMetrics struct {
    ID            string            `json:"id"`
    ProjectPath   string            `json:"project_path"`
    StartTime     time.Time         `json:"start_time"`
    EndTime       *time.Time        `json:"end_time,omitempty"`
    TokensBefore  int               `json:"tokens_before"`
    TokensAfter   int               `json:"tokens_after"`
    TokensSaved   int               `json:"tokens_saved"`
    ReductionPct  float64           `json:"reduction_pct"`
    ClaudeMDSize  int               `json:"claude_md_size"`
    Features      map[string]int    `json:"features"`
    Metadata      map[string]string `json:"metadata"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
}
```


**Indexes**: `start_time` (scalar index for time-based queries)

### FeatureAdoption

```go
type FeatureAdoption struct {
    Feature       string    `json:"feature"`
    Count         int       `json:"count"`
    LastUsed      time.Time `json:"last_used"`
    FirstUsed     time.Time `json:"first_used"`
    ProjectPath   string    `json:"project_path"`
    AvgLatencyMs  float64   `json:"avg_latency_ms"`
    SuccessRate   float64   `json:"success_rate"`
}
```


**Update Strategy**: Upsert on (feature, project_path)

**Calculation**:
- `AvgLatencyMs`: Running average = (PrevAvg * (N-1) + NewLatency) / N
- `SuccessRate`: Running rate = (PrevRate * (N-1) + NewSuccess) / N

### PerformanceMetrics

```go
type PerformanceMetrics struct {
    Timestamp         time.Time `json:"timestamp"`
    Operation         string    `json:"operation"`
    LatencyMs         float64   `json:"latency_ms"`
    Success           bool      `json:"success"`
    EmbeddingCacheHit bool      `json:"embedding_cache_hit"`
    TokensProcessed   int       `json:"tokens_processed"`
    ResultCount       int       `json:"result_count"`
    ProjectPath       string    `json:"project_path"`
}
```


**Indexes**: `timestamp` (scalar index)

**Retention**: Consider retention policy (e.g., 90 days) for high-volume metrics

### BusinessMetrics

```go
type BusinessMetrics struct {
    Period            string    `json:"period"`
    StartDate         time.Time `json:"start_date"`
    EndDate           time.Time `json:"end_date"`
    TotalSessions     int       `json:"total_sessions"`
    AvgTokenReduction float64   `json:"avg_token_reduction"`
    TotalTimeSavedMin float64   `json:"total_time_saved_min"`
    MTTRReductionPct  float64   `json:"mttr_reduction_pct"`
    SearchPrecision   float64   `json:"search_precision"`
    EstimatedCostSave float64   `json:"estimated_cost_save"`
    ActiveProjects    int       `json:"active_projects"`
}
```

**Computed On-Demand**: Not stored, calculated from session + performance metrics

### AggregatedMetrics

```go
type AggregatedMetrics struct {
    Period                string    `json:"period"`
    StartDate             time.Time `json:"start_date"`
    EndDate               time.Time `json:"end_date"`
    TotalSessions         int       `json:"total_sessions"`
    AvgTokenReduction     float64   `json:"avg_token_reduction"`
    MedianTokenReduction  float64   `json:"median_token_reduction"`
    MaxTokenReduction     float64   `json:"max_token_reduction"`
    AvgSessionDuration    float64   `json:"avg_session_duration_min"`
    TopFeatures           []struct {
        Feature string `json:"feature"`
        Count   int    `json:"count"`
    } `json:"top_features"`
    PerformanceSummary struct {
        AvgSearchLatency     float64 `json:"avg_search_latency_ms"`
        AvgCheckpointLatency float64 `json:"avg_checkpoint_latency_ms"`
        CacheHitRate         float64 `json:"cache_hit_rate"`
        OverallSuccessRate   float64 `json:"overall_success_rate"`
    } `json:"performance_summary"`
}
```

**Computed On-Demand**: Aggregates multiple data sources

### ClaudeMDSnapshot

```go
type ClaudeMDSnapshot struct {
    ID          string    `json:"id"`
    ProjectPath string    `json:"project_path"`
    Size        int       `json:"size"`
    LineCount   int       `json:"line_count"`
    Optimized   bool      `json:"optimized"`
    Timestamp   time.Time `json:"timestamp"`
}
```


**Indexes**: `timestamp` (scalar index)

## Aggregation Algorithms

### Token Reduction Calculation

```go
// Per session
reductionPct = (tokensSaved / tokensBefore) * 100

// Average across sessions
avgReduction = sum(reductionPct) / sessionCount

// Median (robust to outliers)
sort(reductionPcts)
medianReduction = reductionPcts[len/2]
```

### Time Saved Estimation

```go
// Assumption: 1 minute saved per 10% token reduction
timeSavedMin = reductionPct / 10.0

// Total across sessions
totalTimeSaved = sum(timeSavedMin)
```

### Cost Savings Estimation

```go
// Assumption: Claude Sonnet pricing ($0.015 per 1K input tokens)
avgTokensSaved = sum(tokensSaved) / sessionCount
costSave = (avgTokensSaved / 1000) * 0.015 * sessionCount
```

### Running Averages (Feature Adoption)

```go
// Latency running average
newAvg = (prevAvg * (count - 1) + newLatency) / count

// Success rate running average
if success {
    newRate = (prevRate * (count - 1) + 1.0) / count
} else {
    newRate = (prevRate * (count - 1)) / count
}
```

### Performance Aggregations

```go
// Average latency by operation
avgLatency[op] = sum(latency where operation=op) / count(op)

// Cache hit rate
cacheHitRate = count(cacheHit=true) / count(*)

// Overall success rate
successRate = count(success=true) / count(*)
```

## Performance Characteristics

### Query Performance

| Operation | Target Latency | Notes |
|-----------|----------------|-------|
| StartSession | <50ms | Single insert |
| EndSession | <100ms | Update + calculation |
| RecordFeatureUsage | <100ms | Upsert with aggregation |
| RecordPerformance | <50ms | Single insert |
| GetBusinessMetrics | <500ms | Aggregates multiple collections |
| GetAggregatedMetrics | <1000ms | Complex aggregations |

### Storage Characteristics

**Storage Growth Rate** (estimated):

| Collection | Records/Day | Size/Record | Growth/Day |
|------------|-------------|-------------|------------|
| session_metrics | 50 | ~1KB | 50KB |
| feature_adoption | 10 updates | ~500B | 5KB |
| performance_metrics | 500 | ~500B | 250KB |
| claude_md_snapshots | 10 | ~200B | 2KB |

**Total Daily Growth**: ~307KB/day = ~9MB/month = ~110MB/year

**Scaling Considerations**:
- Performance metrics have highest volume
- Consider retention policy for performance_metrics (90-180 days)
- Session metrics should be retained indefinitely for historical analysis

### OpenTelemetry Overhead

**Metric Instruments** (5 total):
- `contextd.analytics.sessions` (Counter)
- `contextd.analytics.token_reduction_pct` (Histogram)
- `contextd.analytics.feature_usage` (Counter)
- `contextd.analytics.operation_latency_ms` (Histogram)
- `contextd.analytics.cache_hits` (Counter)

**Export Overhead**: <10ms per operation (async export)

## Error Handling

### Error Scenarios

| Scenario | Error Type | Handling Strategy |
|----------|-----------|-------------------|
| Analytics service unavailable | ValidationError | Return empty metrics, log warning |
| Invalid date format | ValidationError | Return error with format example |
| Session not found | NotFoundError | Return error, check session ID |
| Division by zero (no sessions) | N/A | Return zero values, handle gracefully |
| Malformed JSON in features | InternalError | Log error, skip record |
| CLAUDE.md not found | N/A | Skip tracking, no error |

### Error Codes

```go
const (
    ErrAnalyticsServiceUnavailable = "ANALYTICS_SERVICE_UNAVAILABLE"
    ErrInvalidDateFormat           = "INVALID_DATE_FORMAT"
    ErrSessionNotFound             = "SESSION_NOT_FOUND"
    ErrInvalidPeriod               = "INVALID_PERIOD"
)
```

### Graceful Degradation

**Priority Levels**:
1. **Critical**: Session start/end (must succeed)
2. **High**: Feature usage recording (log errors, continue)
3. **Medium**: Performance metrics (best-effort)
4. **Low**: CLAUDE.md snapshots (skip on error)

**Fallback Behavior**:
- If feature adoption query fails: Return empty array, continue
- If performance query fails: Return default performance summary
- If aggregation fails: Return partial results with warning

## Security Considerations

### Data Privacy

**Sensitive Data Handling**:
- Project paths stored as-is (required for filtering)
- No code content stored in analytics
- No user credentials stored
- Metadata values should not contain secrets

**Redaction Requirements**:
- Do NOT store error messages with sensitive data in metadata
- Do NOT store API keys or tokens in session metadata
- Use security.Redact() for any user-supplied strings

### Multi-Tenant Isolation

**Project Path Filtering**:
- ALWAYS filter by project_path in multi-tenant scenarios
- Validate project_path ownership (if auth layer exists)
- Prevent cross-project data leakage

**Query Safety**:
- Validate all user inputs (period, dates, project_path)
- Sanitize project paths to prevent injection

### Access Control

**Future Considerations** (when adding authentication):
- Only project owners can view their analytics
- Global analytics (all projects) requires admin role
- Rate limiting on analytics queries to prevent abuse

## Testing Requirements

### Unit Tests

**Required Coverage**: ≥80%

**Test Cases**:

1. **Service Tests** (`service_test.go`):
   - StartSession: Creates session with correct initial state
   - EndSession: Calculates reduction percentage correctly
   - RecordFeatureUsage: Updates running averages
   - RecordPerformance: Stores metrics correctly
   - TrackClaudeMDSize: Reads file and counts lines
   - GetBusinessMetrics: Aggregates correctly
   - Error handling: Missing sessions, invalid inputs

   - SaveSessionMetrics: Insert and update
   - GetSessionMetrics: Query by ID
   - ListSessions: Filter by project, pagination
   - RecordFeatureUsage: Upsert behavior
   - GetFeatureAdoption: Correct aggregation
   - RecordPerformance: Bulk insert
   - Date range filtering
   - Empty result handling

3. **MCP Tool Tests** (`analytics_tool_test.go`):
   - Valid requests with various periods
   - Custom date ranges
   - Project filtering
   - Invalid date formats
   - Service unavailable handling
   - Output formatting

### Integration Tests


**Test Scenarios**:
1. End-to-end session lifecycle
2. Multiple sessions with aggregation
3. Feature usage tracking across sessions
4. Performance metrics collection and retrieval
5. Date range queries with edge cases
6. Cross-project aggregation

**Setup**:
```go
}
```

### Table-Driven Tests

**Example**:
```go
func TestGetBusinessMetrics(t *testing.T) {
    tests := []struct {
        name          string
        period        string
        sessions      []SessionMetrics
        want          *BusinessMetrics
        wantErr       bool
    }{
        {
            name:   "no sessions",
            period: "weekly",
            sessions: []SessionMetrics{},
            want: &BusinessMetrics{TotalSessions: 0},
        },
        {
            name:   "single session",
            period: "weekly",
            sessions: []SessionMetrics{
                {TokensBefore: 1000, TokensAfter: 500, ReductionPct: 50.0},
            },
            want: &BusinessMetrics{
                TotalSessions: 1,
                AvgTokenReduction: 50.0,
                TotalTimeSavedMin: 5.0,
            },
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Edge Cases

**Must Test**:
- Zero sessions in period
- Division by zero (AvgTokenReduction with 0 sessions)
- Single session edge cases
- Date range boundaries (start = end, inverted dates)
- Very large reductions (>100% theoretical)
- Negative tokens (error case)
- Concurrent session updates
- Malformed JSON in feature maps

### Performance Tests

**Benchmarks Required**:
```go
func BenchmarkStartSession(b *testing.B)
func BenchmarkRecordFeatureUsage(b *testing.B)
func BenchmarkGetBusinessMetrics(b *testing.B)
```

**Performance Targets**:
- StartSession: <50ms (p95)
- RecordFeatureUsage: <100ms (p95)
- GetBusinessMetrics: <500ms (p95)

## Usage Examples

### Example 1: Track Session with Features

```go
// Start session
session, err := analyticsService.StartSession(ctx, "/home/user/project", 10000)
if err != nil {
    return err
}

// Use features
analyticsService.RecordFeatureUsage(ctx, "checkpoint_search", "/home/user/project", 235.5, true)
analyticsService.RecordFeatureUsage(ctx, "checkpoint_save", "/home/user/project", 180.2, true)

// Record performance
analyticsService.RecordPerformance(ctx, "checkpoint_search", 235.5, true, false, 500, 5, "/home/user/project")

// End session
err = analyticsService.EndSession(ctx, session.ID, 5000)
// Result: 50% token reduction
```

### Example 2: Retrieve Weekly Analytics

```go
// Get business metrics for last week
metrics, err := analyticsService.GetBusinessMetrics(
    ctx,
    "weekly",
    time.Now().AddDate(0, 0, -7),
    time.Now(),
)

fmt.Printf("Sessions: %d\n", metrics.TotalSessions)
fmt.Printf("Avg Reduction: %.1f%%\n", metrics.AvgTokenReduction)
fmt.Printf("Time Saved: %.1f min\n", metrics.TotalTimeSavedMin)
fmt.Printf("Cost Saved: $%.2f\n", metrics.EstimatedCostSave)
```

### Example 3: Track CLAUDE.md Optimization

```go
// Before optimization
analyticsService.TrackClaudeMDSize(ctx, "/home/user/project", false)

// [Run optimization]

// After optimization
analyticsService.TrackClaudeMDSize(ctx, "/home/user/project", true)

// Get history
history, err := store.GetClaudeMDHistory(ctx, "/home/user/project", 10)
// Analyze size reduction over time
```

### Example 4: MCP Tool Usage (from Claude Code)

```bash
# Weekly analytics (default)
mcp__contextd__analytics_get()

# Monthly analytics for specific project
mcp__contextd__analytics_get(
    period: "monthly",
    project_path: "/home/user/projects/my-app"
)

# Custom date range
mcp__contextd__analytics_get(
    start_date: "2025-10-01",
    end_date: "2025-11-01"
)
```

### Example 5: Feature Adoption Analysis

```go
// Get all feature adoption metrics
adoption, err := analyticsService.GetFeatureAdoption(ctx, "/home/user/project")

// Find most used features
sort.Slice(adoption, func(i, j int) bool {
    return adoption[i].Count > adoption[j].Count
})

for i, fa := range adoption[:5] {
    fmt.Printf("%d. %s: %d uses (%.1f ms, %.1f%% success)\n",
        i+1, fa.Feature, fa.Count, fa.AvgLatencyMs, fa.SuccessRate*100)
}
```

## Implementation Status

**Status**: ✅ Complete (v1.0.0)

**Completed Features**:
- [x] Session tracking with token metrics
- [x] Feature adoption recording
- [x] Performance metrics collection
- [x] Business impact calculation
- [x] CLAUDE.md size tracking
- [x] Time period aggregations (daily/weekly/monthly/all-time)
- [x] Project-level filtering
- [x] MCP tool integration (analytics_get)
- [x] OpenTelemetry instrumentation
- [x] Comprehensive test coverage (>80%)

**Future Enhancements** (v2.0.0+):
- [ ] Real-time analytics dashboard
- [ ] Historical trend charts
- [ ] Comparative analytics (before/after)
- [ ] Custom metrics and KPIs
- [ ] Export to CSV/JSON
- [ ] Automated reports (daily/weekly summary)
- [ ] Anomaly detection (unusual drops in performance)
- [ ] Retention policies and archival

## Related Documentation

- **Architecture**: [docs/architecture/adr/](../../architecture/adr/)
- **Testing Standards**: [docs/standards/testing-standards.md](../../standards/testing-standards.md)
- **Coding Standards**: [docs/standards/coding-standards.md](../../standards/coding-standards.md)
- **MCP Integration**: [docs/specs/mcp/](../mcp/)
- **Monitoring**: [docs/guides/MONITORING-SETUP.md](../../guides/MONITORING-SETUP.md)

## References

- OpenTelemetry Go SDK: https://github.com/open-telemetry/opentelemetry-go
- Model Context Protocol: https://modelcontextprotocol.io
- Time-series aggregation patterns: https://www.timescale.com/blog/time-series-data-aggregation/
