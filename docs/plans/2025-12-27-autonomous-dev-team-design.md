# Autonomous AI Development Team Design

**Date**: 2025-12-27
**Status**: Approved
**Orchestration**: Temporal
**Tech Stack**: Go + Temporal + Contextd MCP

## Executive Summary

This design creates an autonomous AI development team that handles the complete development cycle: from GitHub issue to production-ready pull request. The system uses Temporal for durable workflow orchestration, Go for agent implementation, and Contextd's MCP server for shared learning via ReasoningBank and context folding with short-lived collections.

**Key capabilities**:
- Full autonomy (no human checkpoints during execution)
- Maximum rigor (code + tests + usage tests + benchmarks + security + consensus review + docs)
- Long-running workflows (hours to days per feature)
- Observable via Temporal UI
- Learns from experience via ReasoningBank

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│              Temporal Workflows                          │
│              (Durable Orchestration)                     │
└────────────┬────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│           Go Activities (Agent Crews)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Analysis    │  │Implementation│  │   Quality    │  │
│  │    Crew      │  │    Crew      │  │    Crew      │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│  ┌──────────────┐                                       │
│  │ Review &     │                                       │
│  │ Ship Crew    │                                       │
│  └──────────────┘                                       │
└────────────┬────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│              Contextd MCP Server                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ ReasoningBank│  │  Short-lived │  │ Checkpoints  │  │
│  │   (memory)   │  │ Collections  │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**Why Temporal?**
1. **Durable workflows**: Features take hours/days - need crash recovery
2. **Observability**: Built-in UI for monitoring progress
3. **Activity isolation**: Each crew runs as separate activity
4. **Existing infrastructure**: User has Temporal server running

## Development Workflow

### Input: GitHub Issue
**Triggers**: Webhook or polling detects new issue with `ai-dev` label

### Phase 1: Analysis
**Temporal Activity**: `AnalyzeFeature`
**Duration**: 10-30 minutes
**Agents**:
- **Requirements Agent**: Extracts specs, edge cases, acceptance criteria
- **Architecture Agent**: Reviews codebase, identifies affected components
- **Research Agent**: Checks ReasoningBank for similar features

**Outputs**:
- Feature specification document
- Architecture plan
- Affected files list
- ReasoningBank context (similar patterns from past work)

**Short-lived Collection**: `feature-{issue-id}-analysis`

### Phase 2: Implementation
**Temporal Activity**: `ImplementFeature`
**Duration**: 1-6 hours
**Agents**:
- **Code Agent**: Writes production code following architecture plan
- **Test Agent**: Writes unit + integration tests
- **Documentation Agent**: Updates README, CHANGELOG, inline docs

**Outputs**:
- Production code
- Unit tests
- Integration tests
- Updated documentation
- Git branch with commits

**Short-lived Collection**: `feature-{issue-id}-implementation`

**ReasoningBank Integration**:
- Records successful implementation patterns
- Stores edge case handling approaches
- Documents API design decisions

### Phase 3: Quality Assurance
**Temporal Activity**: `ValidateQuality`
**Duration**: 30 minutes - 2 hours
**Agents**:
- **Usage Test Agent**: Writes and runs usage tests
  - ReasoningBank integration validation
  - Feature-specific use cases
  - Edge case discovery (actively explores boundary conditions)
- **Benchmark Agent**: Performance + regression tests
- **Security Agent**: Vulnerability scan, dependency audit

**Outputs**:
- Usage test suite with edge cases
- Performance benchmark results
- Security scan report
- All tests passing confirmation

**Short-lived Collection**: `feature-{issue-id}-quality`

**Usage Test Requirements**:
```go
type UsageTestSuite struct {
    // Validate ReasoningBank integration
    TestReasoningBankIntegration()

    // Feature-specific use cases
    TestPrimaryUseCase()
    TestSecondaryUseCases()

    // Active edge case discovery
    TestBoundaryConditions()
    TestErrorPaths()
    TestConcurrency()
}
```

### Phase 4: Review & Ship
**Temporal Activity**: `ConsensusReview`
**Duration**: 20-40 minutes
**Two-layer validation**:

**Layer 1: Technical Review** (3 agents)
- **Code Reviewer**: Logic, style, maintainability
- **Architecture Reviewer**: Design consistency, patterns
- **Security Reviewer**: Vulnerabilities, best practices

**Layer 2: UX Persona Validation** (4 personas)
Based on `/test/docs/PERSONA-SIMULATION-METHODOLOGY.md`:

| Persona | Role | Focus | Pain Points |
|---------|------|-------|-------------|
| Marcus | Backend Dev (5 yrs) | Reads docs carefully | CGO setup, MCP config |
| Sarah | Frontend Dev (3 yrs) | Skims docs | Binary install, PATH issues |
| Alex | Full Stack (7 yrs) | Jumps to Quick Start | Multi-project management |
| Jordan | DevOps (6 yrs) | Security-first | Team deployment, secrets |

**Consensus Criteria**:
- All 3 technical reviewers approve
- All 4 personas validate: no breaking UX changes
- Any "breaking change" flags rejection

**Outputs**:
- Technical review report
- UX validation report
- Approval/rejection decision
- If approved: PR created with summary

**Short-lived Collection**: `feature-{issue-id}-review`

### Phase 5: Cleanup
**Temporal Activity**: `Cleanup`
**Duration**: <1 minute

**Actions**:
- Archive short-lived collections
- Record feature completion in ReasoningBank
- Update issue status
- Post PR link to issue

## Agent Communication via Contextd MCP

All agents communicate through Contextd's MCP server (HTTP transport, 2025-03-26 spec).

### ReasoningBank (Cross-feature Learning)

**Purpose**: Persistent learning across features

**Operations**:
```go
// Before starting implementation
memories := contextd.MemorySearch(ctx, &MemorySearchRequest{
    ProjectID: "contextd",
    Query: "HTTP server middleware patterns",
    Limit: 5,
})

// After successful implementation
contextd.MemoryRecord(ctx, &MemoryRecordRequest{
    ProjectID: "contextd",
    Title: "HTTP middleware with context propagation",
    Content: "Pattern: use middleware chain with context.Context...",
    Outcome: "success",
    Tags: []string{"http", "middleware", "go"},
})

// After using a memory
contextd.MemoryOutcome(ctx, &MemoryOutcomeRequest{
    MemoryID: "mem_123",
    Succeeded: true, // Did the pattern work?
})
```

**Learning Loop**:
1. Search ReasoningBank before implementation
2. Apply retrieved patterns
3. Record new patterns after success
4. Report outcome to adjust confidence scores

### Short-lived Collections (Context Folding)

**Purpose**: Isolated context per feature, auto-cleanup

**Lifecycle**:
```go
// Phase 1: Analysis crew creates collection
collection := contextd.RepositoryIndex(ctx, &IndexRequest{
    Path: "/path/to/contextd",
    TenantID: "fyrsmithlabs",
    // Returns: collection_name = "fyrsmithlabs_contextd_analysis_issue_123"
})

// Phase 2-4: All crews use explicit collection_name
results := contextd.RepositorySearch(ctx, &SearchRequest{
    Query: "MCP server initialization",
    CollectionName: collection.Name, // Explicit - no tenant_id derivation
})

// Phase 5: Cleanup deletes collection
contextd.CollectionDelete(ctx, collection.Name)
```

**Benefits**:
- Each feature has isolated context
- No pollution between features
- Automatic cleanup after PR merge
- Bounded memory usage

### Checkpoints (Long-running Workflow Recovery)

**Purpose**: Resume workflows after crashes

```go
// Before expensive operations
checkpointID := contextd.CheckpointSave(ctx, &CheckpointRequest{
    SessionID: workflowID,
    TenantID: "fyrsmithlabs",
    ProjectPath: "/contextd",
    Name: "pre-implementation",
    Summary: "Analysis complete, ready for implementation",
    Context: analysisResults.JSON(),
    FullState: workflow.State.JSON(),
})

// After crash - resume from checkpoint
state := contextd.CheckpointResume(ctx, &ResumeRequest{
    CheckpointID: checkpointID,
    Level: "context", // summary | context | full
})
```

## Temporal Workflow Structure

### Main Workflow: `FeatureDevelopmentWorkflow`

```go
func FeatureDevelopmentWorkflow(ctx workflow.Context, issue GitHubIssue) error {
    // Phase 1: Analysis
    var analysisResult AnalysisResult
    err := workflow.ExecuteActivity(ctx, AnalyzeFeature, issue).Get(ctx, &analysisResult)
    if err != nil {
        return err
    }

    // Phase 2: Implementation
    var implResult ImplementationResult
    err = workflow.ExecuteActivity(ctx, ImplementFeature, analysisResult).Get(ctx, &implResult)
    if err != nil {
        return err
    }

    // Phase 3: Quality Assurance
    var qaResult QualityResult
    err = workflow.ExecuteActivity(ctx, ValidateQuality, implResult).Get(ctx, &qaResult)
    if err != nil {
        return err
    }

    // Phase 4: Consensus Review
    var reviewResult ReviewResult
    err = workflow.ExecuteActivity(ctx, ConsensusReview, qaResult).Get(ctx, &reviewResult)
    if err != nil {
        return err
    }

    if !reviewResult.Approved {
        // Rejection: Post review feedback to issue
        workflow.ExecuteActivity(ctx, PostReviewFeedback, reviewResult)
        return fmt.Errorf("consensus review rejected: %s", reviewResult.Reason)
    }

    // Phase 5: Cleanup
    workflow.ExecuteActivity(ctx, Cleanup, reviewResult)

    return nil
}
```

### Activity Retries

**Configuration**: Temporal activities support automatic retries with exponential backoff

```go
retryPolicy := &temporal.RetryPolicy{
    InitialInterval:    time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    time.Minute,
    MaximumAttempts:    3,
}

// Applied to all activities
```

**Crash Recovery**:
- Temporal automatically restarts workflows on worker failure
- Activities checkpoint progress via Contextd
- ReasoningBank preserves learning state
- Short-lived collections survive worker crashes

### Parallel Feature Development (Optional)

**Configuration**: Enable via workflow parameter

```go
type FeatureConfig struct {
    AllowParallel bool // Default: false
    MaxParallel   int  // Default: 1
}

func MultiFeatureWorkflow(ctx workflow.Context, issues []GitHubIssue, config FeatureConfig) error {
    if !config.AllowParallel {
        // Sequential execution
        for _, issue := range issues {
            workflow.ExecuteChildWorkflow(ctx, FeatureDevelopmentWorkflow, issue).Get(ctx, nil)
        }
        return nil
    }

    // Parallel execution with limit
    var futures []workflow.Future
    for _, issue := range issues {
        if len(futures) >= config.MaxParallel {
            futures[0].Get(ctx, nil) // Wait for oldest to complete
            futures = futures[1:]
        }
        future := workflow.ExecuteChildWorkflow(ctx, FeatureDevelopmentWorkflow, issue)
        futures = append(futures, future)
    }

    // Wait for remaining
    for _, f := range futures {
        f.Get(ctx, nil)
    }
    return nil
}
```

## Go Agent Implementation

### Agent Interface

```go
type Agent interface {
    // Execute agent task with MCP context
    Execute(ctx context.Context, input AgentInput) (AgentOutput, error)

    // Name for logging/observability
    Name() string
}

type AgentInput struct {
    Task          string
    Context       map[string]interface{}
    MCPClient     *contextd.Client
    CollectionName string
}

type AgentOutput struct {
    Result    interface{}
    Artifacts []Artifact
    Error     error
}
```

### Example: Code Agent

```go
type CodeAgent struct {
    llmClient *anthropic.Client
    mcp       *contextd.Client
}

func (a *CodeAgent) Execute(ctx context.Context, input AgentInput) (AgentOutput, error) {
    // 1. Search ReasoningBank for relevant patterns
    memories, err := a.mcp.MemorySearch(ctx, &contextd.MemorySearchRequest{
        ProjectID: "contextd",
        Query: input.Task,
        Limit: 5,
    })
    if err != nil {
        return AgentOutput{}, err
    }

    // 2. Search short-lived collection for feature context
    codeContext, err := a.mcp.RepositorySearch(ctx, &contextd.SearchRequest{
        Query: input.Task,
        CollectionName: input.CollectionName,
        Limit: 10,
    })
    if err != nil {
        return AgentOutput{}, err
    }

    // 3. Generate code with LLM + context
    prompt := buildPrompt(input.Task, memories, codeContext)
    code, err := a.llmClient.GenerateCode(ctx, prompt)
    if err != nil {
        return AgentOutput{}, err
    }

    // 4. Record successful pattern
    if code.Valid {
        a.mcp.MemoryRecord(ctx, &contextd.MemoryRecordRequest{
            ProjectID: "contextd",
            Title: fmt.Sprintf("Implementation: %s", input.Task),
            Content: code.Pattern,
            Outcome: "success",
        })
    }

    return AgentOutput{
        Result: code,
        Artifacts: []Artifact{{Type: "code", Path: code.FilePath}},
    }, nil
}
```

## Quality Gates (Maximum Rigor)

All features must pass these gates before PR creation:

1. **Code Quality**
   - No linting errors
   - Code coverage ≥ 80%
   - No code smells (cyclomatic complexity ≤ 15)

2. **Testing**
   - All unit tests pass
   - All integration tests pass
   - Usage tests validate ReasoningBank integration
   - Edge cases actively discovered and tested

3. **Performance**
   - Benchmarks show no regression (≤5% slower)
   - Memory usage within bounds

4. **Security**
   - No vulnerabilities in dependencies
   - No secrets in code
   - Security best practices followed

5. **Documentation**
   - README updated
   - CHANGELOG entry added
   - Inline comments for complex logic

6. **Consensus Review**
   - 3/3 technical reviewers approve
   - 4/4 UX personas validate (no breaking changes)

**Rejection Handling**:
- Any gate failure posts detailed feedback to GitHub issue
- Workflow terminates with error
- Human reviews feedback and may restart workflow with updated issue

## Observability

### Temporal UI

**Dashboards**:
- Active workflows (in-progress features)
- Workflow history (completed/failed features)
- Activity execution times
- Retry counts and failure reasons

**Metrics**:
- Time per phase (analysis, implementation, QA, review)
- Success rate by feature complexity
- Common failure points

### Contextd Metrics

**ReasoningBank**:
- Memory retrieval success rate
- Memory confidence scores over time
- Most-used patterns

**Short-lived Collections**:
- Active collection count
- Collection size distribution
- Cleanup success rate

## Configuration

### Environment Variables

```bash
# Temporal
TEMPORAL_HOST=localhost:7233
TEMPORAL_NAMESPACE=default

# Contextd MCP
CONTEXTD_MCP_URL=http://localhost:8080
CONTEXTD_MCP_TIMEOUT=30s

# GitHub
GITHUB_TOKEN=ghp_xxx
GITHUB_WEBHOOK_SECRET=xxx

# LLM
ANTHROPIC_API_KEY=sk-ant-xxx

# Feature Flags
ALLOW_PARALLEL_FEATURES=false
MAX_PARALLEL_FEATURES=1
```

### Workflow Configuration

```yaml
# config/workflow.yaml
analysis:
  timeout: 30m
  retry_policy:
    max_attempts: 3

implementation:
  timeout: 6h
  retry_policy:
    max_attempts: 2

quality:
  timeout: 2h
  coverage_threshold: 0.80
  benchmark_regression_threshold: 0.05

review:
  timeout: 40m
  consensus_required: true
  personas:
    - marcus  # Backend dev
    - sarah   # Frontend dev
    - alex    # Full stack
    - jordan  # DevOps
```

## Deployment

### Prerequisites

1. **Temporal Server**: Running on cluster (user has this)
2. **Contextd MCP Server**: Running with HTTP transport
3. **GitHub Webhook**: Configured for issue events
4. **Anthropic API Key**: For LLM agents

### Worker Deployment

```bash
# Build worker
go build -o worker cmd/worker/main.go

# Run worker (connects to Temporal)
./worker \
  --temporal-host localhost:7233 \
  --contextd-url http://localhost:8080 \
  --github-token $GITHUB_TOKEN \
  --anthropic-key $ANTHROPIC_API_KEY
```

**Scaling**:
- Deploy multiple workers for parallelism
- Temporal load-balances activities across workers
- Each worker maintains own Contextd MCP connection

## Future Enhancements

1. **Multi-repository Support**: Coordinate changes across repos
2. **Dependency Updates**: Auto-update dependencies with tests
3. **Performance Optimization**: Auto-profile and optimize slow code
4. **Documentation Generation**: Auto-generate API docs from code
5. **Release Management**: Auto-tag releases, generate release notes

## Success Criteria

**Feature Development**:
- Issue → PR in <24 hours for medium complexity
- 90%+ test coverage
- Zero security vulnerabilities
- Consensus review pass rate >80%

**Learning**:
- ReasoningBank memory reuse rate >50%
- Pattern retrieval improves code quality
- Edge case discovery rate increases over time

**Reliability**:
- Workflow crash recovery <1 minute
- Activity retry success rate >95%
- Zero data loss during failures

## References

- Temporal Documentation: https://docs.temporal.io
- Contextd MCP Spec: `/var/home/dahendel/projects/contextd/docs/spec/MCP.md`
- Persona Simulation: `/var/home/dahendel/projects/contextd/test/docs/PERSONA-SIMULATION-METHODOLOGY.md`
- ReasoningBank Design: `/var/home/dahendel/projects/contextd/docs/spec/REASONING_BANK.md`
