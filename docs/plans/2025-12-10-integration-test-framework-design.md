# Contextd Integration Test Framework Design

**Status**: Draft
**Created**: 2025-12-10
**Purpose**: Validate contextd's core value proposition through comprehensive integration testing

---

## Executive Summary

Contextd's value hinges on one question: *Does knowledge recorded by one developer actually help another?*

This design describes a Temporal-based integration test framework that validates:
1. **Policy Compliance** - Learnings get followed when starting fresh work
2. **Bug-Fix Learning** - Dev B benefits from Dev A's recorded fixes
3. **Multi-Session Continuity** - Checkpoint/resume preserves working context

The framework uses real infrastructure (Temporal, Qdrant, LLM on Kubernetes) to ensure tests reflect production behavior.

---

## Architecture

### Infrastructure (All on Kubernetes)

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Temporal   │  │    Qdrant    │  │   Ollama / LLM       │  │
│  │   Server     │  │   (Shared)   │  │   (External Model)   │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
           │                  │                    │
           ▼                  ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Test Harness (Local / CI)                     │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Temporal Workflow Orchestrator              │   │
│  │                                                          │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌──────────────────┐  │   │
│  │  │   Suite A   │ │   Suite C   │ │     Suite D      │  │   │
│  │  │   Policy    │ │   Bug-Fix   │ │   Multi-Session  │  │   │
│  │  │ Compliance  │ │  Learning   │ │  Feature Build   │  │   │
│  │  └─────────────┘ └─────────────┘ └──────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Developer Simulators                    │   │
│  │                                                          │   │
│  │  ┌─────────────────┐        ┌─────────────────┐        │   │
│  │  │   Dev A         │        │   Dev B         │        │   │
│  │  │   Contextd      │        │   Contextd      │        │   │
│  │  │   (MCP Server)  │        │   (MCP Server)  │        │   │
│  │  │   tenant: dev-a │        │   tenant: dev-b │        │   │
│  │  └─────────────────┘        └─────────────────┘        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │  GitHub Private Test Repo     │
              │  fyrsmithlabs/contextd-test   │
              └───────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| Temporal Server | Orchestrate test workflows, provide durability, history, and retry logic |
| Qdrant | Shared vector store for team knowledge simulation |
| Ollama/LLM | External model for agent reasoning during tests |
| Test Harness | Go test framework, Temporal workers, developer simulators |
| Dev A/B Contextd | Separate MCP server instances per simulated developer |
| Test Repo | Real GitHub private repo for realistic git workflows |

### Developer Simulation Model

Each simulated developer has:
- **Own contextd instance** - Separate MCP server process
- **Own tenant ID** - `dev-a`, `dev-b`, etc.
- **Shared project collection** - Same Qdrant collection for team knowledge
- **Session metadata** - Tracks who learned what, when, for provenance

This enables testing:
- Dev A records fix → Dev B retrieves it (cross-developer knowledge)
- Dev A's memory stays isolated when appropriate (tenant boundaries)
- Team-wide queries find relevant knowledge regardless of who recorded it

---

## Test Suites

### Suite A: Policy Compliance

**Purpose**: Validate that recorded policies/learnings are followed when starting fresh work.

#### Test A.1: TDD Policy Enforcement

```yaml
name: tdd_policy_enforcement
description: Agent follows recorded TDD policy on new feature

setup:
  - record_memory:
      title: "Always use TDD"
      content: "When implementing new features, always write a failing test first, then implement the minimum code to pass, then refactor. Never write implementation before tests."
      outcome: success
      tags: [policy, tdd, development-practice]

scenario:
  - prompt: "Add a new function to calculate fibonacci numbers in the math package"

assertions:
  - type: behavioral
    check: "Test file created before implementation file"
    method: git_commit_order
  - type: binary
    check: "memory_search was called"
  - type: threshold
    check: "TDD policy memory retrieved with confidence > 0.7"
```

#### Test A.2: Conventional Commits Policy

```yaml
name: conventional_commits_policy
description: Agent uses conventional commit format after policy recorded

setup:
  - record_memory:
      title: "Use conventional commits"
      content: "All commit messages must follow conventional commits format: type(scope): description. Types: feat, fix, docs, refactor, test, chore."
      outcome: success
      tags: [policy, git, commits]

scenario:
  - prompt: "Fix the bug in the validation function and commit the change"

assertions:
  - type: behavioral
    check: "Commit message matches conventional format"
    method: regex_match
    pattern: "^(feat|fix|docs|refactor|test|chore)(\\(.+\\))?: .+"
  - type: binary
    check: "memory_search called before commit"
```

#### Test A.3: No Secrets Policy (Commit Prevention)

```yaml
name: no_secrets_commit_policy
description: Agent refuses to commit secrets after policy recorded

setup:
  - record_memory:
      title: "Never commit secrets"
      content: "Never commit .env files, API keys, passwords, or any credentials to git. If a .env file exists, ensure it is in .gitignore."
      outcome: success
      tags: [policy, security, secrets]
  - create_file:
      path: ".env"
      content: "API_KEY=sk-secret-key-12345"

scenario:
  - prompt: "Add all files and commit the current changes"

assertions:
  - type: behavioral
    check: ".env file not in commit"
    method: git_diff_check
  - type: binary
    check: "Agent either refused or added .env to .gitignore"
```

#### Test A.4: Secret Scrubbing (Core Functionality)

```yaml
name: secret_scrubbing_before_storage
description: Secrets are scrubbed BEFORE reaching vector store or logs

setup:
  - create_file:
      path: "config/database.go"
      content: |
        package config
        const DBPassword = "super-secret-password-123"
        const APIKey = "sk-ant-api03-realkey123456"
        const AWSSecret = "AKIAIOSFODNN7EXAMPLE"

scenario:
  - prompt: "Record a memory about the database configuration patterns in this project"
  - memory_record:
      title: "Database configuration pattern"
      content: "The project uses config/database.go with DBPassword, APIKey, and AWSSecret constants"

assertions:
  # Binary: Scrubbing happened
  - type: binary
    check: "gitleaks scrubber was invoked"
    method: trace_check
    span: "secrets.Scrub"

  # Behavioral: Secrets NOT in stored memory
  - type: behavioral
    check: "Stored memory content does not contain actual secret values"
    method: vector_store_content_check
    negative_patterns:
      - "super-secret-password-123"
      - "sk-ant-api03-realkey123456"
      - "AKIAIOSFODNN7EXAMPLE"

  # Behavioral: Secrets replaced with redaction markers
  - type: behavioral
    check: "Secrets replaced with [REDACTED] or similar marker"
    method: vector_store_content_check
    patterns:
      - "[REDACTED]"
      - "\\*\\*\\*"

  # Binary: Secrets NOT in logs
  - type: binary
    check: "Secrets do not appear in contextd logs"
    method: log_content_check
    negative_patterns:
      - "super-secret-password-123"
      - "sk-ant-api03-realkey123456"

  # Binary: Secrets NOT in OTEL traces
  - type: binary
    check: "Secrets do not appear in trace attributes"
    method: trace_attribute_check
    negative_patterns:
      - "super-secret-password-123"
      - "sk-ant-api03-realkey123456"
```

#### Test A.5: Secret Scrubbing in Search Results

```yaml
name: secret_scrubbing_in_retrieval
description: Secrets scrubbed from search results even if they somehow got stored

setup:
  # Simulate a memory that somehow contains a secret (defense in depth test)
  - direct_vector_insert:
      collection: "test-project"
      content: "Use connection string: postgres://user:password123@localhost:5432/db"
      metadata:
        title: "Database connection"

scenario:
  - prompt: "Search for database connection information"
  - memory_search:
      query: "database connection string"

assertions:
  - type: behavioral
    check: "Search results have secrets scrubbed"
    method: search_response_check
    negative_patterns:
      - "password123"
  - type: behavioral
    check: "Connection pattern visible but credentials redacted"
    method: search_response_check
    patterns:
      - "postgres://user:[REDACTED]@localhost"
```

#### Test A.6: Secret Scrubbing Failure Detection (Known Failure)

```yaml
name: secret_scrubbing_bypass_failure
description: EXPECTED TO FAIL - Detect if scrubbing is bypassed
expect_failure: true

setup:
  - disable_scrubbing: true  # Intentionally disable for this test
  - create_file:
      path: ".env"
      content: "SECRET_KEY=this-should-be-caught"

scenario:
  - memory_record:
      title: "Test record"
      content: "Config uses SECRET_KEY=this-should-be-caught from .env"

assertions:
  - type: binary
    check: "Secret detected in stored content (scrubbing was bypassed)"
    method: vector_store_content_check
    patterns:
      - "this-should-be-caught"
  - type: binary
    check: "Alert/metric emitted for scrubbing bypass"
    method: metrics_check
    metric: "contextd_secret_scrub_bypass_total"
```

#### Test A.7: Code Review Policy

```yaml
name: code_review_policy
description: Agent runs linter before creating PR

setup:
  - record_memory:
      title: "Run linter before PR"
      content: "Always run golangci-lint and fix any issues before creating a pull request. Do not create PRs with linting errors."
      outcome: success
      tags: [policy, code-review, linting]

scenario:
  - prompt: "Create a PR for the current changes"

assertions:
  - type: behavioral
    check: "golangci-lint was executed"
    method: command_history_check
  - type: binary
    check: "PR created only after lint passes"
```

---

### Suite C: Bug-Fix Learning

**Purpose**: Validate that Dev B benefits from Dev A's recorded fixes.

#### Test C.1: Same Bug Retrieval

```yaml
name: same_bug_retrieval
description: Dev B retrieves and applies Dev A's fix for identical bug

dev_a_session:
  - introduce_bug:
      file: "pkg/validator/validator.go"
      bug_type: null_pointer_dereference
      description: "Accessing field on nil struct"
  - prompt: "Fix the panic in the validator package"
  - wait_for_fix: true
  - assert_remediation_recorded:
      title_contains: "null pointer"

dev_b_session:
  - introduce_bug:
      file: "pkg/parser/parser.go"
      bug_type: null_pointer_dereference
      description: "Same pattern - accessing field on nil struct"
  - prompt: "Fix the panic in the parser package"

assertions:
  - type: binary
    check: "Dev B's contextd called remediation_search"
  - type: threshold
    check: "Dev A's remediation retrieved with confidence > 0.6"
  - type: behavioral
    check: "Dev B applied similar fix pattern (nil check before access)"
    method: ast_pattern_match
```

#### Test C.2: Similar Bug Adaptation

```yaml
name: similar_bug_adaptation
description: Dev B adapts Dev A's fix pattern to different context

dev_a_session:
  - introduce_bug:
      file: "pkg/auth/session.go"
      bug_type: race_condition
      description: "Concurrent map access without mutex"
  - prompt: "Fix the race condition in the auth package"
  - wait_for_fix: true
  - assert_remediation_recorded:
      title_contains: "race condition"
      content_contains: "sync.RWMutex"

dev_b_session:
  - introduce_bug:
      file: "pkg/cache/store.go"
      bug_type: race_condition
      description: "Different concurrent map access without mutex"
  - prompt: "Fix the race condition in the cache package"

assertions:
  - type: binary
    check: "Dev B's contextd called remediation_search"
  - type: behavioral
    check: "Dev B used mutex pattern from Dev A's fix"
    method: ast_pattern_match
    pattern: "sync.RWMutex or sync.Mutex usage"
  - type: threshold
    check: "Retrieval latency < 500ms"
```

#### Test C.3: False Positive Prevention

```yaml
name: false_positive_prevention
description: Dev B does NOT incorrectly apply Dev A's unrelated fix

dev_a_session:
  - introduce_bug:
      file: "pkg/auth/session.go"
      bug_type: race_condition
      description: "Concurrent map access"
  - prompt: "Fix the race condition"
  - wait_for_fix: true

dev_b_session:
  - introduce_bug:
      file: "pkg/http/handler.go"
      bug_type: missing_error_handling
      description: "Ignoring error return value"
  - prompt: "Fix the bug in the HTTP handler"

assertions:
  - type: behavioral
    check: "Dev B did NOT add mutex (wrong fix for this bug)"
    method: ast_negative_match
    pattern: "sync.Mutex added to http/handler.go"
  - type: behavioral
    check: "Dev B added proper error handling"
    method: ast_pattern_match
    pattern: "if err != nil"
```

#### Test C.4: Confidence Decay on Negative Feedback

```yaml
name: confidence_decay_negative_feedback
description: Memory confidence decreases when fix doesn't help

dev_a_session:
  - record_memory:
      title: "Fix for timeout errors"
      content: "Increase timeout to 30 seconds"
      outcome: success

dev_b_session:
  - prompt: "Fix the timeout error in the API client"
  - retrieve_memory: "Fix for timeout errors"
  - apply_suggested_fix: true
  - report_outcome:
      succeeded: false
      description: "Increasing timeout did not fix the issue, root cause was connection pooling"

assertions:
  - type: threshold
    check: "Memory confidence decreased by at least 0.1"
    method: confidence_delta
  - type: binary
    check: "memory_feedback called with helpful=false"
```

---

### Suite D: Multi-Session Feature Build

**Purpose**: Validate that checkpoint/resume preserves working context across sessions.

#### Test D.1: Clean Resume

```yaml
name: clean_resume
description: Agent resumes feature work without re-discovering context

session_1:
  - prompt: "Start implementing user authentication feature. Begin with the data models."
  - wait_for_progress:
      files_created: ["pkg/auth/models.go"]
  - checkpoint_save:
      summary: "Auth feature: models complete, starting handlers next"
  - clear_context: true

session_2:
  - checkpoint_resume: "latest"
  - prompt: "Continue working on the authentication feature"

assertions:
  - type: binary
    check: "checkpoint_resume was called"
  - type: behavioral
    check: "Agent continued with handlers, did not recreate models"
    method: git_diff_check
    expect_no_changes: ["pkg/auth/models.go"]
  - type: behavioral
    check: "Agent knew models were complete without re-reading"
    method: prompt_analysis
    negative_patterns: ["let me check what we have", "let me review the models"]
```

#### Test D.2: Stale Resume Detection

```yaml
name: stale_resume_detection
description: Agent detects and reconciles external changes after resume

session_1:
  - prompt: "Implement the login endpoint"
  - wait_for_progress:
      files_created: ["pkg/auth/login.go"]
  - checkpoint_save:
      summary: "Login endpoint complete"
  - clear_context: true

external_changes:
  - git_commit:
      message: "refactor: rename auth package to authentication"
      changes:
        - rename: "pkg/auth" -> "pkg/authentication"

session_2:
  - checkpoint_resume: "latest"
  - prompt: "Continue with the logout endpoint"

assertions:
  - type: behavioral
    check: "Agent detected package was renamed"
    method: prompt_analysis
    patterns: ["package renamed", "authentication instead of auth", "reconcile"]
  - type: behavioral
    check: "Agent created logout.go in correct location"
    method: file_exists
    path: "pkg/authentication/logout.go"
```

#### Test D.3: Partial Work Resume

```yaml
name: partial_work_resume
description: Agent knows which tasks are complete and continues with remaining

session_1:
  - prompt: |
      Implement user registration with these 5 tasks:
      1. Create User model
      2. Create UserRepository interface
      3. Implement PostgresUserRepository
      4. Create RegisterUserHandler
      5. Add input validation
  - wait_for_progress:
      tasks_complete: [1, 2]
  - checkpoint_save:
      summary: "Registration: User model and repository interface done. Next: Postgres implementation."
  - clear_context: true

session_2:
  - checkpoint_resume: "latest"
  - prompt: "Continue with the user registration feature"

assertions:
  - type: behavioral
    check: "Agent started with task 3 (PostgresUserRepository)"
    method: first_file_created
    expect: "repository implementation, not model or interface"
  - type: binary
    check: "Agent did not recreate User model"
  - type: threshold
    check: "All 5 tasks completed by end of session 2"
```

#### Test D.4: Cross-Session Memory Accumulation

```yaml
name: cross_session_memory_accumulation
description: Learnings from session 1 are available and useful in session 2

session_1:
  - prompt: "Set up the database connection with proper error handling"
  - wait_for_completion: true
  - record_memory:
      title: "Database connection pattern"
      content: "Use connection pooling with max 10 connections, always defer Close(), wrap errors with context"
      outcome: success
  - checkpoint_save:
      summary: "Database setup complete with connection pooling"
  - clear_context: true

session_2:
  - checkpoint_resume: "latest"
  - prompt: "Now add a Redis cache connection using similar patterns"

assertions:
  - type: binary
    check: "memory_search called for connection patterns"
  - type: behavioral
    check: "Redis connection uses similar patterns (pooling, defer Close, error wrapping)"
    method: ast_pattern_match
```

---

### Known Failure Tests

These tests SHOULD fail to verify our system catches bad behavior.

#### Failure F.1: Memory Degradation Below Threshold

```yaml
name: memory_degradation_failure
description: EXPECTED TO FAIL - Memory becomes unusable after repeated negative feedback
expect_failure: true

setup:
  - record_memory:
      title: "Outdated fix pattern"
      content: "Use deprecated API for file handling"
      outcome: success
      initial_confidence: 0.8

scenario:
  - repeat: 5
    action:
      - retrieve_memory: "Outdated fix pattern"
      - report_outcome:
          succeeded: false
          description: "Fix did not work, API is deprecated"

assertions:
  - type: threshold
    check: "Memory confidence below 0.3 (unusable threshold)"
  - type: behavioral
    check: "Memory no longer returned in search results"
    method: search_results_check
    expect_absent: "Outdated fix pattern"
```

#### Failure F.2: Policy Violation Detection

```yaml
name: policy_violation_detection
description: EXPECTED TO FAIL - Agent violates recorded policy
expect_failure: true

setup:
  - record_memory:
      title: "Always use TDD"
      content: "Write tests before implementation"
      outcome: success

scenario:
  - prompt: "Quickly add a utility function to format dates"
  - force_behavior: "Skip tests, write implementation only"  # Simulated bad behavior

assertions:
  - type: behavioral
    check: "Implementation written before tests"
    method: git_commit_order
    expect: "violation detected"
```

#### Failure F.3: Checkpoint Corruption

```yaml
name: checkpoint_corruption_handling
description: EXPECTED TO FAIL - Resume from corrupted checkpoint
expect_failure: true

session_1:
  - prompt: "Start feature work"
  - checkpoint_save:
      summary: "Feature in progress"

corruption:
  - corrupt_checkpoint:
      method: "truncate metadata"

session_2:
  - checkpoint_resume: "latest"

assertions:
  - type: binary
    check: "Corruption detected and reported"
  - type: behavioral
    check: "Agent did not proceed with corrupted state"
```

#### Failure F.4: False Retrieval Application

```yaml
name: false_retrieval_application
description: EXPECTED TO FAIL - Agent applies retrieved memory incorrectly
expect_failure: true

setup:
  - record_memory:
      title: "Database connection fix"
      content: "Add retry logic with exponential backoff"
      outcome: success
      tags: [database, connection, retry]

scenario:
  - prompt: "Fix the HTTP client timeout issue"
  - force_behavior: "Apply database retry pattern to HTTP client incorrectly"

assertions:
  - type: behavioral
    check: "Wrong pattern applied to wrong context"
    method: semantic_mismatch_detection
```

---

## Observability

### OpenTelemetry Integration

All test workflows emit traces and metrics to the existing contextd OTEL infrastructure.

#### Traces

```
test_workflow
├── suite_execution (suite=policy_compliance)
│   ├── test_execution (test=tdd_policy_enforcement)
│   │   ├── setup_phase
│   │   │   └── record_memory (memory_id=xxx, confidence=0.8)
│   │   ├── scenario_phase
│   │   │   ├── llm_prompt (model=ollama/qwen2.5, tokens=1234)
│   │   │   ├── contextd_tool_call (tool=memory_search, results=3)
│   │   │   └── agent_action (action=create_file, path=pkg/math/fib_test.go)
│   │   └── assertion_phase
│   │       ├── assertion_check (type=behavioral, passed=true)
│   │       ├── assertion_check (type=binary, passed=true)
│   │       └── assertion_check (type=threshold, passed=true, value=0.85)
```

#### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `contextd_test_suite_duration_seconds` | Histogram | Time to complete each test suite |
| `contextd_test_pass_total` | Counter | Number of passed tests by suite |
| `contextd_test_fail_total` | Counter | Number of failed tests by suite |
| `contextd_memory_retrieval_latency_ms` | Histogram | Time to retrieve relevant memories |
| `contextd_memory_confidence_score` | Gauge | Confidence score at retrieval time |
| `contextd_memory_hit_rate` | Gauge | Percentage of searches that return useful results |
| `contextd_checkpoint_resume_success_rate` | Gauge | Percentage of successful checkpoint resumes |
| `contextd_cross_developer_retrieval_rate` | Gauge | How often Dev B retrieves Dev A's knowledge |
| `contextd_policy_compliance_rate` | Gauge | How often recorded policies are followed |

#### Dashboards (Grafana)

1. **Test Health Dashboard**
   - Test pass/fail rates over time
   - Flaky test detection
   - Duration trends

2. **Memory Effectiveness Dashboard**
   - Confidence score distributions
   - Retrieval latency percentiles
   - Hit/miss rates by memory type

3. **Cross-Developer Knowledge Flow**
   - Knowledge propagation from Dev A to Dev B
   - Time-to-useful-recall
   - False positive rates

4. **Checkpoint Reliability Dashboard**
   - Resume success rates
   - Staleness detection rates
   - Corruption incidents

---

## Test Execution

### Directory Structure

```
test/
├── integration/
│   ├── framework/
│   │   ├── workflow.go          # Temporal workflow definitions
│   │   ├── activities.go        # Temporal activities (LLM calls, contextd ops)
│   │   ├── assertions.go        # Assertion implementations
│   │   ├── developer.go         # Developer simulator
│   │   └── metrics.go           # OTEL metrics setup
│   ├── suites/
│   │   ├── policy_compliance_test.go
│   │   ├── bugfix_learning_test.go
│   │   └── multisession_test.go
│   ├── scenarios/
│   │   ├── policy/              # Policy compliance YAML scenarios
│   │   ├── bugfix/              # Bug-fix learning YAML scenarios
│   │   └── multisession/        # Multi-session YAML scenarios
│   └── testdata/
│       ├── bugs/                # Intentional bugs to introduce
│       └── fixtures/            # Test fixtures
```

### Makefile Targets

```makefile
# Run all integration tests
test-integration: test-integration-policy test-integration-bugfix test-integration-multisession

# Run individual suites
test-integration-policy:
	go test -v -tags=integration ./test/integration/suites/... -run TestPolicyCompliance

test-integration-bugfix:
	go test -v -tags=integration ./test/integration/suites/... -run TestBugfixLearning

test-integration-multisession:
	go test -v -tags=integration ./test/integration/suites/... -run TestMultiSession

# Run with specific backend
test-integration-chromem:
	CONTEXTD_VECTORSTORE=chromem go test -v -tags=integration ./test/integration/...

test-integration-qdrant:
	CONTEXTD_VECTORSTORE=qdrant QDRANT_URL=$(QDRANT_URL) go test -v -tags=integration ./test/integration/...

# Run known failure tests
test-integration-failures:
	go test -v -tags=integration,expect_failure ./test/integration/...

# Health checks
test-integration-preflight:
	@echo "Checking Temporal..."
	temporal workflow list --namespace contextd-test || (echo "Temporal not available" && exit 1)
	@echo "Checking Qdrant..."
	curl -s $(QDRANT_URL)/health || (echo "Qdrant not available" && exit 1)
	@echo "Checking LLM..."
	curl -s $(LLM_URL)/api/tags || (echo "LLM not available" && exit 1)
	@echo "All services available"
```

### Configuration

```yaml
# test/integration/config.yaml
temporal:
  address: "temporal.contextd.svc.cluster.local:7233"
  namespace: "contextd-test"
  task_queue: "integration-tests"

qdrant:
  url: "http://qdrant.contextd.svc.cluster.local:6333"
  collection_prefix: "test_"

llm:
  provider: "ollama"
  url: "http://ollama.contextd.svc.cluster.local:11434"
  model: "qwen2.5:14b"  # Or whatever model is deployed

claude:
  enabled: false  # Enable for final validation runs

test_repo:
  url: "git@github.com:fyrsmithlabs/contextd-test.git"
  branch_prefix: "test/"
  cleanup: true  # Delete test branches after run

observability:
  otel_endpoint: "http://otel-collector.observability.svc.cluster.local:4317"
  metrics_port: 9090

developers:
  - id: "dev-a"
    tenant: "test-tenant-a"
  - id: "dev-b"
    tenant: "test-tenant-b"
```

---

## Temporal Workflows

### Main Test Orchestrator Workflow

```go
// TestOrchestratorWorkflow coordinates all test suites
func TestOrchestratorWorkflow(ctx workflow.Context, config TestConfig) (*TestReport, error) {
    // Run suites (can be parallel or sequential based on config)
    var futures []workflow.Future

    if config.RunPolicy {
        f := workflow.ExecuteChildWorkflow(ctx, PolicyComplianceWorkflow, config)
        futures = append(futures, f)
    }

    if config.RunBugfix {
        f := workflow.ExecuteChildWorkflow(ctx, BugfixLearningWorkflow, config)
        futures = append(futures, f)
    }

    if config.RunMultiSession {
        f := workflow.ExecuteChildWorkflow(ctx, MultiSessionWorkflow, config)
        futures = append(futures, f)
    }

    // Collect results
    report := &TestReport{}
    for _, f := range futures {
        var result SuiteResult
        if err := f.Get(ctx, &result); err != nil {
            report.Errors = append(report.Errors, err.Error())
        }
        report.Suites = append(report.Suites, result)
    }

    return report, nil
}
```

### Developer Session Workflow

```go
// DeveloperSessionWorkflow simulates a developer using contextd
func DeveloperSessionWorkflow(ctx workflow.Context, session SessionConfig) (*SessionResult, error) {
    // Start contextd MCP server for this developer
    var contextdHandle ContextdHandle
    err := workflow.ExecuteActivity(ctx, StartContextdActivity, session.Developer).Get(ctx, &contextdHandle)
    if err != nil {
        return nil, err
    }
    defer workflow.ExecuteActivity(ctx, StopContextdActivity, contextdHandle)

    // Execute scenario steps
    result := &SessionResult{Developer: session.Developer}

    for _, step := range session.Steps {
        switch step.Type {
        case "prompt":
            var response LLMResponse
            err := workflow.ExecuteActivity(ctx, LLMPromptActivity, LLMPromptInput{
                Developer: session.Developer,
                Contextd:  contextdHandle,
                Prompt:    step.Prompt,
            }).Get(ctx, &response)
            if err != nil {
                result.Errors = append(result.Errors, err.Error())
            }
            result.Responses = append(result.Responses, response)

        case "checkpoint_save":
            err := workflow.ExecuteActivity(ctx, CheckpointSaveActivity, contextdHandle, step.Summary).Get(ctx, nil)
            if err != nil {
                result.Errors = append(result.Errors, err.Error())
            }

        case "checkpoint_resume":
            err := workflow.ExecuteActivity(ctx, CheckpointResumeActivity, contextdHandle, step.CheckpointID).Get(ctx, nil)
            if err != nil {
                result.Errors = append(result.Errors, err.Error())
            }

        case "clear_context":
            // Simulates /clear in Claude Code
            err := workflow.ExecuteActivity(ctx, ClearContextActivity, contextdHandle).Get(ctx, nil)
            if err != nil {
                result.Errors = append(result.Errors, err.Error())
            }
        }
    }

    return result, nil
}
```

---

## Assertion System

### Three-Level Assertions

```go
// Level 1: Binary (pass/fail)
type BinaryAssertion struct {
    Check    string // What to check
    Method   string // How to check (tool_called, file_exists, etc.)
    Target   string // Specific target (tool name, file path, etc.)
}

// Level 2: Threshold
type ThresholdAssertion struct {
    Check     string  // What to check
    Method    string  // How to measure
    Threshold float64 // Minimum/maximum value
    Operator  string  // >, <, >=, <=, ==
}

// Level 3: Behavioral (LLM-as-judge for complex checks)
type BehavioralAssertion struct {
    Check           string   // What to check
    Method          string   // AST analysis, semantic match, etc.
    Patterns        []string // Positive patterns to find
    NegativePatterns []string // Patterns that should NOT appear
    LLMJudge        bool     // Use LLM to evaluate if automated check insufficient
}
```

### Assertion Methods

| Method | Type | Description |
|--------|------|-------------|
| `tool_called` | Binary | Check if specific contextd tool was invoked |
| `file_exists` | Binary | Check if file was created |
| `git_commit_order` | Behavioral | Verify order of file creation in commits |
| `regex_match` | Behavioral | Match output against regex pattern |
| `ast_pattern_match` | Behavioral | Check for code patterns in AST |
| `ast_negative_match` | Behavioral | Ensure code patterns are NOT present |
| `confidence_delta` | Threshold | Measure change in confidence score |
| `latency_check` | Threshold | Verify operation completed within time limit |
| `search_results_check` | Binary/Behavioral | Verify search returned expected results |
| `llm_judge` | Behavioral | Use LLM to evaluate complex behavioral criteria |

---

## Implementation Phases

### Phase 1: Framework Foundation
- [ ] Temporal workflow definitions
- [ ] Activity implementations (LLM, contextd, git)
- [ ] Developer simulator
- [ ] Basic assertion system
- [ ] Configuration loading
- [ ] Makefile targets

### Phase 2: Suite A (Policy Compliance)
- [ ] TDD policy test
- [ ] Conventional commits test
- [ ] No secrets test
- [ ] Code review policy test
- [ ] Policy violation failure test

### Phase 3: Suite C (Bug-Fix Learning)
- [ ] Same bug retrieval test
- [ ] Similar bug adaptation test
- [ ] False positive prevention test
- [ ] Confidence decay test
- [ ] False retrieval failure test

### Phase 4: Suite D (Multi-Session)
- [ ] Clean resume test
- [ ] Stale resume detection test
- [ ] Partial work resume test
- [ ] Cross-session memory accumulation test
- [ ] Checkpoint corruption failure test

### Phase 5: Observability
- [ ] OTEL trace instrumentation
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Test report generation

### Phase 6: CI Integration
- [ ] GitHub Actions workflow
- [ ] Kubernetes job for test execution
- [ ] Slack/Discord notifications
- [ ] Badge generation

---

## Success Criteria

The integration test framework is complete when:

1. **All three suites pass** against real infrastructure (Temporal, Qdrant, Ollama on K8s)
2. **Known failure tests fail as expected** and report clear diagnostics
3. **Cross-developer knowledge flow** is validated (Dev A's fix helps Dev B)
4. **Checkpoint/resume** preserves context accurately
5. **Metrics show** >80% memory retrieval hit rate, >90% checkpoint resume success
6. **Dashboards** provide visibility into knowledge propagation effectiveness
7. **Tests complete** within reasonable time (<10 min for full suite)
8. **Claude CLI validation** passes for final validation runs

---

## Open Questions

1. **Test isolation**: Should each test run get a fresh Qdrant collection, or should we test accumulation effects?
2. **LLM determinism**: How do we handle non-deterministic LLM responses in assertions?
3. **Test repo cleanup**: Automatic branch deletion, or keep for debugging?
4. **Flaky test handling**: Retry policy for transient failures?

---

## References

- [Temporal Go SDK](https://docs.temporal.io/dev-guide/go)
- [contextd MCP Tools](../internal/mcp/tools.go)
- [Bayesian Confidence System](../internal/reasoningbank/confidence.go)
- [Issue #20: Go Agent Orchestrator](https://github.com/fyrsmithlabs/contextd/issues/20)
