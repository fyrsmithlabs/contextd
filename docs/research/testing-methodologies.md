# Testing Methodologies for Bayesian Confidence System

**Status**: R&D Discovery
**Branch**: `research/testing-methodologies`
**Date**: 2025-12-10

## Objective

Validate the Bayesian confidence system using realistic test scenarios. Specifically:
1. Behavioral validation - prove the math works
2. End-to-end simulation - use real conversation data to verify confidence converges correctly

## Key Findings

### 1. Testing Approaches for Self-Improving AI Systems

| Approach | Description | Fit for contextd |
|----------|-------------|------------------|
| **LLM-as-Evaluator** | Use Claude to simulate users providing feedback | High - can generate realistic feedback patterns |
| **Conversation Replay** | Replay real JSONL session logs against test system | High - we have session data in `~/.claude/projects/` |
| **Property-Based Testing** | Test invariants (confidence increases with positive feedback) | High - Hypothesis library |
| **Agent Simulation** | Synthetic user agents that create projects, use memories, report outcomes | High - can use Anthropic Agent SDK |
| **Docker Compose Multi-Container** | Isolated test env with user-agent + system-under-test | High - reproducible, clean ONNX env |

### 2. Recommended Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Compose Environment                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐     ┌──────────────────┐                  │
│  │  User Agent      │────▶│  contextd        │                  │
│  │  (Claude-based)  │     │  (System Under   │                  │
│  │                  │◀────│   Test)          │                  │
│  │  - Create project│     │                  │                  │
│  │  - Record memory │     │  - MCP Server    │                  │
│  │  - Search memory │     │  - chromem DB    │                  │
│  │  - Give feedback │     │  - FastEmbed     │                  │
│  │  - Report outcome│     │                  │                  │
│  └──────────────────┘     └──────────────────┘                  │
│           │                        │                             │
│           ▼                        ▼                             │
│  ┌──────────────────────────────────────────┐                   │
│  │           Shared Volume                   │                   │
│  │  - Test results (JSONL)                   │                   │
│  │  - Seed data (exported vectordb)          │                   │
│  │  - Session replays                        │                   │
│  └──────────────────────────────────────────┘                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Data Sources Available

| Source | Location | Format | Use |
|--------|----------|--------|-----|
| Session exports | `~/.claude/projects/-home-dahendel-projects-contextd/*.jsonl` | JSONL | Replay real conversations |
| Global history | `~/.claude/history.jsonl` | JSONL | Index of all sessions |
| Vectorstore | `~/.config/contextd/vectorstore/` | GOB files | Seed data for tests |
| Checkpoints | contextd vectorstore | Stored via MCP | Resume points |

### 4. User Agent Simulation Patterns

#### Pattern A: LLM-as-User with Persona
```go
// Pseudocode for synthetic user agent
type SyntheticUser struct {
    Persona     string   // "Developer working on Go project"
    Goals       []string // ["Find error handling patterns", "Avoid past mistakes"]
    Constraints []string // ["Budget-conscious", "Prefers tested solutions"]
    History     []Turn
}

func (u *SyntheticUser) GenerateMessage(systemResponse string) string {
    prompt := fmt.Sprintf(`
        You are %s.
        Goals: %v
        History: %v
        System just said: %s

        Generate your next message.
    `, u.Persona, u.Goals, u.History, systemResponse)

    return claude.Generate(prompt)
}

func (u *SyntheticUser) GenerateFeedback(memory Memory) Feedback {
    // Use bounded rationality - satisfice, don't optimize
    prompt := fmt.Sprintf(`
        You retrieved this memory: %s
        Your task outcome: %s

        Was this memory helpful? Rate:
        - relevance (0-10)
        - accuracy (0-10)
        - actionability (0-10)

        Would you mark this as helpful? (yes/no)
    `, memory.Content, u.TaskOutcome)

    return claude.GenerateStructured(prompt)
}
```

#### Pattern B: Conversation Replay
```go
// Replay real sessions against test system
func ReplaySession(sessionFile string, sut *contextd.Server) TestResult {
    lines := readJSONL(sessionFile)

    for _, line := range lines {
        if line.Type == "user_message" {
            // Extract any memory_record, memory_search, memory_feedback calls
            toolCalls := extractToolCalls(line)
            for _, call := range toolCalls {
                result := sut.HandleToolCall(call)
                // Compare with original result if available
            }
        }
    }

    return compareConfidenceEvolution(original, replayed)
}
```

#### Pattern C: Property-Based Testing with Hypothesis
```python
from hypothesis import given, strategies as st

@given(st.lists(st.booleans(), min_size=1, max_size=100))
def test_confidence_converges_with_consistent_feedback(feedbacks):
    """If all feedback is positive, confidence should trend upward."""
    agent = MemoryAgent()
    memory_id = agent.record("Test memory", "content")

    confidences = []
    for is_positive in feedbacks:
        agent.feedback(memory_id, helpful=is_positive)
        confidences.append(agent.get_confidence(memory_id))

    if all(feedbacks):
        # All positive → confidence should increase or stay high
        assert confidences[-1] >= confidences[0]
    elif not any(feedbacks):
        # All negative → confidence should decrease or stay low
        assert confidences[-1] <= confidences[0]
```

### 5. Behavioral Test Scenarios

#### Scenario 1: Confidence Increases with Positive Signals
```
Given: A memory with initial confidence 0.5
When: 10 positive outcome signals are recorded
Then: Confidence should be > 0.7
```

#### Scenario 2: Weight Learning Shifts Toward Predictive Signals
```
Given: Initial weights (explicit: 7:3, outcome: 5:5)
When: Outcome signals consistently predict feedback correctly
Then: Outcome weight should increase relative to explicit
```

#### Scenario 3: Convergence Under Mixed Signals
```
Given: A memory receiving 70% positive, 30% negative signals
When: 100 signals are recorded
Then: Confidence should stabilize around 0.65-0.75
And: Variance of confidence should decrease over time
```

#### Scenario 4: Real Conversation Replay
```
Given: Session JSONL from ~/.claude/projects/
When: Replayed against clean contextd instance
Then: Memories recorded should have confidence matching expected patterns
And: Feedback signals should adjust confidence as designed
```

### 6. Docker Container Design

```dockerfile
# test/Dockerfile
FROM golang:1.25-bookworm

# Install ONNX runtime
RUN apt-get update && apt-get install -y curl tar
RUN curl -L https://github.com/microsoft/onnxruntime/releases/download/v1.20.1/onnxruntime-linux-x64-1.20.1.tgz | \
    tar xz -C /usr/local --strip-components=1

ENV LD_LIBRARY_PATH=/usr/local/lib
ENV ONNX_RUNTIME_LIB_PATH=/usr/local/lib/libonnxruntime.so

# Copy contextd source
WORKDIR /app
COPY . .

# Build
RUN CGO_ENABLED=1 go build -o /usr/local/bin/contextd ./cmd/contextd

# Import seed data (optional)
COPY test/seed-data/ /root/.config/contextd/vectorstore/

# Run tests
CMD ["go", "test", "./...", "-v", "-tags=integration"]
```

```yaml
# test/docker-compose.yml
version: '3.8'

services:
  contextd:
    build:
      context: ..
      dockerfile: test/Dockerfile
    volumes:
      - test-results:/results
      - ./seed-data:/seed:ro
    environment:
      - CONTEXTD_VECTORSTORE_PROVIDER=chromem
      - CONTEXTD_LOG_LEVEL=debug

  user-agent:
    build:
      context: ./user-agent
    depends_on:
      - contextd
    environment:
      - CONTEXTD_URL=http://contextd:8080
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - TEST_SCENARIOS=/scenarios
    volumes:
      - ./scenarios:/scenarios:ro
      - test-results:/results

volumes:
  test-results:
```

### 7. Implementation Options

#### Option A: Go-native Test Framework
- Use `testing` package with table-driven tests
- Property tests via `gopter` or `rapid`
- Docker via `testcontainers-go`
- **Pros**: Single language, fast iteration
- **Cons**: Less mature LLM tooling

#### Option B: Python Test Harness
- Use `pytest` + `hypothesis` for property tests
- Use `anthropic` SDK for user agent simulation
- Use `pytest-docker` for container orchestration
- **Pros**: Rich ecosystem, better LLM tooling
- **Cons**: Two languages to maintain

#### Option C: Hybrid (Recommended)
- Go for unit/integration tests of contextd internals
- Python for LLM-driven user agent simulation
- Docker Compose to orchestrate both
- **Pros**: Best of both worlds
- **Cons**: More complex setup

### 8. Next Steps

1. **Decide on architecture** (Option A, B, or C)
2. **Export seed data** from current vectorstore
3. **Build test Dockerfile** with ONNX runtime
4. **Implement behavioral tests** for confidence system
5. **Build user agent simulator** (Go or Python)
6. **Create session replay tooling**
7. **Write spec** based on validated approach

### 9. Open Questions

1. Should user agent use Claude API directly or local model?
2. How much seed data is needed for meaningful tests?
3. Should we test signal rollup (30-day aggregation) or just recent signals?
4. How to handle non-determinism in LLM-generated feedback?

## References

- [DeepEval](https://deepeval.com) - LLM evaluation framework
- [Hypothesis](https://hypothesis.works) - Property-based testing
- [GoReplay](https://goreplay.org) - HTTP traffic replay
- [Anthropic Agent SDK](https://github.com/anthropics/anthropic-sdk-python)
- [Scenario Framework](https://scenario.langwatch.ai) - Agent simulation testing
- [LongMemEval](https://arxiv.org/html/2510.23730v1) - Cross-session memory benchmarks
