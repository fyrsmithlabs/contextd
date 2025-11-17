# Context Monitoring Specification

## Problem Statement

**URGENT**: Claude Code context window fills up quickly (reaching 70-90% of 200K token limit), but there's **no visibility** into:
- Current context usage percentage
- When auto-checkpoint triggers fire (70% = silent save, 90% = warning)
- Token reduction effectiveness from checkpoints
- Whether compaction strategy is working

**Current State**: Analytics tracks SessionMetrics (TokensBefore/TokensAfter/TokensSaved) in database, but NOT exposed as Prometheus metrics for monitoring dashboard.

**Desired State**: Real-time context usage monitoring in `ctxd monitor` dashboard with alerts when approaching limits.

## Requirements

### Functional Requirements

1. **Real-Time Context Usage Tracking**
   - Display current session token count
   - Show usage percentage (tokens_used / 200000 * 100)
   - Visual progress bar (green < 70%, yellow 70-90%, red > 90%)
   - Sparkline showing usage trend

2. **Threshold Monitoring**
   - Track 70% threshold hits (auto-checkpoint triggers)
   - Track 90% threshold hits (warning triggers)
   - Count threshold violations per session
   - Alert when thresholds crossed

3. **Checkpoint Effectiveness Metrics**
   - Tokens before checkpoint
   - Tokens after checkpoint
   - Tokens saved (reduction amount)
   - Reduction percentage
   - Average reduction over time

### Non-Functional Requirements

1. **Performance**: Metric recording < 1ms overhead
2. **Accuracy**: Real-time updates (not delayed)
3. **Visibility**: Metrics appear in VictoriaMetrics within 15s

## Architecture

### New OTEL Metrics

Add to `pkg/analytics/service.go`:

```go
// Context usage metrics
contextTokensUsed      metric.Int64UpDownCounter  // Current session tokens
contextTokensLimit     metric.Int64Gauge          // Max tokens (200K)
contextUsagePercent    metric.Float64Gauge        // Usage percentage
contextThresholdHit    metric.Int64Counter        // 70%/90% hits

// Checkpoint effectiveness
checkpointTokensBefore metric.Int64Histogram      // Before checkpoint
checkpointTokensAfter  metric.Int64Histogram      // After checkpoint
checkpointTokensSaved  metric.Int64Histogram      // Tokens saved
checkpointReductionPct metric.Float64Histogram    // Reduction %
```

### Prometheus Metric Names

```
contextd_context_tokens_used         - Current session token count
contextd_context_tokens_limit        - Maximum token limit (200000)
contextd_context_usage_percent       - Usage percentage (0-100)
contextd_context_threshold_hit_total - Threshold violation counter
  {threshold="70_percent"}           - Auto-checkpoint triggers
  {threshold="90_percent"}           - Warning triggers
contextd_checkpoint_tokens_before    - Tokens before checkpoint
contextd_checkpoint_tokens_after     - Tokens after checkpoint
contextd_checkpoint_tokens_saved     - Tokens reduction amount
contextd_checkpoint_reduction_pct    - Reduction percentage
```

### Integration Points

**1. Metric Initialization** (`pkg/analytics/service.go:NewService()`)

Add after line 113:

```go
// Context usage metrics
contextTokensUsed, err := meter.Int64UpDownCounter(
    "contextd.context.tokens_used",
    metric.WithDescription("Current session token count"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create context tokens used: %w", err)
}

contextTokensLimit, err := meter.Int64Gauge(
    "contextd.context.tokens_limit",
    metric.WithDescription("Maximum token limit"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create context tokens limit: %w", err)
}

contextUsagePercent, err := meter.Float64Gauge(
    "contextd.context.usage_percent",
    metric.WithDescription("Context usage percentage"),
    metric.WithUnit("%"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create context usage percent: %w", err)
}

contextThresholdHit, err := meter.Int64Counter(
    "contextd.context.threshold_hit",
    metric.WithDescription("Context threshold violations (70%/90%)"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create context threshold hit: %w", err)
}

// Checkpoint effectiveness metrics
checkpointTokensBefore, err := meter.Int64Histogram(
    "contextd.checkpoint.tokens_before",
    metric.WithDescription("Tokens before checkpoint"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create checkpoint tokens before: %w", err)
}

checkpointTokensAfter, err := meter.Int64Histogram(
    "contextd.checkpoint.tokens_after",
    metric.WithDescription("Tokens after checkpoint"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create checkpoint tokens after: %w", err)
}

checkpointTokensSaved, err := meter.Int64Histogram(
    "contextd.checkpoint.tokens_saved",
    metric.WithDescription("Tokens saved by checkpoint"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create checkpoint tokens saved: %w", err)
}

checkpointReductionPct, err := meter.Float64Histogram(
    "contextd.checkpoint.reduction_pct",
    metric.WithDescription("Checkpoint token reduction percentage"),
    metric.WithUnit("%"),
)
if err != nil {
    return nil, fmt.Errorf("failed to create checkpoint reduction pct: %w", err)
}
```

Update return statement to include new metrics (line 115):

```go
return &Service{
    store:                  store,
    tracer:                 tracer,
    meter:                  meter,
    sessionCounter:         sessionCounter,
    tokenReductionHist:     tokenReductionHist,
    featureUsageCounter:    featureUsageCounter,
    operationLatencyHist:   operationLatencyHist,
    cacheHitCounter:        cacheHitCounter,
    // Context monitoring
    contextTokensUsed:      contextTokensUsed,
    contextTokensLimit:     contextTokensLimit,
    contextUsagePercent:    contextUsagePercent,
    contextThresholdHit:    contextThresholdHit,
    checkpointTokensBefore: checkpointTokensBefore,
    checkpointTokensAfter:  checkpointTokensAfter,
    checkpointTokensSaved:  checkpointTokensSaved,
    checkpointReductionPct: checkpointReductionPct,
}, nil
```

**2. Recording Methods** (add after EndSession method)

```go
// RecordContextUsage records current context window usage
func (s *Service) RecordContextUsage(ctx context.Context, tokensUsed, tokensLimit int) {
    ctx, span := s.tracer.Start(ctx, "analytics.RecordContextUsage")
    defer span.End()

    // Record current usage
    s.contextTokensUsed.Add(ctx, int64(tokensUsed))
    s.contextTokensLimit.Record(ctx, int64(tokensLimit))

    // Calculate percentage
    usagePct := 0.0
    if tokensLimit > 0 {
        usagePct = (float64(tokensUsed) / float64(tokensLimit)) * 100.0
    }
    s.contextUsagePercent.Record(ctx, usagePct)

    // Track threshold violations
    attrs := []attribute.KeyValue{
        attribute.Int("tokens_used", tokensUsed),
        attribute.Int("tokens_limit", tokensLimit),
        attribute.Float64("usage_percent", usagePct),
    }

    if usagePct >= 90.0 {
        s.contextThresholdHit.Add(ctx, 1, metric.WithAttributes(
            append(attrs, attribute.String("threshold", "90_percent"))...,
        ))
        span.SetAttributes(attribute.Bool("threshold_90_hit", true))
    } else if usagePct >= 70.0 {
        s.contextThresholdHit.Add(ctx, 1, metric.WithAttributes(
            append(attrs, attribute.String("threshold", "70_percent"))...,
        ))
        span.SetAttributes(attribute.Bool("threshold_70_hit", true))
    }
}

// RecordCheckpointEffectiveness records checkpoint token reduction
func (s *Service) RecordCheckpointEffectiveness(ctx context.Context, tokensBefore, tokensAfter int) {
    ctx, span := s.tracer.Start(ctx, "analytics.RecordCheckpointEffectiveness")
    defer span.End()

    tokensSaved := tokensBefore - tokensAfter
    reductionPct := 0.0
    if tokensBefore > 0 {
        reductionPct = (float64(tokensSaved) / float64(tokensBefore)) * 100.0
    }

    attrs := []attribute.KeyValue{
        attribute.Int("tokens_before", tokensBefore),
        attribute.Int("tokens_after", tokensAfter),
        attribute.Int("tokens_saved", tokensSaved),
        attribute.Float64("reduction_pct", reductionPct),
    }

    s.checkpointTokensBefore.Record(ctx, int64(tokensBefore), metric.WithAttributes(attrs...))
    s.checkpointTokensAfter.Record(ctx, int64(tokensAfter), metric.WithAttributes(attrs...))
    s.checkpointTokensSaved.Record(ctx, int64(tokensSaved), metric.WithAttributes(attrs...))
    s.checkpointReductionPct.Record(ctx, reductionPct, metric.WithAttributes(attrs...))

    span.SetAttributes(
        attribute.Int("tokens_saved", tokensSaved),
        attribute.Float64("reduction_pct", reductionPct),
    )
}
```

**3. Instrumentation in StartSession** (line 128)

Add after session creation (around line 150):

```go
// Record initial context usage
s.RecordContextUsage(ctx, tokensBefore, 200000)
```

**4. Instrumentation in EndSession** (around line 195)

Add before return:

```go
// Record checkpoint effectiveness
s.RecordCheckpointEffectiveness(ctx, session.TokensBefore, tokensAfter)
```

## Dashboard Integration

### Query for Context Usage

```promql
# Current context usage percentage
contextd_context_usage_percent

# Current tokens used
contextd_context_tokens_used

# 70% threshold hits in last 5 minutes
rate(contextd_context_threshold_hit_total{threshold="70_percent"}[5m])

# 90% threshold hits in last 5 minutes
rate(contextd_context_threshold_hit_total{threshold="90_percent"}[5m])

# Average tokens saved per checkpoint
avg(contextd_checkpoint_tokens_saved)

# Average reduction percentage
avg(contextd_checkpoint_reduction_pct)
```

### Dashboard Display

Add to `internal/monitor/dashboard.go`:

```
┃ Context Window                                         │
┃ Usage: 142K / 200K tokens (71%) [⚠]                   │
┃ Progress: ██████████░░░░ 71%      ▃▄▅▆▇███▇▆▅        │
┃ Threshold: 70% auto-checkpoint triggered 3x today     │
┃                                                        │
┃ Checkpoint Effectiveness                               │
┃ Avg Reduction: 88%                ▇▇▇▇████▇▇▇▇        │
┃ Tokens Saved: 176K avg           Last: 200K → 24K    │
```

## Testing Strategy

### Unit Tests

Add to `pkg/analytics/service_test.go`:

```go
func TestRecordContextUsage_Below70Percent(t *testing.T) {
    // Test usage below 70% threshold
}

func TestRecordContextUsage_70PercentThreshold(t *testing.T) {
    // Test 70% threshold detection and counter increment
}

func TestRecordContextUsage_90PercentThreshold(t *testing.T) {
    // Test 90% threshold detection and counter increment
}

func TestRecordCheckpointEffectiveness_88PercentReduction(t *testing.T) {
    // Test typical checkpoint reduction (200K → 24K = 88%)
}

func TestRecordCheckpointEffectiveness_ZeroDivision(t *testing.T) {
    // Test edge case with tokensBefore = 0
}
```

### Integration Tests

1. Start contextd with analytics enabled
2. Trigger StartSession with 140K tokens (70%)
3. Verify `contextd_context_threshold_hit_total{threshold="70_percent"}` increments
4. Trigger EndSession with 24K tokens
5. Verify checkpoint effectiveness metrics recorded

## Success Criteria

- [ ] All 8 context monitoring metrics exported to VictoriaMetrics
- [ ] Dashboard displays current context usage percentage
- [ ] Progress bar changes color at 70% (yellow) and 90% (red)
- [ ] Threshold hit counters increment when crossing 70%/90%
- [ ] Checkpoint effectiveness metrics show realistic values (88% reduction)
- [ ] All tests pass (≥80% coverage)
- [ ] User can monitor context usage in real-time via `ctxd monitor`

## Implementation Priority

**CRITICAL (Immediate)**:
1. Add metric declarations to Service struct ✅ (DONE)
2. Initialize metrics in NewService ✅ (DONE)
3. Add RecordContextUsage method ✅ (DONE)
4. Add RecordCheckpointEffectiveness method ✅ (DONE)
5. Instrument StartSession ✅ (DONE)
6. Instrument EndSession ✅ (DONE)

**HIGH (Next Session)**:
7. Update dashboard to display context metrics
8. Add tests for all new methods ✅ (DONE - 9 tests added)
9. Verify metrics in VictoriaMetrics
10. Document usage in monitoring guide

**MEDIUM (Future)**:
11. Add historical context usage graphs
12. Add predictive alerts ("you'll hit 90% in 15 minutes")
13. Track context efficiency score
14. Add context optimization recommendations
