# Research: Temporal for Testing fyrsmithlabs/contextd

**Type**: Research / Enhancement Proposal
**Priority**: Medium
**Labels**: `research`, `testing`, `infrastructure`, `enhancement`

---

## Summary

Investigate using [Temporal.io](https://temporal.io/) as a durable workflow orchestration layer for contextd's CI/CD and testing infrastructure. Temporal provides reliable execution guarantees, automatic retries, and observability that could replace or augment GitHub Actions for complex testing scenarios.

---

## Background

### What is Temporal?

Temporal is a durable execution platform used by companies like Stripe, Netflix, Datadog, and HashiCorp. It provides:

- **Durable Execution**: Workflows survive failures and can run for seconds to years
- **Automatic Retries**: Built-in failure handling with configurable retry policies
- **Time Skipping**: Test long-running workflows in seconds using in-memory simulation
- **Complete Observability**: Full audit trail and debugging via workflow history
- **Native Go SDK**: First-class Go support with comprehensive testing framework

### Why Consider Temporal for contextd?

Current GitHub Actions limitations:
1. **No dry-run capability** - Dangerous for workflows with side effects
2. **Timeout constraints** - Long-running tests can fail silently
3. **Limited orchestration** - Complex multi-step pipelines are hard to manage
4. **Flat logs** - Debugging failures requires digging through unstructured output
5. **No built-in retry logic** - Manual retry handling needed

Temporal addresses all of these with:
- Workflow replay for safe testing
- Unlimited execution duration with heartbeats
- Declarative workflow composition
- Structured event history for debugging
- Configurable retry policies per activity

---

## Proposed Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    GitHub (Trigger Layer)                        │
│  - Push/PR events trigger Temporal workflows                     │
│  - Minimal GHA: just webhook to Temporal                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Temporal Server (Orchestration)                │
│  - Self-hosted via docker-compose                               │
│  - Manages workflow state and retries                           │
│  - Web UI at localhost:8080 for observability                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Temporal Workers (Execution)                   │
│  - Go workers running test activities                           │
│  - Can run on multiple machines / containers                    │
│  - Worker-specific task queues for platform tests               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Example: Replace `release.yml` with Temporal Workflow

### Current GHA Flow

```yaml
# Current: GitHub Actions release.yml
jobs:
  build-linux → build-macos → release → docker
```

### Temporal Equivalent

```go
// workflows/release.go
package workflows

import (
    "time"

    "go.temporal.io/sdk/workflow"
)

type ReleaseParams struct {
    Version    string
    CommitSHA  string
    TagName    string
}

type BuildResult struct {
    Platform   string
    Artifacts  []string
    Checksums  map[string]string
}

// ReleaseWorkflow orchestrates the complete release process
func ReleaseWorkflow(ctx workflow.Context, params ReleaseParams) error {
    logger := workflow.GetLogger(ctx)
    logger.Info("Starting release workflow", "version", params.Version)

    // Configure activity options with retries
    activityOpts := workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Minute,
        HeartbeatTimeout:    5 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    time.Minute,
            MaximumAttempts:    3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, activityOpts)

    // Phase 1: Build Linux and macOS in parallel
    var linuxResult, macosAMD64Result, macosARM64Result BuildResult

    buildFutures := []workflow.Future{
        workflow.ExecuteActivity(ctx, BuildLinux, params),
        workflow.ExecuteActivity(ctx, BuildMacOS, params, "amd64"),
        workflow.ExecuteActivity(ctx, BuildMacOS, params, "arm64"),
    }

    // Collect all build results
    results := make([]BuildResult, 3)
    for i, future := range buildFutures {
        if err := future.Get(ctx, &results[i]); err != nil {
            return fmt.Errorf("build failed for platform %d: %w", i, err)
        }
    }

    // Phase 2: Create GitHub release
    var releaseURL string
    if err := workflow.ExecuteActivity(ctx, CreateGitHubRelease, params, results).Get(ctx, &releaseURL); err != nil {
        return fmt.Errorf("release creation failed: %w", err)
    }

    // Phase 3: Build and push Docker image
    if err := workflow.ExecuteActivity(ctx, BuildDockerImage, params).Get(ctx, nil); err != nil {
        return fmt.Errorf("docker build failed: %w", err)
    }

    // Phase 4: Update Homebrew formula
    if err := workflow.ExecuteActivity(ctx, UpdateHomebrew, params).Get(ctx, nil); err != nil {
        // Non-fatal: log but continue
        logger.Warn("Homebrew update failed", "error", err)
    }

    logger.Info("Release completed successfully", "url", releaseURL)
    return nil
}
```

### Activities Implementation

```go
// activities/build.go
package activities

import (
    "context"
    "os/exec"

    "go.temporal.io/sdk/activity"
)

func BuildLinux(ctx context.Context, params ReleaseParams) (BuildResult, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Building Linux binaries", "version", params.Version)

    // Report progress via heartbeats
    activity.RecordHeartbeat(ctx, "Installing ONNX Runtime...")

    // Run goreleaser
    cmd := exec.CommandContext(ctx, "goreleaser", "release",
        "--skip=publish",
        "--config", ".goreleaser-linux.yaml")

    activity.RecordHeartbeat(ctx, "Running GoReleaser...")
    if err := cmd.Run(); err != nil {
        return BuildResult{}, fmt.Errorf("goreleaser failed: %w", err)
    }

    // Collect artifacts
    artifacts, err := filepath.Glob("dist/*.tar.gz")
    if err != nil {
        return BuildResult{}, err
    }

    return BuildResult{
        Platform:  "linux",
        Artifacts: artifacts,
    }, nil
}

func BuildMacOS(ctx context.Context, params ReleaseParams, arch string) (BuildResult, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Building macOS binaries", "arch", arch)

    activity.RecordHeartbeat(ctx, fmt.Sprintf("Building for darwin/%s...", arch))

    // Set environment
    os.Setenv("GOARCH", arch)
    os.Setenv("CGO_ENABLED", "1")

    // Build binaries
    for _, binary := range []string{"contextd", "ctxd"} {
        cmd := exec.CommandContext(ctx, "go", "build",
            "-ldflags", fmt.Sprintf("-s -w -X main.version=%s", params.Version),
            "-o", fmt.Sprintf("dist/%s", binary),
            fmt.Sprintf("./cmd/%s", binary))

        if err := cmd.Run(); err != nil {
            return BuildResult{}, fmt.Errorf("build %s failed: %w", binary, err)
        }
    }

    // Create archive
    archiveName := fmt.Sprintf("contextd_%s_darwin_%s.tar.gz", params.Version, arch)

    return BuildResult{
        Platform:  fmt.Sprintf("darwin_%s", arch),
        Artifacts: []string{archiveName},
    }, nil
}
```

---

## Example: Integration Test Workflow

Replace complex GHA matrix with Temporal workflow:

```go
// workflows/integration_test.go
package workflows

import (
    "time"

    "go.temporal.io/sdk/workflow"
)

type TestParams struct {
    CommitSHA string
    Branch    string
    PRNumber  int
}

type TestResult struct {
    Package  string
    Passed   bool
    Coverage float64
    Duration time.Duration
    Output   string
}

// IntegrationTestWorkflow runs all tests with proper orchestration
func IntegrationTestWorkflow(ctx workflow.Context, params TestParams) ([]TestResult, error) {
    logger := workflow.GetLogger(ctx)

    // Fast timeout for test activities
    activityOpts := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 2, // Retry flaky tests once
        },
    }
    ctx = workflow.WithActivityOptions(ctx, activityOpts)

    // Define test packages (could be discovered dynamically)
    packages := []string{
        "./internal/secrets/...",
        "./internal/reasoningbank/...",
        "./internal/checkpoint/...",
        "./internal/remediation/...",
        "./internal/repository/...",
        "./internal/embeddings/...",
        "./internal/vectorstore/...",
        "./internal/mcp/...",
        "./internal/compression/...",
    }

    // Run tests in parallel with bounded concurrency
    var results []TestResult
    var futures []workflow.Future

    for _, pkg := range packages {
        future := workflow.ExecuteActivity(ctx, RunPackageTests, params, pkg)
        futures = append(futures, future)
    }

    // Collect results
    allPassed := true
    for i, future := range futures {
        var result TestResult
        if err := future.Get(ctx, &result); err != nil {
            result = TestResult{
                Package: packages[i],
                Passed:  false,
                Output:  err.Error(),
            }
        }
        results = append(results, result)
        if !result.Passed {
            allPassed = false
        }
    }

    // Report final status
    if !allPassed {
        logger.Error("Some tests failed", "results", results)
        return results, fmt.Errorf("test failures detected")
    }

    logger.Info("All tests passed", "total", len(results))
    return results, nil
}
```

### Activity for Running Tests

```go
// activities/test.go
package activities

func RunPackageTests(ctx context.Context, params TestParams, pkg string) (TestResult, error) {
    start := time.Now()

    activity.RecordHeartbeat(ctx, fmt.Sprintf("Testing %s...", pkg))

    cmd := exec.CommandContext(ctx, "go", "test", "-v", "-cover", "-race", pkg)
    output, err := cmd.CombinedOutput()

    result := TestResult{
        Package:  pkg,
        Duration: time.Since(start),
        Output:   string(output),
    }

    if err != nil {
        result.Passed = false
        return result, nil // Return result, not error (for reporting)
    }

    result.Passed = true
    // Parse coverage from output
    result.Coverage = parseCoverage(string(output))

    return result, nil
}
```

---

## Testing Temporal Workflows (Time Skipping)

Key advantage: test long-running workflows instantly:

```go
// workflows/release_test.go
package workflows

import (
    "testing"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
    "go.temporal.io/sdk/testsuite"
)

type ReleaseWorkflowTestSuite struct {
    suite.Suite
    testsuite.WorkflowTestSuite
    env *testsuite.TestWorkflowEnvironment
}

func (s *ReleaseWorkflowTestSuite) SetupTest() {
    s.env = s.NewTestWorkflowEnvironment()
}

func (s *ReleaseWorkflowTestSuite) AfterTest(suiteName, testName string) {
    s.env.AssertExpectations(s.T())
}

func TestReleaseWorkflowTestSuite(t *testing.T) {
    suite.Run(t, new(ReleaseWorkflowTestSuite))
}

// Test successful release
func (s *ReleaseWorkflowTestSuite) Test_ReleaseWorkflow_Success() {
    params := ReleaseParams{
        Version:   "1.0.0",
        CommitSHA: "abc123",
        TagName:   "v1.0.0",
    }

    // Mock all activities
    s.env.OnActivity(BuildLinux, mock.Anything, mock.Anything).Return(
        BuildResult{Platform: "linux", Artifacts: []string{"dist/linux.tar.gz"}}, nil)

    s.env.OnActivity(BuildMacOS, mock.Anything, mock.Anything, "amd64").Return(
        BuildResult{Platform: "darwin_amd64", Artifacts: []string{"dist/darwin_amd64.tar.gz"}}, nil)

    s.env.OnActivity(BuildMacOS, mock.Anything, mock.Anything, "arm64").Return(
        BuildResult{Platform: "darwin_arm64", Artifacts: []string{"dist/darwin_arm64.tar.gz"}}, nil)

    s.env.OnActivity(CreateGitHubRelease, mock.Anything, mock.Anything, mock.Anything).Return(
        "https://github.com/fyrsmithlabs/contextd/releases/v1.0.0", nil)

    s.env.OnActivity(BuildDockerImage, mock.Anything, mock.Anything).Return(nil)
    s.env.OnActivity(UpdateHomebrew, mock.Anything, mock.Anything).Return(nil)

    // Execute workflow
    s.env.ExecuteWorkflow(ReleaseWorkflow, params)

    // Assert completion
    s.True(s.env.IsWorkflowCompleted())
    s.NoError(s.env.GetWorkflowError())
}

// Test retry behavior on build failure
func (s *ReleaseWorkflowTestSuite) Test_ReleaseWorkflow_BuildRetry() {
    params := ReleaseParams{Version: "1.0.0"}

    // First call fails, second succeeds
    s.env.OnActivity(BuildLinux, mock.Anything, mock.Anything).Return(
        BuildResult{}, errors.New("network timeout")).Once()
    s.env.OnActivity(BuildLinux, mock.Anything, mock.Anything).Return(
        BuildResult{Platform: "linux"}, nil).Once()

    // ... other mocks

    s.env.ExecuteWorkflow(ReleaseWorkflow, params)
    s.True(s.env.IsWorkflowCompleted())
    s.NoError(s.env.GetWorkflowError())
}

// Test partial failure handling
func (s *ReleaseWorkflowTestSuite) Test_ReleaseWorkflow_HomebrewFailure_NonFatal() {
    params := ReleaseParams{Version: "1.0.0"}

    // All builds succeed
    s.env.OnActivity(BuildLinux, mock.Anything, mock.Anything).Return(BuildResult{}, nil)
    s.env.OnActivity(BuildMacOS, mock.Anything, mock.Anything, mock.Anything).Return(BuildResult{}, nil)
    s.env.OnActivity(CreateGitHubRelease, mock.Anything, mock.Anything, mock.Anything).Return("url", nil)
    s.env.OnActivity(BuildDockerImage, mock.Anything, mock.Anything).Return(nil)

    // Homebrew fails - but workflow should still succeed
    s.env.OnActivity(UpdateHomebrew, mock.Anything, mock.Anything).Return(
        errors.New("homebrew tap auth failed"))

    s.env.ExecuteWorkflow(ReleaseWorkflow, params)

    // Workflow completes despite homebrew failure
    s.True(s.env.IsWorkflowCompleted())
    s.NoError(s.env.GetWorkflowError())
}
```

---

## Local Development Setup

### Docker Compose for Temporal

```yaml
# docker-compose.temporal.yml
version: '3.8'

services:
  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"  # gRPC frontend
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=temporal-postgresql
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
    depends_on:
      - temporal-postgresql

  temporal-postgresql:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: temporal
      POSTGRES_USER: temporal
    ports:
      - "5432:5432"
    volumes:
      - temporal-postgres-data:/var/lib/postgresql/data

  temporal-ui:
    image: temporalio/ui:latest
    ports:
      - "8080:8080"
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
      - TEMPORAL_CORS_ORIGINS=http://localhost:3000
    depends_on:
      - temporal

  # Worker for running contextd CI/CD activities
  contextd-worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    depends_on:
      - temporal
    volumes:
      - ./:/workspace
      - /var/run/docker.sock:/var/run/docker.sock

volumes:
  temporal-postgres-data:
```

### Quick Start Script

```bash
#!/bin/bash
# scripts/temporal-dev.sh

# Start Temporal server
docker-compose -f docker-compose.temporal.yml up -d

# Wait for Temporal to be ready
echo "Waiting for Temporal server..."
until curl -s http://localhost:8080 > /dev/null; do
    sleep 1
done
echo "Temporal UI available at http://localhost:8080"

# Register namespace for contextd
docker exec temporal-admin-tools tctl --ns contextd namespace register -rd 3

# Start the worker
go run ./cmd/temporal-worker
```

---

## Migration Path

### Phase 1: Local Testing (Low Risk)
- Add Temporal testing framework for workflow tests
- Use time-skipping for long-running test scenarios
- Keep GHA as primary CI, Temporal as local dev tool

### Phase 2: Parallel Operation
- Run both GHA and Temporal for releases
- Compare reliability and observability
- Temporal handles complex orchestration, GHA handles simple triggers

### Phase 3: Full Migration (Optional)
- Replace GHA workflows with minimal triggers
- Temporal handles all orchestration
- GHA only sends webhooks to start Temporal workflows

---

## Benefits for contextd

1. **Better Test Orchestration**: Run integration tests with proper dependency management
2. **Flaky Test Handling**: Built-in retries with exponential backoff
3. **Long-Running Tests**: Test compression workflows, embedding batch jobs without timeouts
4. **Observability**: Debug test failures with complete workflow history
5. **Local Development**: Same workflow definitions run locally and in CI
6. **Mirror Repos**: Test releases safely in "mirror" repositories

---

## Risks & Considerations

1. **Infrastructure Overhead**: Need to run Temporal server (mitigated by docker-compose)
2. **Learning Curve**: Team needs to learn Temporal concepts
3. **Vendor Lock-in**: Temporal is open-source but specialized
4. **GitHub Integration**: Need custom integration code for PR status updates

---

## Research Resources

- [Running GitHub Actions Through Temporal](https://temporal.io/blog/running-github-actions-temporal-guide) - Official guide
- [Temporal Go SDK Documentation](https://docs.temporal.io/develop/go)
- [Go SDK Testing Suite](https://docs.temporal.io/develop/go/testing-suite)
- [Sample Go Workflows](https://github.com/temporalio/samples-go)
- [Docker Compose Setup](https://github.com/temporalio/docker-compose)
- [Self-Hosted Deployment Guide](https://docs.temporal.io/self-hosted-guide)

---

## Next Steps

1. [ ] Set up local Temporal development environment
2. [ ] Create proof-of-concept workflow for `go test ./...`
3. [ ] Implement test workflow with activity mocking
4. [ ] Compare reliability vs current GHA setup
5. [ ] Document findings and decide on adoption level

---

## Acceptance Criteria

- [ ] Temporal server running locally via docker-compose
- [ ] At least one Temporal workflow that runs contextd tests
- [ ] Workflow tests using time-skipping feature
- [ ] Documentation comparing Temporal vs GHA for this use case
