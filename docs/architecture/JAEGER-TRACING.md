# Jaeger Distributed Tracing Integration

This document explains how to use Jaeger to trace and analyze contextd operations, helping verify performance goals and troubleshoot issues.

## Overview

Jaeger is integrated with contextd to provide:
- **End-to-end tracing** of MCP tool invocations
- **Dependency visualization** showing how components interact
- **Latency breakdowns** to identify bottlenecks
- **Error tracking** with full context

## Architecture

```
┌──────────────────┐
│   Claude Code    │
│   (MCP Client)   │
└────────┬─────────┘
         │ stdio
         ▼
┌──────────────────┐
│  contextd --mcp  │
│  OpenTelemetry   │
│  Instrumentation │
└────────┬─────────┘
         │ OTLP/gRPC
         ▼
┌──────────────────┐
│  Jaeger          │
│  All-in-One      │
│  - Collector     │
│  - Query Service │
│  - UI            │
└──────────────────┘
```

## Setup

### 1. Start Jaeger with Docker Compose

Jaeger is already configured in `docker-compose.yml`:

```bash
docker-compose up -d

# Verify Jaeger is running
docker ps | grep jaeger
curl -s http://localhost:16686/
```

### 2. Configure contextd for Jaeger

Set the OTLP endpoint to point to local Jaeger:

```bash
# For API mode
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
./contextd

# For MCP mode (in Claude Code config)
{
  "mcpServers": {
    "contextd": {
      "command": "/path/to/contextd",
      "args": ["--mcp"],
      "env": {
        "OPENAI_API_KEY": "sk-xxx",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
        "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf"
      }
    }
  }
}
```

### 3. Access Jaeger UI

Open in your browser:
```
http://localhost:16686
```

You should see the Jaeger UI with "contextd" in the service dropdown.

## Using Jaeger for Analysis

### Viewing Traces

1. **Select Service**: Choose "contextd" from the service dropdown
2. **Select Operation**: Choose an MCP tool (e.g., "mcp.checkpoint_save")
3. **Set Time Range**: Last hour, last 15 minutes, etc.
4. **Click "Find Traces"**

### Understanding Trace Structure

Each MCP tool invocation creates a trace with this hierarchy:

```
mcp.checkpoint_save (root span)
├── checkpoint.create (service call)
│   ├── embedding.generate (OpenAI API)
│   │   └── http.client.request (HTTP call)
│       └── grpc.client.request (gRPC call)
└── validation (input validation)
```

### Key Metrics to Monitor

#### 1. Tool Invocation Latency

**Goal**: All MCP tools complete in <500ms

**Where to check**:
- Jaeger UI → Service "contextd" → Operation "mcp.checkpoint_save"
- Look at the total span duration

**What to look for**:
- p50 (median): Should be <200ms
- p95: Should be <500ms
- p99: Should be <1000ms

**Red flags**:
- Consistently over 500ms
- High variance (some fast, some very slow)
- Increasing trend over time

#### 2. Embedding Generation Time

**Goal**: OpenAI API calls complete in <200ms

**Where to check**:
- Expand a trace → Find "embedding.generate" span
- Check duration

**What to look for**:
- Cache hits: <5ms
- Cache misses: 100-300ms
- High cache hit rate (>70%)

**Red flags**:
- All requests taking >300ms (API issues)
- Cache hit rate <50% (cache not working)
- Timeouts or errors


**Goal**: Vector searches complete in <50ms

**Where to check**:
- Check duration

**What to look for**:
- Searches: 10-50ms
- Inserts: 5-20ms
- Consistent performance

**Red flags**:
- Searches >100ms (index issues)
- Inserts >50ms (batch size too large)
- Increasing latency over time (data growth)

#### 4. End-to-End Latency Breakdown

Example trace for `checkpoint_save`:

```
Total: 245ms
├── Validation: 2ms (1%)
├── Embedding generation: 180ms (73%)
│   └── OpenAI API: 175ms
    └── gRPC call: 60ms
```

**Analysis**:
- Embedding is the bottleneck (73% of time)
- Validation is negligible

**Optimization opportunities**:
- Enable embedding cache (if not already)
- Consider batching multiple operations
- Use smaller embedding model if acceptable

## Performance Testing Scenarios

### Scenario 1: Cold Start (No Cache)

**Test**: First checkpoint save of the session

**Expected trace**:
- Embedding generation: 150-250ms (OpenAI API)
- Total: 200-350ms

**Command**:
```bash
# Clear embedding cache
rm -rf /tmp/embedding_cache/*

# Run test
# In Claude Code: "Save checkpoint: Test cold start performance"
```

**What to verify in Jaeger**:
- embedding.generate has "cache_hit=false" tag
- OpenAI API call duration is reasonable
- No errors or retries

### Scenario 2: Warm Cache

**Test**: Second checkpoint with similar text

**Expected trace**:
- Embedding generation: 1-5ms (cache hit)
- Total: 30-80ms

**Command**:
```bash
# In Claude Code: "Save checkpoint: Test warm cache performance"
# Then immediately: "Save checkpoint: Test warm cache performance again"
```

**What to verify in Jaeger**:
- Second trace has "cache_hit=true" tag
- Embedding generation is <10ms
- Overall latency reduced by >50%

### Scenario 3: Search Performance

**Test**: Semantic search across checkpoints

**Expected trace**:
- Embedding generation: 150-200ms (query embedding)
- Total: 200-350ms

**Command**:
```bash
# Create 10 checkpoints first
# In Claude Code: "Search for checkpoints about performance testing"
```

**What to verify in Jaeger**:
- Search with top_k=5: <50ms
- Search with top_k=100: <100ms

### Scenario 4: Error Handling

**Test**: Tool invocation with validation error

**Expected trace**:
- Validation: 1-2ms
- Error recorded in span
- No downstream calls

**Command**:
```bash
# In Claude Code, ask to save checkpoint with empty summary
# This should fail validation
```

**What to verify in Jaeger**:
- Span status is "error"
- Error message is recorded
- Fast failure (<10ms)

### Scenario 5: Timeout Handling

**Test**: Operation that times out

**Expected trace**:
- Operation reaches timeout (30s)
- Context cancellation propagates
- Error recorded with "timeout" category

**Command**:
```bash
# Simulate slow OpenAI API by setting low timeout
OPENAI_TIMEOUT=100ms ./contextd --mcp
```

**What to verify in Jaeger**:
- Span duration ~100ms
- Error: "context deadline exceeded"
- Proper cleanup (no hanging operations)

## Analyzing Common Issues

### Issue: High Latency for All Operations

**Symptoms**:
- All tool invocations >1s
- Consistent across all tools

**How to diagnose in Jaeger**:
1. Find a slow trace
2. Expand all spans
3. Look for the longest span

**Common causes**:
- OpenAI API slow: Upgrade plan or use faster model
- Network issues: Check connectivity to external services

### Issue: Intermittent Slow Operations

**Symptoms**:
- Most operations fast (<200ms)
- Occasional operations very slow (>2s)

**How to diagnose in Jaeger**:
1. Compare fast vs slow traces side-by-side
2. Look for differences in span structure
3. Check error logs in slow traces

**Common causes**:
- Embedding cache misses
- API rate limiting (check for retry spans)

### Issue: Memory Leaks

**Symptoms**:
- Increasing latency over time
- Eventually OOM or crash

**How to diagnose in Jaeger**:
1. Plot latency over time
2. Look for gradual increase
3. Check trace count over time

**Common causes**:
- Embedding cache growing unbounded
- Connection pool exhaustion

## Advanced Analysis

### Comparing API vs MCP Mode Performance

Run the same operation in both modes and compare:

```bash
# API mode trace (HTTP transport)
curl http://localhost:8080/api/v1/checkpoints \
  -H "Content-Type: application/json" \
  -d '{"summary":"test","description":"test","project_path":"/tmp"}'

# MCP mode trace (via Claude Code)
# Ask Claude: "Save checkpoint: test"
```

**What to compare**:
- Overhead of MCP protocol vs direct API
- Difference in span structure
- Any additional validation or processing

### Service Dependency Map

Jaeger shows service dependencies:

1. Go to "Dependencies" tab
2. Select time range
3. View graph of service interactions

**Expected graph**:
```
contextd → OpenAI API
```

**Unexpected patterns**:
- Calls to unknown services (security issue?)
- Circular dependencies (architecture problem)
- Missing expected calls (service down?)

### Trace Sampling and Retention

**Default configuration**:
- Sample rate: 100% (all traces)
- Retention: 24 hours

**For production**:
```yaml
# docker-compose.yml - Jaeger service
environment:
  COLLECTOR_OTLP_ENABLED: "true"
  SPAN_STORAGE_TYPE: badger
  BADGER_EPHEMERAL: "false"
  BADGER_DIRECTORY_VALUE: /badger/data
  BADGER_DIRECTORY_KEY: /badger/key
  SAMPLING_STRATEGIES_FILE: /etc/jaeger/sampling.json
volumes:
  - ./jaeger-data:/badger
  - ./jaeger-sampling.json:/etc/jaeger/sampling.json:ro
```

**sampling.json**:
```json
{
  "default_strategy": {
    "type": "probabilistic",
    "param": 0.1
  },
  "service_strategies": [
    {
      "service": "contextd",
      "type": "ratelimiting",
      "param": 100
    }
  ]
}
```

## Performance Benchmarks

### Baseline Performance Goals

| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| checkpoint_save | <200ms | <400ms | <800ms |
| checkpoint_search | <150ms | <300ms | <600ms |
| checkpoint_list | <50ms | <100ms | <200ms |
| remediation_save | <200ms | <400ms | <800ms |
| remediation_search | <100ms | <200ms | <400ms |
| troubleshoot | <500ms | <1000ms | <2000ms |
| list_patterns | <100ms | <200ms | <400ms |

### Component-Level Goals

| Component | Target Latency |
|-----------|----------------|
| Input validation | <5ms |
| Embedding generation (cached) | <10ms |
| Embedding generation (uncached) | <250ms |

## Troubleshooting Jaeger

### Jaeger UI Not Accessible

```bash
# Check if Jaeger is running
docker ps | grep jaeger

# Check logs
docker logs local-jaeger

# Restart Jaeger
docker-compose restart jaeger
```

### No Traces Appearing

1. **Check OTLP endpoint**:
   ```bash
   echo $OTEL_EXPORTER_OTLP_ENDPOINT
   # Should be: http://localhost:4318
   ```

2. **Verify contextd is sending traces**:
   ```bash
   # Run contextd with verbose logging
   OTEL_LOG_LEVEL=debug ./contextd
   ```

3. **Test OTLP endpoint**:
   ```bash
   curl -X POST http://localhost:4318/v1/traces \
     -H "Content-Type: application/json" \
     -d '{"resourceSpans":[]}'
   ```

### Traces Missing Spans

**Symptoms**: Traces appear but some spans are missing

**Causes**:
- Spans not properly closed (missing `defer span.End()`)
- Context not propagated correctly
- Errors in span recording

**How to fix**:
1. Check pkg/mcp/tools.go for proper span management
2. Verify context is passed to all service calls
3. Check for panics that skip span.End()

## Exporting Trace Data

### Export traces for analysis:

```bash
# Export all traces for last hour
curl 'http://localhost:16686/api/traces?service=contextd&limit=1000' > traces.json

# Analyze with jq
cat traces.json | jq '.data[].spans[] | {operation: .operationName, duration: .duration}'

# Calculate p95 latency
cat traces.json | jq -r '.data[].spans[] | select(.operationName=="mcp.checkpoint_save") | .duration' | \
  sort -n | awk '{a[NR]=$1} END{print a[int(NR*0.95)]}'
```

## Integration with Grafana

Jaeger can be used as a data source in Grafana:

1. Add Jaeger data source in Grafana
2. Create dashboard with trace queries
3. Set up alerts for slow operations

**Example Grafana panel**:
- Query: p95 latency for mcp.checkpoint_save
- Threshold: 500ms (yellow), 1000ms (red)
- Alert: If p95 > 500ms for 5 minutes

## Next Steps

1. **Run performance tests**: Use scenarios above to validate goals
2. **Establish baselines**: Record p50/p95/p99 for each tool
3. **Set up monitoring**: Create Grafana dashboards
4. **Continuous testing**: Run tests after each change
5. **Optimize bottlenecks**: Use traces to identify and fix slow operations

## Summary

Jaeger provides complete visibility into contextd operations:

- ✅ End-to-end tracing of MCP tools
- ✅ Performance analysis with latency breakdowns
- ✅ Error tracking with full context
- ✅ Service dependency visualization
- ✅ Easy troubleshooting with detailed traces

Use Jaeger to verify that contextd meets performance goals and to identify optimization opportunities.

**Quick Reference**:
- Jaeger UI: http://localhost:16686
- OTLP HTTP: http://localhost:4318
- OTLP gRPC: localhost:4317
- Service name: "contextd"
