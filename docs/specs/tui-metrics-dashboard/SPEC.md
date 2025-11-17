# TUI Metrics Dashboard Specification

## Overview

Implement a terminal-based metrics dashboard for `ctxd` that displays live contextd metrics in a tmux-friendly format.

## Problem Statement

**Current state:**
- Grafana dashboards exist but require browser
- No quick way to monitor contextd from terminal
- Tmux workflow needs terminal-based monitoring pane

**Desired state:**
- Run `ctxd monitor` to see live metrics in terminal
- BubbleTea TUI updates every 5 seconds
- Works in tmux pane for monitoring workflow
- Fetches metrics from VictoriaMetrics HTTP API

## Requirements

### Functional Requirements

1. **Command**: `ctxd monitor [--interval=5s] [--vm-url=http://localhost:8428]`
   - Launch interactive TUI dashboard
   - Refresh interval configurable (default: 5s)
   - VictoriaMetrics URL configurable via flag or env var

2. **Metrics Display**:
   - HTTP Requests: rate (req/min), p95 latency, status code distribution
   - Embeddings: operations/min, tokens/min, cost/min
   - MCP Tools: top 5 by call count with average latency
   - System: goroutines, memory usage
   - Uptime: contextd service uptime

3. **TUI Features**:
   - Auto-refresh at configured interval
   - `q` to quit
   - `r` to force refresh
   - `j/k` or arrow keys to scroll (if content overflows)
   - Clear error messages if VictoriaMetrics unreachable

### Non-Functional Requirements

1. **Performance**:
   - Dashboard refresh < 100ms
   - VictoriaMetrics query timeout: 2s
   - Graceful degradation if metrics unavailable

2. **Usability**:
   - Works in tmux panes (minimum 80x24 terminal)
   - Clear visual hierarchy
   - No color dependency (works in non-color terminals)

3. **Reliability**:
   - Handle VictoriaMetrics downtime gracefully
   - Display "No data" for missing metrics
   - Never crash on malformed API responses

## Architecture

### Component Structure

```
cmd/ctxd/
├── monitor.go           # Cobra command for `ctxd monitor`
└── monitor_test.go      # Command tests

internal/monitor/        # New package for TUI dashboard
├── dashboard.go         # BubbleTea model
├── dashboard_test.go
├── metrics.go           # VictoriaMetrics client
├── metrics_test.go
├── formatter.go         # Format metrics for display
└── formatter_test.go
```

### Data Flow

```
1. User runs `ctxd monitor`
2. Command creates BubbleTea model
3. Model starts ticker (5s interval)
4. On tick:
   a. Fetch metrics from VictoriaMetrics API
   b. Parse and format metrics
   c. Update TUI view
5. User presses 'q' → exit
```

### VictoriaMetrics Queries

**HTTP Request Rate:**
```promql
rate(http_server_request_duration_seconds_count[1m])
```

**HTTP P95 Latency:**
```promql
histogram_quantile(0.95, rate(http_server_request_duration_seconds_bucket[1m]))
```

**Embedding Operations:**
```promql
rate(contextd_embedding_operations_total[1m])
```

**Embedding Tokens:**
```promql
rate(contextd_embedding_tokens_total[1m])
```

**Embedding Cost:**
```promql
rate(contextd_embedding_cost_USD_total[1m])
```

**MCP Tool Calls:**
```promql
topk(5, rate(contextd_mcp_tool_calls_total[1m]))
```

## Implementation Plan

### Phase 1: VictoriaMetrics Client (TDD)

**Test-first implementation:**

1. **RED**: Write test for VictoriaMetrics query
   ```go
   func TestMetricsClient_Query(t *testing.T) {
       client := NewMetricsClient("http://localhost:8428")
       result, err := client.Query(context.Background(), "up")
       require.NoError(t, err)
       assert.NotEmpty(t, result)
   }
   ```

2. **GREEN**: Implement minimal HTTP client
   ```go
   type MetricsClient struct {
       baseURL string
       client  *http.Client
   }

   func (c *MetricsClient) Query(ctx context.Context, query string) (QueryResult, error) {
       // Minimal implementation
   }
   ```

3. **REFACTOR**: Add timeout, error handling, retries

### Phase 2: Metrics Formatter (TDD)

**Test-first implementation:**

1. **RED**: Write test for formatting HTTP rate
   ```go
   func TestFormatter_FormatHTTPRate(t *testing.T) {
       rate := 45.7
       formatted := FormatRate(rate)
       assert.Equal(t, "45.7 req/min", formatted)
   }
   ```

2. **GREEN**: Implement formatter
3. **REFACTOR**: Add formatting for latency, cost, etc.

### Phase 3: BubbleTea Dashboard (TDD)

**Test-first implementation:**

1. **RED**: Write test for model initialization
2. **GREEN**: Create BubbleTea model with Init/Update/View
3. **REFACTOR**: Add keyboard handling, auto-refresh

### Phase 4: Cobra Command Integration

1. Add `monitor` subcommand to `cmd/ctxd/main.go`
2. Wire up flags (`--interval`, `--vm-url`)
3. Launch BubbleTea program

## TUI Layout

```
┌─ contextd Metrics Dashboard ──────────────────────────────┐
│ Uptime: 2h 15m        Last Update: 12:34:56 PM            │
│ VictoriaMetrics: http://localhost:8428                     │
├────────────────────────────────────────────────────────────┤
│ HTTP Requests                                              │
│ ├─ Rate: 45.7 req/min    Latency (p95): 12.3ms           │
│ └─ Status: 200 (98.5%) | 404 (1.2%) | 500 (0.3%)         │
├────────────────────────────────────────────────────────────┤
│ Embeddings                                                 │
│ ├─ Operations: 120/min      Tokens: 15.2k/min            │
│ └─ Cost: $0.0034/min                                      │
├────────────────────────────────────────────────────────────┤
│ MCP Tools (top 5)                                          │
│ ├─ checkpoint_search  : 23 calls    12ms avg             │
│ ├─ checkpoint_save    : 8 calls     45ms avg             │
│ ├─ remediation_search : 5 calls     8ms avg              │
│ ├─ troubleshoot       : 3 calls     120ms avg            │
│ └─ index_repository   : 1 call      2.5s avg             │
├────────────────────────────────────────────────────────────┤
│ System                                                     │
│ └─ Goroutines: 42    Memory: 24.5 MB                     │
└────────────────────────────────────────────────────────────┘
[q] quit  [r] refresh  [j/k] scroll  Auto-refresh: 5s
```

## Error Handling

**VictoriaMetrics unreachable:**
```
┌─ contextd Metrics Dashboard ──────────────────────────────┐
│ ⚠ Cannot connect to VictoriaMetrics                       │
│ URL: http://localhost:8428                                 │
│ Error: dial tcp 127.0.0.1:8428: connect: connection refused│
│                                                            │
│ Please ensure:                                             │
│ 1. docker-compose up -d victoriametrics                   │
│ 2. VictoriaMetrics is running on :8428                    │
│                                                            │
│ Press [q] to quit, [r] to retry                           │
└────────────────────────────────────────────────────────────┘
```

**No metrics available:**
```
│ HTTP Requests                                              │
│ └─ No data (contextd may not be running)                  │
```

## Testing Strategy

### Unit Tests

1. **MetricsClient**:
   - Query execution
   - HTTP error handling
   - Timeout handling
   - Malformed response handling

2. **Formatter**:
   - Rate formatting (req/min)
   - Latency formatting (ms, s)
   - Cost formatting ($0.00)
   - Percentage formatting
   - Edge cases (0, very large numbers, NaN)

3. **Dashboard Model**:
   - Initialization
   - Update messages (tick, key press)
   - View rendering

### Integration Tests

1. **End-to-End**:
   - Mock VictoriaMetrics server
   - Send metrics
   - Verify TUI displays correctly

### Manual Testing

1. Run against live VictoriaMetrics
2. Test in tmux pane
3. Test with VictoriaMetrics down
4. Test with no metrics
5. Test terminal resize

## Dependencies

**New dependencies:**
```go
require (
    github.com/charmbracelet/bubbletea v0.24.2
    github.com/charmbracelet/lipgloss v0.9.1  // Optional: styling
)
```

## Success Criteria

- [ ] `ctxd monitor` launches TUI dashboard
- [ ] Dashboard displays all 5 metric categories
- [ ] Auto-refresh every 5s
- [ ] Graceful error handling (VictoriaMetrics down)
- [ ] Works in tmux pane (80x24 minimum)
- [ ] All tests pass (≥80% coverage)
- [ ] TDD workflow followed (RED-GREEN-REFACTOR)
- [ ] Code reviewed by golang-reviewer skill

## Future Enhancements (Out of Scope)

- Historical graphs (sparklines)
- Alert thresholds (highlight red if p95 > 100ms)
- Multiple contextd instances
- Export snapshot to JSON
- Mouse support for scrolling
