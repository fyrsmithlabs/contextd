# ADR 003: Implement 12-Factor Agents Pre-Fetch Pattern

## Status

**Accepted**

## Context

contextd's MCP tools currently follow a traditional reactive pattern where Claude must explicitly request each tool invocation. This creates unnecessary round-trips that consume tokens and add latency. The 12-Factor Agents methodology proposes a pre-fetch pattern that proactively executes high-probability tool calls to optimize performance and token efficiency.

### Problem Statement

1. **Token Inefficiency**: Multiple round-trips consume tokens for "what should I fetch?" decisions
2. **Latency Issues**: Sequential tool execution adds unnecessary delays
3. **Poor User Experience**: Slower response times and less context-aware interactions
4. **Suboptimal Resource Usage**: Underutilization of available context and computational resources

### Research Findings

Analysis of contextd usage patterns revealed high-probability scenarios for pre-fetching:

- **Project Context**: 85% likelihood of needing checkpoint search when project_path present
- **Error Messages**: 90% likelihood of needing remediation search when errors detected
- **Troubleshooting**: 75% likelihood of needing pattern listing for debugging workflows
- **Workflow Tasks**: 70% likelihood of needing skill search for development tasks

Expected benefits based on research:
- **Token Reduction**: 20-30% for applicable workflows
- **Latency Improvement**: 40-50% through parallel execution
- **Hit Rate**: 74% accuracy for pre-fetch predictions

## Decision

**Implement the 12-Factor Agents pre-fetch pattern** for contextd MCP tools with the following architecture:

### Core Components

1. **Pre-Fetch Orchestrator** (`pkg/mcp/prefetch.go`)
   - Analyzes incoming requests for pre-fetch opportunities
   - Coordinates parallel execution of pre-fetch operations
   - Injects results into MCP response context

2. **Context Analyzer** (`pkg/mcp/context_analyzer.go`)
   - Detects patterns indicating likely tool needs
   - Calculates confidence scores for pre-fetch candidates
   - Supports pluggable pattern detection algorithms

3. **Configuration System** (`pkg/mcp/prefetch_config.go`)
   - Per-tool enable/disable controls
   - Configurable confidence thresholds
   - Resource limits and timeouts

### Implementation Strategy

**Phased Rollout** over 8 weeks:

1. **Phase 1 (Weeks 1-2)**: Core infrastructure and basic pattern detection
2. **Phase 2 (Weeks 3-4)**: Tool integration for checkpoint and remediation search
3. **Phase 3 (Weeks 5-6)**: Optimization and adaptive learning
4. **Phase 4 (Weeks 7-8)**: Production hardening and monitoring

### Key Design Decisions

#### 1. Parallel Execution with Limits

**Decision**: Implement parallel pre-fetching with configurable concurrency limits

**Rationale**:
- Parallel execution maximizes latency benefits
- Limits prevent resource exhaustion
- Configurable for different deployment scenarios

**Implementation**:
```go
type PreFetchConfig struct {
    MaxParallel int           `yaml:"max_parallel"` // Default: 3
    Timeout     time.Duration `yaml:"timeout"`      // Default: 2s
}
```

#### 2. Confidence-Based Execution

**Decision**: Use confidence thresholds to determine when to pre-fetch

**Rationale**:
- Prevents wasteful pre-fetching of low-probability operations
- Allows tuning based on accuracy measurements
- Supports A/B testing of different thresholds

**Thresholds**:
- `remediation_search`: 0.8 (high precision for error matching)
- `checkpoint_search`: 0.7 (moderate for session context)
- `list_patterns`: 0.6 (lower for exploratory troubleshooting)
- `skill_search`: 0.7 (balanced for workflow assistance)

#### 3. Result Injection Strategy

**Decision**: Inject pre-fetched results directly into MCP response context

**Rationale**:
- Eliminates additional round-trips
- Provides immediate context to the model
- Maintains compatibility with existing MCP protocol

**Format**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [...],
    "prefetch": {
      "checkpoint_search": {
        "results": [...],
        "execution_time_ms": 150,
        "confidence": 0.85
      }
    }
  }
}
```

#### 4. Fallback and Error Handling

**Decision**: Implement graceful fallback when pre-fetching fails

**Rationale**:
- Pre-fetch failures must not break normal tool operation
- Users should be unaware of pre-fetch failures
- Comprehensive error tracking for debugging

**Behavior**:
- Failed pre-fetches are silently ignored
- Normal tool execution continues unaffected
- Errors are logged with full context for analysis

## Alternatives Considered

### Alternative 1: Model-Driven Pre-Fetching

**Description**: Ask the model to predict what tools it will need

**Pros**:
- Potentially higher accuracy
- Leverages model intelligence

**Cons**:
- Consumes additional tokens for prediction
- Adds complexity to prompt engineering
- Model predictions may be inconsistent

**Decision**: Rejected due to token overhead and complexity

### Alternative 2: Cache-Based Pre-Fetching

**Description**: Cache recent tool results and reuse based on context similarity

**Pros**:
- Lower computational cost
- Simpler implementation

**Cons**:
- Limited to previously executed operations
- May not capture new context patterns
- Cache invalidation complexity

**Decision**: Considered complementary, not alternative. Can be added in future phases.

### Alternative 3: Probabilistic Pre-Fetching

**Description**: Use machine learning to predict tool needs

**Pros**:
- Potentially higher accuracy over time
- Adapts to usage patterns

**Cons**:
- Significant implementation complexity
- Requires training data and model maintenance
- May not be cost-effective for current scale

**Decision**: Deferred to future optimization phase if rule-based approach proves insufficient

## Consequences

### Positive

1. **Performance Improvements**
   - 20-30% token reduction for applicable workflows
   - 40-50% latency improvement through parallelization
   - More responsive user interactions

2. **Enhanced User Experience**
   - Seamless context awareness
   - Faster response times
   - More intelligent tool suggestions

3. **Strategic Alignment**
   - Advances contextd's primary goal of context optimization
   - Supports 60% compression target from Product Roadmap
   - Differentiates from Claude Desktop's 30-40% compression

### Negative

1. **Increased Complexity**
   - Additional orchestration layer
   - Pattern detection logic to maintain
   - Configuration management overhead

2. **Resource Usage**
   - Additional server load from pre-fetch operations
   - Potential for wasted computation on failed predictions
   - Memory usage for result caching

3. **Operational Complexity**
   - Additional monitoring and metrics
   - Tuning confidence thresholds
   - Managing feature flags for rollout

### Risks

1. **Accuracy Risk**: Low pre-fetch accuracy could waste resources
   - **Mitigation**: Conservative initial thresholds, A/B testing, monitoring

2. **Performance Risk**: Pre-fetching could slow down responses
   - **Mitigation**: Strict timeouts, parallel limits, performance monitoring

3. **Context Risk**: Excessive pre-fetching could overflow context windows
   - **Mitigation**: Result size limits, priority filtering, compression

## Implementation Plan

### Phase 1: Foundation (Weeks 1-2)
- [ ] Create PreFetchOrchestrator structure
- [ ] Implement basic ContextAnalyzer
- [ ] Add configuration system
- [ ] Set up telemetry and metrics

### Phase 2: Tool Integration (Weeks 3-4)
- [ ] Implement checkpoint_search pre-fetching
- [ ] Implement remediation_search pre-fetching
- [ ] Add parallel execution framework
- [ ] Integrate result injection

### Phase 3: Optimization (Weeks 5-6)
- [ ] Measure and tune accuracy
- [ ] Optimize parallel execution
- [ ] Add adaptive learning
- [ ] Implement caching strategies

### Phase 4: Production (Weeks 7-8)
- [ ] Comprehensive testing (≥80% coverage)
- [ ] Error handling and fallbacks
- [ ] Documentation and monitoring
- [ ] Gradual rollout with feature flags

## Success Metrics

- **Pre-fetch Hit Rate**: ≥70%
- **Token Reduction**: ≥20% for applicable workflows
- **Latency Improvement**: ≥40% for pre-fetchable operations
- **Error Rate**: Zero increase in tool errors
- **Test Coverage**: ≥80%

## Monitoring Plan

### Key Metrics
1. Pre-fetch hit rate and accuracy
2. Token savings and latency improvements
3. Resource usage and performance impact
4. Error rates and fallback success

### Dashboards
- Pre-fetch performance dashboard
- Accuracy and hit rate monitoring
- Token savings visualization
- Error analysis and troubleshooting

## Rollback Plan

- Feature flags allow instant disable
- Monitoring alerts for performance regressions
- Automated rollback if error rate > threshold
- Gradual ramp-down if accuracy < acceptable level

## Related Documents

- [MCP Pre-Fetch Specification](../SPEC.md)
- [12-Factor Agents Pre-Fetch Research](../research/12-factor-agents-prefetch-analysis.md)
- [Context Optimization Architecture](../../../architecture/CONTEXT-OPTIMIZATION.md)
- [Product Roadmap](../../../PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md)

## Decision Date

2025-11-06

## Decision Makers

- contextd development team

## References

- [12-Factor Agents: Pre-Fetch Pattern](https://github.com/humanlayer/12-factor-agents/blob/main/content/appendix-13-pre-fetch.md)
- [MCP Pre-Fetch Implementation Research](../research/12-factor-agents-prefetch-analysis.md)