# Temporal Workflows in contextd

**Complete guide to Temporal-based automation in the contextd project.**

---

## Table of Contents

1. [What is Temporal?](#what-is-temporal)
2. [Why Temporal for CI/CD?](#why-temporal-for-cicd)
3. [Architecture Overview](#architecture-overview)
4. [Component Deep Dive](#component-deep-dive)
5. [Workflow Execution Flow](#workflow-execution-flow)
6. [Trigger Mechanisms](#trigger-mechanisms)
7. [Workflows Reference](#workflows-reference)
8. [Deployment Guide](#deployment-guide)
9. [Monitoring & Debugging](#monitoring--debugging)
10. [Best Practices](#best-practices)

---

## What is Temporal?

**Temporal** is a durable workflow execution engine that provides:

- **Durability**: Workflows survive crashes and restarts
- **Reliability**: Automatic retries with exponential backoff
- **Visibility**: Full execution history and state inspection
- **Scalability**: Horizontal scaling of workers
- **Consistency**: Exactly-once execution guarantees

### Key Concepts

```
┌─────────────────────────────────────────────────────────────┐
│                    Temporal Concepts                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Workflow                                                    │
│  ├─ Orchestration logic (deterministic)                    │
│  ├─ Long-running (hours/days)                              │
│  └─ Survives process restarts                              │
│                                                              │
│  Activity                                                    │
│  ├─ Individual task execution (non-deterministic OK)       │
│  ├─ Can fail and retry                                     │
│  └─ Timeout-protected                                      │
│                                                              │
│  Worker                                                      │
│  ├─ Executes workflows and activities                      │
│  ├─ Polls for tasks                                        │
│  └─ Horizontally scalable                                  │
│                                                              │
│  Temporal Server                                             │
│  ├─ Orchestrates execution                                 │
│  ├─ Persists state to database                             │
│  └─ Manages task queues                                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Why Temporal for CI/CD?

We chose Temporal over GitHub Actions for several key reasons:

| Feature | Temporal | GitHub Actions |
|---------|----------|----------------|
| **Execution Environment** | Self-hosted, full control | GitHub-hosted, limited control |
| **State Persistence** | PostgreSQL, survives crashes | Ephemeral, restarts from scratch |
| **Retry Logic** | Configurable per-activity | Limited, workflow-level only |
| **Visibility** | Full history, queryable state | Logs only, limited debugging |
| **Complex Workflows** | Native support for branching, loops | Requires workarounds |
| **Resource Control** | Full control over compute | Limited to GitHub runners |
| **Cost** | Pay for infrastructure only | Pay per minute |
| **Integration** | Any API, any service | GitHub ecosystem focus |

**Key Advantages for contextd:**

1. **Internal Automation** - Temporal runs alongside contextd services
2. **Durable Execution** - Workflows survive server restarts
3. **Rich State Management** - Full execution history for debugging
4. **Flexible Retry Policies** - Per-activity retry configuration
5. **Testability** - Full local testing without cloud dependencies

---

## Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           GitHub Repository                              │
│                                                                          │
│  Pull Request Event (opened, synchronize, reopened)                     │
└────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ Webhook POST
                                  │ (signed with HMAC-SHA256)
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     GitHub Webhook Server                                │
│                   (cmd/github-webhook/main.go)                          │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐              │
│  │  1. Validate webhook signature                        │              │
│  │  2. Parse pull request event                          │              │
│  │  3. Extract PR metadata (owner, repo, number, SHA)    │              │
│  │  4. Create Temporal client                            │              │
│  │  5. Start workflow execution                          │              │
│  └──────────────────────────────────────────────────────┘              │
└────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ Start Workflow
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Temporal Server                                   │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐              │
│  │  Task Queues:                                         │              │
│  │  ├─ plugin-validation-queue                           │              │
│  │  │   └─ Plugin update validation workflows            │              │
│  │  └─ version-validation-queue (future)                 │              │
│  │      └─ Version consistency workflows                 │              │
│  └──────────────────────────────────────────────────────┘              │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐              │
│  │  State Persistence (PostgreSQL)                       │              │
│  │  ├─ Workflow execution history                        │              │
│  │  ├─ Activity results                                  │              │
│  │  ├─ Timer states                                      │              │
│  │  └─ Event sourcing for recovery                       │              │
│  └──────────────────────────────────────────────────────┘              │
└────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ Poll for Tasks
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Temporal Worker                                     │
│                 (cmd/plugin-validator/main.go)                          │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐              │
│  │  Registered Workflows:                                │              │
│  │  ├─ PluginUpdateValidationWorkflow                    │              │
│  │  └─ VersionValidationWorkflow (future)                │              │
│  │                                                        │              │
│  │  Registered Activities:                               │              │
│  │  ├─ FetchPRFilesActivity                              │              │
│  │  ├─ CategorizeFilesActivity                           │              │
│  │  ├─ ValidatePluginSchemasActivity                     │              │
│  │  ├─ PostReminderCommentActivity                       │              │
│  │  ├─ PostSuccessCommentActivity                        │              │
│  │  └─ FetchFileContentActivity                          │              │
│  └──────────────────────────────────────────────────────┘              │
│                                                                          │
│  Execution Flow:                                                         │
│  1. Poll task queue for workflow/activity tasks                         │
│  2. Execute workflow orchestration logic                                │
│  3. Schedule activities for execution                                   │
│  4. Execute activities (GitHub API calls, validations)                  │
│  5. Report results back to Temporal Server                              │
└────────────────────────────────┬────────────────────────────────────────┘
                                  │
                                  │ GitHub API Calls
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           GitHub API                                     │
│                                                                          │
│  Activities interact with:                                               │
│  ├─ Pull Requests API (list files, comments)                           │
│  ├─ Repository Contents API (fetch file content)                       │
│  └─ Issues API (post/update comments)                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Docker Compose Stack

```
┌─────────────────────────────────────────────────────────────────────────┐
│                  docker-compose.temporal.yml                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────┐                                                 │
│  │   PostgreSQL       │                                                 │
│  │   Port: 5432       │  ◄──── Temporal Server reads/writes state      │
│  └────────────────────┘                                                 │
│                                                                          │
│  ┌────────────────────┐                                                 │
│  │  Temporal Server   │                                                 │
│  │  Port: 7233 (gRPC) │  ◄──── Workers connect here                    │
│  └────────────────────┘       ◄──── Webhook server connects here       │
│                                                                          │
│  ┌────────────────────┐                                                 │
│  │  Temporal Web UI   │                                                 │
│  │  Port: 8080 (HTTP) │  ◄──── Access via http://localhost:8080        │
│  └────────────────────┘                                                 │
│                                                                          │
│  ┌─────────────────────────────┐                                        │
│  │  Plugin Validator Worker    │                                        │
│  │  (Background Service)        │  ◄──── Polls plugin-validation-queue │
│  └─────────────────────────────┘                                        │
│                                                                          │
│  ┌─────────────────────────────┐                                        │
│  │  GitHub Webhook Server      │                                        │
│  │  Port: 3000 (HTTP)           │  ◄──── GitHub sends webhooks here    │
│  └─────────────────────────────┘                                        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Component Deep Dive

### 1. GitHub Webhook Server (`cmd/github-webhook/main.go`)

**Purpose:** Receives GitHub webhook events and starts Temporal workflows.

**Responsibilities:**
- Listen for HTTP POST requests from GitHub
- Validate webhook signatures (HMAC-SHA256)
- Parse pull request events
- Create Temporal client connection
- Start workflows with appropriate configuration

**Key Code:**

```go
type WebhookServer struct {
    temporalClient client.Client        // Temporal client for starting workflows
    webhookSecret  config.Secret         // HMAC secret for signature validation
    logger         *logging.Logger
}

func (s *WebhookServer) handlePullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
    // Extract PR metadata
    action := event.GetAction()
    if action != "opened" && action != "synchronize" && action != "reopened" {
        return nil  // Ignore other actions
    }

    // Start Temporal workflow
    workflowOptions := client.StartWorkflowOptions{
        ID:        fmt.Sprintf("plugin-validation-%s-%d", repo, prNumber),
        TaskQueue: "plugin-validation-queue",
    }

    _, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions,
        workflows.PluginUpdateValidationWorkflow, config)

    return err
}
```

**Security:**
- Validates webhook signature before processing
- Uses `github.ValidatePayload()` for HMAC-SHA256 verification
- Rejects requests with invalid signatures (401 Unauthorized)

**Error Handling:**
- Logs all errors with context
- Returns appropriate HTTP status codes
- Does not expose internal errors to GitHub

---

### 2. Temporal Worker (`cmd/plugin-validator/main.go`)

**Purpose:** Executes workflows and activities.

**Responsibilities:**
- Connect to Temporal Server
- Register workflow and activity implementations
- Poll task queues for work
- Execute tasks and report results
- Handle graceful shutdown

**Key Code:**

```go
func main() {
    // Create Temporal client
    c, err := client.Dial(client.Options{
        HostPort: temporalHost,
    })

    // Create worker
    w := worker.New(c, "plugin-validation-queue", worker.Options{})

    // Register workflows
    w.RegisterWorkflow(workflows.PluginUpdateValidationWorkflow)
    w.RegisterWorkflow(workflows.VersionValidationWorkflow)

    // Register activities
    w.RegisterActivity(workflows.FetchPRFilesActivity)
    w.RegisterActivity(workflows.CategorizeFilesActivity)
    w.RegisterActivity(workflows.ValidatePluginSchemasActivity)
    w.RegisterActivity(workflows.PostReminderCommentActivity)
    w.RegisterActivity(workflows.PostSuccessCommentActivity)

    // Start worker (blocks until shutdown)
    err = w.Run(worker.InterruptCh())
}
```

**Worker Lifecycle:**
```
┌────────────────────────────────────────────┐
│         Worker Lifecycle                    │
├────────────────────────────────────────────┤
│                                             │
│  1. Startup                                 │
│     ├─ Connect to Temporal Server           │
│     ├─ Register workflows & activities      │
│     └─ Start polling task queues            │
│                                             │
│  2. Execution Loop                          │
│     ├─ Poll for workflow tasks              │
│     ├─ Execute workflow logic               │
│     ├─ Poll for activity tasks              │
│     ├─ Execute activity logic               │
│     └─ Report results to server             │
│                                             │
│  3. Shutdown (on SIGTERM/SIGINT)            │
│     ├─ Stop accepting new tasks             │
│     ├─ Complete in-progress tasks           │
│     ├─ Close Temporal connection            │
│     └─ Exit cleanly                         │
│                                             │
└────────────────────────────────────────────┘
```

**Scalability:**
- Multiple worker instances can run concurrently
- Each worker polls the same task queue
- Temporal Server distributes tasks across workers
- Horizontally scalable by adding more workers

---

### 3. Workflows (`internal/workflows/`)

**Purpose:** Orchestrate multi-step automation processes.

**Characteristics:**
- **Deterministic**: Must produce same result given same inputs
- **Durable**: Survive process crashes and restarts
- **Versioned**: Can be updated without breaking running workflows
- **Testable**: Fully testable with Temporal's test framework

**Workflow Structure:**

```go
func PluginUpdateValidationWorkflow(ctx workflow.Context, config Config) (*Result, error) {
    logger := workflow.GetLogger(ctx)

    // Configure activity options
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 2 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    result := &Result{}

    // Step 1: Execute activity
    var activityResult ActivityResult
    err := workflow.ExecuteActivity(ctx, SomeActivity, activityInput).Get(ctx, &activityResult)
    if err != nil {
        result.Errors = append(result.Errors, fmt.Sprintf("activity failed: %v", err))
        return result, err
    }

    // Step 2: Conditional logic based on result
    if activityResult.NeedsAction {
        err = workflow.ExecuteActivity(ctx, AnotherActivity, input).Get(ctx, nil)
    }

    return result, nil
}
```

**Important Constraints:**
- ❌ Cannot use `time.Now()` - use `workflow.Now(ctx)`
- ❌ Cannot use `rand.Random()` - use `workflow.NewRandom(ctx)`
- ❌ Cannot make direct HTTP calls - use activities
- ❌ Cannot read files - use activities
- ✅ Can use conditional logic, loops, functions
- ✅ Can schedule multiple activities in parallel
- ✅ Can sleep with `workflow.Sleep(ctx, duration)`

---

### 4. Activities (`internal/workflows/*_activities.go`)

**Purpose:** Execute non-deterministic operations (I/O, API calls, etc).

**Characteristics:**
- **Non-deterministic**: Can interact with external systems
- **Idempotent**: Should be safe to retry
- **Timeout-protected**: Automatically killed after timeout
- **Retryable**: Automatically retried on failure

**Activity Structure:**

```go
func FetchPRFilesActivity(ctx context.Context, input FetchPRFilesInput) ([]FileChange, error) {
    // Create GitHub client
    client, err := NewGitHubClient(ctx, input.GitHubToken)
    if err != nil {
        return nil, fmt.Errorf("failed to create GitHub client: %w", err)
    }

    // Fetch PR files with pagination
    opts := &github.ListOptions{PerPage: 100}
    var allFiles []*github.CommitFile
    for {
        files, resp, err := client.PullRequests.ListFiles(ctx, input.Owner, input.Repo, input.PRNumber, opts)
        if err != nil {
            return nil, fmt.Errorf("failed to list PR files: %w", err)
        }
        allFiles = append(allFiles, files...)
        if resp.NextPage == 0 {
            break
        }
        opts.Page = resp.NextPage
    }

    // Convert to domain model
    result := make([]FileChange, 0, len(allFiles))
    for _, f := range allFiles {
        result = append(result, FileChange{
            Path:   f.GetFilename(),
            Status: f.GetStatus(),
        })
    }

    return result, nil
}
```

**Activity Options:**

```go
type ActivityOptions struct {
    // How long activity can run before timeout
    StartToCloseTimeout time.Duration

    // How long between heartbeats before timeout
    HeartbeatTimeout time.Duration

    // Retry policy for activity failures
    RetryPolicy *RetryPolicy
}

type RetryPolicy struct {
    // Maximum number of retry attempts
    MaximumAttempts int32

    // Initial retry interval
    InitialInterval time.Duration

    // Maximum retry interval (with exponential backoff)
    MaximumInterval time.Duration

    // Backoff multiplier (e.g., 2.0 for doubling)
    BackoffCoefficient float64
}
```

---

## Workflow Execution Flow

### Plugin Update Validation Workflow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                  Plugin Update Validation Flow                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  GitHub PR Event (opened/sync/reopen)                                   │
│       │                                                                  │
│       ▼                                                                  │
│  ┌────────────────────────────┐                                        │
│  │  Webhook Server             │                                        │
│  │  - Validate signature       │                                        │
│  │  - Parse event              │                                        │
│  │  - Start workflow           │                                        │
│  └────────────┬───────────────┘                                        │
│               │                                                          │
│               ▼                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │  PluginUpdateValidationWorkflow                                  │  │
│  │                                                                   │  │
│  │  Step 1: Fetch PR Files                                          │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  FetchPRFilesActivity                     │                   │  │
│  │  │  - GitHub API: GET /pulls/:pr/files      │                   │  │
│  │  │  - Pagination (100 per page)              │                   │  │
│  │  │  - Returns: []FileChange                  │                   │  │
│  │  └──────────────┬───────────────────────────┘                   │  │
│  │                 │                                                 │  │
│  │                 ▼                                                 │  │
│  │  Step 2: Categorize Files                                        │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  CategorizeFilesActivity                  │                   │  │
│  │  │  - Regex pattern matching                 │                   │  │
│  │  │  - CodeFiles: internal/mcp/tools.go, etc  │                   │  │
│  │  │  - PluginFiles: .claude-plugin/**         │                   │  │
│  │  │  - Returns: CategorizedFiles              │                   │  │
│  │  └──────────────┬───────────────────────────┘                   │  │
│  │                 │                                                 │  │
│  │                 ▼                                                 │  │
│  │  Decision: NeedsUpdate?                                          │  │
│  │  (len(CodeFiles) > 0)                                            │  │
│  │       │                                                           │  │
│  │       ├─ No ──► Skip remaining steps ──► Return result           │  │
│  │       │                                                           │  │
│  │       └─ Yes                                                      │  │
│  │           │                                                       │  │
│  │           ▼                                                       │  │
│  │  Step 3: Validate Plugin Schemas (if plugin files changed)       │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  ValidatePluginSchemasActivity            │                   │  │
│  │  │  - For each JSON file in PluginFiles:     │                   │  │
│  │  │    ├─ Skip if status == "removed"         │                   │  │
│  │  │    ├─ Fetch content from GitHub           │                   │  │
│  │  │    ├─ Parse JSON                          │                   │  │
│  │  │    └─ Validate schema structure           │                   │  │
│  │  │  - Returns: SchemaValidationResult        │                   │  │
│  │  └──────────────┬───────────────────────────┘                   │  │
│  │                 │                                                 │  │
│  │                 ▼                                                 │  │
│  │  Decision: Plugin files changed?                                 │  │
│  │       │                                                           │  │
│  │       ├─ No (code changed, plugin didn't)                        │  │
│  │       │   │                                                       │  │
│  │       │   ▼                                                       │  │
│  │       │  ┌──────────────────────────────────┐                   │  │
│  │       │  │  PostReminderCommentActivity      │                   │  │
│  │       │  │  - Build reminder message         │                   │  │
│  │       │  │  - List changed code files        │                   │  │
│  │       │  │  - Provide checklist              │                   │  │
│  │       │  │  - Update if comment exists       │                   │  │
│  │       │  │  - Create new if not              │                   │  │
│  │       │  └──────────────────────────────────┘                   │  │
│  │       │                                                           │  │
│  │       └─ Yes (plugin updated correctly)                          │  │
│  │           │                                                       │  │
│  │           ▼                                                       │  │
│  │          ┌──────────────────────────────────┐                   │  │
│  │          │  PostSuccessCommentActivity       │                   │  │
│  │          │  - Build success message          │                   │  │
│  │          │  - Acknowledge plugin update      │                   │  │
│  │          │  - Include schema validation      │                   │  │
│  │          │  - Update if comment exists       │                   │  │
│  │          └──────────────────────────────────┘                   │  │
│  │                                                                   │  │
│  │  Return: PluginUpdateValidationResult                            │  │
│  │  - CodeFilesChanged: []string                                    │  │
│  │  - PluginFilesChanged: []string                                  │  │
│  │  - NeedsUpdate: bool                                             │  │
│  │  - SchemaValid: bool                                             │  │
│  │  - CommentPosted: bool                                           │  │
│  │  - CommentURL: string                                            │  │
│  │  - Errors: []string                                              │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### State Persistence & Recovery

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Workflow State Persistence                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Workflow Execution:                                                     │
│  ┌──────────────────────────┐                                          │
│  │  Workflow starts          │                                          │
│  └───────────┬──────────────┘                                          │
│              │                                                           │
│              ▼                                                           │
│  ┌──────────────────────────────────────────┐                          │
│  │  Event: WorkflowStarted                   │ ───┐                     │
│  │  - Timestamp                               │    │                     │
│  │  - Input: Config                           │    │                     │
│  └──────────────────────────────────────────┘    │                     │
│              │                                     │                     │
│              ▼                                     │                     │
│  ┌──────────────────────────────────────────┐    │                     │
│  │  Event: ActivityScheduled                 │    │                     │
│  │  - Activity: FetchPRFilesActivity          │    │                     │
│  │  - Input: FetchPRFilesInput                │    │ Persisted to      │
│  └──────────────────────────────────────────┘    │ PostgreSQL          │
│              │                                     │ (Event Sourcing)   │
│              ▼                                     │                     │
│  ┌──────────────────────────────────────────┐    │                     │
│  │  Event: ActivityCompleted                 │    │                     │
│  │  - Result: []FileChange                    │    │                     │
│  └──────────────────────────────────────────┘    │                     │
│              │                                     │                     │
│              ▼                                     │                     │
│  ┌──────────────────────────────────────────┐    │                     │
│  │  Event: ActivityScheduled                 │    │                     │
│  │  - Activity: CategorizeFilesActivity       │    │                     │
│  └──────────────────────────────────────────┘    │                     │
│              │                                     │                     │
│              ▼                                    ─┘                     │
│            ...                                                           │
│                                                                          │
│  ══════════════════════════════════════════════════════════════        │
│                                                                          │
│  Worker Crash Scenario:                                                 │
│  ┌──────────────────────────┐                                          │
│  │  Worker crashes after     │                                          │
│  │  ActivityScheduled event  │                                          │
│  └───────────┬──────────────┘                                          │
│              │                                                           │
│              ▼                                                           │
│  ┌──────────────────────────────────────────┐                          │
│  │  Temporal Server:                         │                          │
│  │  - Detects activity timeout               │                          │
│  │  - Activity not completed after 2 min     │                          │
│  │  - Reschedules activity on another worker │                          │
│  └──────────────────────────────────────────┘                          │
│              │                                                           │
│              ▼                                                           │
│  ┌──────────────────────────────────────────┐                          │
│  │  New Worker:                               │                          │
│  │  - Picks up activity task                 │                          │
│  │  - Executes FetchPRFilesActivity           │                          │
│  │  - Reports result                          │                          │
│  └──────────────────────────────────────────┘                          │
│              │                                                           │
│              ▼                                                           │
│  ┌──────────────────────────────────────────┐                          │
│  │  Temporal Server:                         │                          │
│  │  - Receives ActivityCompleted event       │                          │
│  │  - Persists to event log                  │                          │
│  │  - Continues workflow execution           │                          │
│  └──────────────────────────────────────────┘                          │
│                                                                          │
│  Result: Workflow completes successfully despite crash!                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Trigger Mechanisms

### GitHub Webhook Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      GitHub Webhook Trigger Flow                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  User Action: Open PR, Push Commits, Reopen PR                          │
│       │                                                                  │
│       ▼                                                                  │
│  ┌────────────────────────────┐                                        │
│  │  GitHub                     │                                        │
│  │  - Detects PR event         │                                        │
│  │  - Generates webhook payload│                                        │
│  │  - Computes HMAC-SHA256     │                                        │
│  │    signature                │                                        │
│  └────────────┬───────────────┘                                        │
│               │                                                          │
│               │ HTTP POST                                                │
│               │ X-Hub-Signature-256: sha256=<signature>                  │
│               │ Content-Type: application/json                           │
│               │                                                          │
│               ▼                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │  Webhook Server (Port 3000)                                      │  │
│  │                                                                   │  │
│  │  Step 1: Validate Signature                                      │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  payload, err := github.ValidatePayload( │                   │  │
│  │  │      request,                             │                   │  │
│  │  │      []byte(webhookSecret)                │                   │  │
│  │  │  )                                        │                   │  │
│  │  │                                           │                   │  │
│  │  │  If err != nil:                           │                   │  │
│  │  │    Return 401 Unauthorized                │                   │  │
│  │  └──────────────────────────────────────────┘                   │  │
│  │                                                                   │  │
│  │  Step 2: Parse Event                                             │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  event := &github.PullRequestEvent{}      │                   │  │
│  │  │  json.Unmarshal(payload, event)           │                   │  │
│  │  │                                           │                   │  │
│  │  │  Extract:                                 │                   │  │
│  │  │  - action (opened/synchronize/reopened)   │                   │  │
│  │  │  - owner (repo owner)                     │                   │  │
│  │  │  - repo (repo name)                       │                   │  │
│  │  │  - number (PR number)                     │                   │  │
│  │  │  - head SHA (commit hash)                 │                   │  │
│  │  │  - base branch                            │                   │  │
│  │  │  - head branch                            │                   │  │
│  │  └──────────────────────────────────────────┘                   │  │
│  │                                                                   │  │
│  │  Step 3: Filter Events                                           │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  if action not in [opened, synchronize,   │                   │  │
│  │  │                     reopened]:            │                   │  │
│  │  │    Return 200 OK (ignore event)           │                   │  │
│  │  └──────────────────────────────────────────┘                   │  │
│  │                                                                   │  │
│  │  Step 4: Start Workflow                                          │  │
│  │  ┌──────────────────────────────────────────┐                   │  │
│  │  │  workflowID := fmt.Sprintf(                │                   │  │
│  │  │    "plugin-validation-%s-%s-%d",          │                   │  │
│  │  │    owner, repo, prNumber                  │                   │  │
│  │  │  )                                        │                   │  │
│  │  │                                           │                   │  │
│  │  │  config := PluginUpdateValidationConfig{  │                   │  │
│  │  │    Owner: owner,                          │                   │  │
│  │  │    Repo: repo,                            │                   │  │
│  │  │    PRNumber: prNumber,                    │                   │  │
│  │  │    HeadSHA: headSHA,                      │                   │  │
│  │  │    GitHubToken: githubToken,              │                   │  │
│  │  │  }                                        │                   │  │
│  │  │                                           │                   │  │
│  │  │  temporalClient.ExecuteWorkflow(          │                   │  │
│  │  │    ctx,                                   │                   │  │
│  │  │    client.StartWorkflowOptions{          │                   │  │
│  │  │      ID: workflowID,                      │                   │  │
│  │  │      TaskQueue: "plugin-validation-queue",│                   │  │
│  │  │    },                                     │                   │  │
│  │  │    PluginUpdateValidationWorkflow,        │                   │  │
│  │  │    config                                 │                   │  │
│  │  │  )                                        │                   │  │
│  │  └──────────────────────────────────────────┘                   │  │
│  │                                                                   │  │
│  │  Return: 200 OK                                                  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Workflow ID Strategy

**Workflow IDs must be unique** to prevent duplicate executions:

```
Format: "plugin-validation-{owner}-{repo}-{pr_number}"
Examples:
  - "plugin-validation-fyrsmithlabs-contextd-65"
  - "plugin-validation-acme-myapp-123"

Benefits:
  - Prevents duplicate workflow executions for same PR
  - Easy to identify workflows in Temporal UI
  - Automatic deduplication by Temporal Server

Behavior:
  - If workflow with same ID is already running, new request is ignored
  - If workflow with same ID completed, new request starts fresh workflow
  - Supports retrying failed workflows by reusing workflow ID
```

---

## Workflows Reference

### 1. Plugin Update Validation Workflow

**File:** `internal/workflows/plugin_validation.go`

**Purpose:** Detect code changes requiring Claude plugin updates and ensure plugins are updated correctly.

**Triggers:**
- Pull request opened
- Pull request synchronized (new commits)
- Pull request reopened

**Input:**
```go
type PluginUpdateValidationConfig struct {
    Owner       string        // GitHub repo owner
    Repo        string        // GitHub repo name
    PRNumber    int           // Pull request number
    BaseBranch  string        // Base branch (usually "main")
    HeadBranch  string        // PR branch
    HeadSHA     string        // PR commit SHA
    GitHubToken config.Secret // GitHub API token
}
```

**Output:**
```go
type PluginUpdateValidationResult struct {
    CodeFilesChanged   []string // Files affecting plugin
    PluginFilesChanged []string // Files in .claude-plugin/
    NeedsUpdate        bool     // Whether plugin needs updating
    SchemaValid        bool     // Whether schemas are valid JSON
    CommentPosted      bool     // Whether we posted a comment
    CommentURL         string   // URL of posted comment
    Errors             []string // Any errors encountered
}
```

**Activities:**
1. `FetchPRFilesActivity` - List all changed files in PR
2. `CategorizeFilesActivity` - Categorize files (code vs plugin)
3. `ValidatePluginSchemasActivity` - Validate JSON schemas
4. `PostReminderCommentActivity` - Post reminder if plugin not updated
5. `PostSuccessCommentActivity` - Post success if plugin updated

**File Categorization Patterns:**
```go
CodeFiles (require plugin update):
  - internal/mcp/tools.go
  - internal/mcp/handlers/*.go
  - internal/*/service.go
  - internal/config/{types,config}.go

PluginFiles:
  - .claude-plugin/**/*
```

---

### 2. Version Validation Workflow

**File:** `internal/workflows/version_validation.go`

**Purpose:** Ensure VERSION file matches version in plugin.json.

**Triggers:**
- Pull request opened
- Pull request synchronized (new commits)
- Pull request reopened

**Input:**
```go
type VersionValidationConfig struct {
    Owner       string        // GitHub repo owner
    Repo        string        // GitHub repo name
    PRNumber    int           // Pull request number
    HeadSHA     string        // PR commit SHA
    GitHubToken config.Secret // GitHub API token
}
```

**Output:**
```go
type VersionValidationResult struct {
    VersionMatches bool     // Whether versions match
    VersionFile    string   // Version from VERSION file
    PluginVersion  string   // Version from plugin.json
    CommentPosted  bool     // Whether we posted a comment
    CommentURL     string   // URL of posted comment
    Errors         []string // Any errors encountered
}
```

**Activities:**
1. `FetchFileContentActivity` - Fetch VERSION file
2. `FetchFileContentActivity` - Fetch plugin.json
3. `PostVersionMismatchCommentActivity` - Post comment if mismatch

**Version Format Support:**
- Standard: `1.2.3`
- Pre-release: `1.0.0-rc.1`, `1.0.0-beta.1`
- Build metadata: `1.0.0+build.123`
- Complex: `2.0.0-beta.1+exp.sha.5114f85`

---

## Deployment Guide

### Local Development

**1. Start Temporal Stack:**

```bash
# Set environment variables
export GITHUB_TOKEN=ghp_your_token_here
export GITHUB_WEBHOOK_SECRET=your_secret_here

# Start all services
docker-compose -f deploy/docker-compose.temporal.yml up

# Access Temporal Web UI
open http://localhost:8080
```

**Services Started:**
- PostgreSQL (port 5432) - State persistence
- Temporal Server (port 7233) - Workflow engine
- Temporal Web UI (port 8080) - Monitoring dashboard
- Plugin Validator Worker - Executes workflows/activities
- GitHub Webhook Server (port 3000) - Receives webhooks

**2. Configure GitHub Webhook:**

```
Repository Settings → Webhooks → Add webhook

Payload URL: http://your-server:3000/webhook
Content type: application/json
Secret: <your_webhook_secret>
SSL verification: Enable

Events:
  ☑ Pull requests (opened, synchronize, reopened)
```

**3. Test Workflow:**

```bash
# Option 1: Open a real PR
# GitHub will send webhook automatically

# Option 2: Manually trigger via Temporal CLI
temporal workflow start \
  --task-queue plugin-validation-queue \
  --type PluginUpdateValidationWorkflow \
  --workflow-id test-123 \
  --input '{
    "Owner": "fyrsmithlabs",
    "Repo": "contextd",
    "PRNumber": 65,
    "HeadSHA": "abc123"
  }'

# Option 3: Send test webhook
curl -X POST http://localhost:3000/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=..." \
  -d @test-payload.json
```

---

### Production Deployment

**Architecture:**

```
┌──────────────────────────────────────────────────────────────┐
│                    Production Architecture                    │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Internet                                                     │
│    │                                                          │
│    │ HTTPS                                                    │
│    ▼                                                          │
│  ┌─────────────────┐                                         │
│  │  Load Balancer  │                                         │
│  │  (nginx/HAProxy)│                                         │
│  └────────┬────────┘                                         │
│           │                                                   │
│           ├─────────────┬─────────────────┐                 │
│           │             │                 │                 │
│           ▼             ▼                 ▼                 │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐          │
│  │  Webhook   │  │  Webhook   │  │  Webhook   │          │
│  │  Server 1  │  │  Server 2  │  │  Server 3  │          │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘          │
│        │               │               │                   │
│        └───────────────┴───────────────┘                   │
│                        │                                    │
│                        │ gRPC                               │
│                        ▼                                    │
│               ┌──────────────────┐                         │
│               │  Temporal Server │                         │
│               │   (Clustered)    │                         │
│               └────────┬─────────┘                         │
│                        │                                    │
│                        │ PostgreSQL Protocol                │
│                        ▼                                    │
│               ┌──────────────────┐                         │
│               │   PostgreSQL     │                         │
│               │   (HA Setup)     │                         │
│               └──────────────────┘                         │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐          │
│  │  Worker 1  │  │  Worker 2  │  │  Worker 3  │          │
│  │            │  │            │  │            │          │
│  │  Polls:    │  │  Polls:    │  │  Polls:    │          │
│  │  - plugin  │  │  - plugin  │  │  - plugin  │          │
│  │    -queue  │  │    -queue  │  │    -queue  │          │
│  └────────────┘  └────────────┘  └────────────┘          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Deployment Steps:**

**1. Deploy PostgreSQL (HA):**

```bash
# Use managed service (AWS RDS, GCP Cloud SQL, Azure Database)
# OR deploy PostgreSQL cluster with replication

# Example: AWS RDS
aws rds create-db-instance \
  --db-instance-identifier contextd-temporal-prod \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --master-username temporal \
  --master-user-password <strong-password> \
  --allocated-storage 100 \
  --backup-retention-period 7 \
  --multi-az
```

**2. Deploy Temporal Server:**

```bash
# Use Temporal Cloud (recommended)
# OR deploy self-hosted cluster

# Self-hosted via Kubernetes
helm repo add temporalio https://go.temporal.io/helm-charts
helm repo update

helm install temporal-prod temporalio/temporal \
  --namespace temporal \
  --create-namespace \
  --values temporal-values.yaml
```

**3. Deploy Webhook Servers:**

```bash
# Build container
docker build -f Dockerfile.github-webhook -t contextd-webhook:latest .

# Deploy to Kubernetes
kubectl apply -f k8s/webhook-deployment.yaml

# OR deploy to cloud service
# AWS ECS, GCP Cloud Run, Azure Container Instances
```

**4. Deploy Workers:**

```bash
# Build container
docker build -f Dockerfile.plugin-validator -t contextd-worker:latest .

# Deploy to Kubernetes
kubectl apply -f k8s/worker-deployment.yaml

# Scale workers
kubectl scale deployment contextd-worker --replicas=5
```

**5. Configure Secrets:**

```bash
# Kubernetes secrets
kubectl create secret generic temporal-secrets \
  --from-literal=github-token=ghp_xxx \
  --from-literal=github-webhook-secret=xxx

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name contextd/temporal/github-token \
  --secret-string ghp_xxx
```

**6. Configure Load Balancer:**

```nginx
# nginx.conf
upstream webhook_servers {
    server webhook-1:3000;
    server webhook-2:3000;
    server webhook-3:3000;
}

server {
    listen 443 ssl;
    server_name webhooks.yourcompany.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location /webhook {
        proxy_pass http://webhook_servers;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $host;
    }
}
```

---

## Monitoring & Debugging

### Temporal Web UI

**Access:** `http://localhost:8080` (local) or `https://temporal-ui.yourcompany.com` (prod)

**Features:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       Temporal Web UI Features                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Workflows Page                                                          │
│  ├─ List all workflow executions                                       │
│  ├─ Filter by status (running, completed, failed)                      │
│  ├─ Filter by workflow type                                            │
│  ├─ Search by workflow ID                                              │
│  └─ View execution history                                             │
│                                                                          │
│  Workflow Details Page                                                   │
│  ├─ Event history (all workflow events)                                │
│  ├─ Activity results                                                    │
│  ├─ Error stack traces                                                 │
│  ├─ Input/output payloads                                              │
│  ├─ Retry attempts                                                     │
│  └─ Execution timeline                                                 │
│                                                                          │
│  Task Queues Page                                                        │
│  ├─ View task queue status                                             │
│  ├─ See backlog size                                                   │
│  ├─ Monitor worker connections                                         │
│  └─ Track processing rates                                             │
│                                                                          │
│  Namespaces Page                                                         │
│  ├─ Manage workflow namespaces                                         │
│  ├─ Configure retention policies                                       │
│  └─ Set archival settings                                              │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

**Example: Debugging Failed Workflow**

```
1. Navigate to Workflows → Filter: Status=Failed
2. Click on failed workflow ID
3. View Event History:
   - WorkflowStarted (input config)
   - ActivityScheduled (FetchPRFilesActivity)
   - ActivityFailed (error: rate limit exceeded)
   - RetryAttempt #1
   - ActivityFailed (error: rate limit exceeded)
   - RetryAttempt #2
   - ActivityCompleted (success!)
4. View final result in WorkflowCompleted event
```

### Logging

**Structured Logging in Workflows:**

```go
func PluginUpdateValidationWorkflow(ctx workflow.Context, config Config) (*Result, error) {
    logger := workflow.GetLogger(ctx)

    logger.Info("Starting plugin validation",
        "owner", config.Owner,
        "repo", config.Repo,
        "pr", config.PRNumber)

    // ... workflow logic ...

    logger.Info("Plugin validation complete",
        "needs_update", result.NeedsUpdate,
        "schema_valid", result.SchemaValid)

    return result, nil
}
```

**Viewing Logs:**

```bash
# Docker Compose logs
docker-compose -f deploy/docker-compose.temporal.yml logs -f plugin-validator-worker

# Kubernetes logs
kubectl logs -f deployment/contextd-worker -n temporal

# Grep for specific workflow
kubectl logs deployment/contextd-worker -n temporal | grep "plugin-validation-fyrsmithlabs-contextd-65"
```

### Metrics

contextd workflows emit comprehensive OpenTelemetry metrics for monitoring and alerting.

#### contextd Workflow Metrics

**Version Validation Workflow:**

```
# Workflow Execution Counters
contextd.workflows.version_validation.executions
  - Description: Total number of version validation workflow executions
  - Unit: {execution}
  - Labels: status (success/failure)
  - Use: Track workflow execution rate and success rate

# Workflow Duration
contextd.workflows.version_validation.duration
  - Description: Duration of version validation workflow executions
  - Unit: seconds
  - Type: Histogram
  - Use: Monitor workflow latency, set SLOs

# Version Match/Mismatch Counters
contextd.workflows.version_validation.matches
  - Description: Number of version matches detected
  - Unit: {match}
  - Use: Track successful version synchronization

contextd.workflows.version_validation.mismatches
  - Description: Number of version mismatches detected
  - Unit: {mismatch}
  - Use: Alert on version inconsistencies

# Activity Metrics
contextd.workflows.activity.duration
  - Description: Duration of workflow activity executions
  - Unit: seconds
  - Type: Histogram
  - Labels: activity (FetchFileContent, PostVersionMismatchComment, etc.)
  - Use: Identify slow activities, optimize performance

contextd.workflows.activity.errors
  - Description: Number of activity execution errors
  - Unit: {error}
  - Labels: activity, error_type (invalid_path, rate_limit, api_error, etc.)
  - Use: Alert on specific error patterns, troubleshoot issues
```

**Plugin Validation Workflow:**

```
# Similar pattern for plugin validation
contextd.workflows.plugin_validation.executions
contextd.workflows.plugin_validation.duration
contextd.workflows.plugin_validation.plugin_updates_required
contextd.workflows.plugin_validation.schema_validation_failures
```

#### Temporal Server Metrics

**Built-in Temporal Metrics:**

```
Workflow Metrics:
  - temporal_workflow_start_total
  - temporal_workflow_completed_total
  - temporal_workflow_failed_total
  - temporal_workflow_timeout_total

Activity Metrics:
  - temporal_activity_execution_latency
  - temporal_activity_task_schedule_to_start_latency
  - temporal_activity_succeed_total
  - temporal_activity_failed_total

Worker Metrics:
  - temporal_worker_task_slots_available
  - temporal_sticky_cache_size
  - temporal_worker_registered_activities
```

#### Example Prometheus Queries

**Version Validation Monitoring:**

```promql
# Version mismatch rate (per hour)
rate(contextd_workflows_version_validation_mismatches_total[1h])

# Workflow success rate
sum(rate(contextd_workflows_version_validation_executions_total{status="success"}[5m]))
/
sum(rate(contextd_workflows_version_validation_executions_total[5m]))

# Average workflow duration
rate(contextd_workflows_version_validation_duration_sum[5m])
/
rate(contextd_workflows_version_validation_duration_count[5m])

# P95 workflow latency
histogram_quantile(0.95,
  sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))

# Activity error rate by type
sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type)

# Top 5 slowest activities
topk(5,
  rate(contextd_workflows_activity_duration_sum[5m])
  /
  rate(contextd_workflows_activity_duration_count[5m]))
```

**Temporal Server Monitoring:**

```promql
# Activity failure rate
rate(temporal_activity_failed_total{activity_type="FetchPRFilesActivity"}[5m])

# Workflow completion latency p95
histogram_quantile(0.95,
  rate(temporal_workflow_endtoend_latency_bucket{workflow_type="PluginUpdateValidationWorkflow"}[5m]))

# Worker task slot utilization
(temporal_worker_task_slots_total - temporal_worker_task_slots_available)
/
temporal_worker_task_slots_total
```

#### Alerting Rules

**Recommended Prometheus Alerts:**

```yaml
groups:
  - name: contextd_workflows
    interval: 30s
    rules:
      # Alert on high version mismatch rate
      - alert: HighVersionMismatchRate
        expr: |
          rate(contextd_workflows_version_validation_mismatches_total[1h]) > 0.5
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High version mismatch rate detected"
          description: "More than 0.5 version mismatches per hour over the last 15 minutes"

      # Alert on workflow failures
      - alert: WorkflowFailureRate
        expr: |
          sum(rate(contextd_workflows_version_validation_executions_total{status="failure"}[5m]))
          /
          sum(rate(contextd_workflows_version_validation_executions_total[5m]))
          > 0.1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "High workflow failure rate"
          description: "More than 10% of workflows are failing"

      # Alert on high activity error rate
      - alert: HighActivityErrorRate
        expr: |
          sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High activity error rate for {{ $labels.error_type }}"
          description: "More than 1 error per second for {{ $labels.error_type }}"

      # Alert on slow workflows
      - alert: SlowWorkflowExecution
        expr: |
          histogram_quantile(0.95,
            sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))
          > 120
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Slow workflow execution detected"
          description: "P95 workflow duration is over 2 minutes"

      # Alert on GitHub rate limiting
      - alert: GitHubRateLimitHit
        expr: |
          sum(rate(contextd_workflows_activity_errors_total{error_type="rate_limit"}[5m])) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "GitHub API rate limit exceeded"
          description: "Workflows are being rate limited by GitHub API"
```

#### Grafana Dashboard

**Example Dashboard JSON:**

```json
{
  "dashboard": {
    "title": "contextd Temporal Workflows",
    "panels": [
      {
        "title": "Workflow Execution Rate",
        "targets": [
          {
            "expr": "rate(contextd_workflows_version_validation_executions_total[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Workflow Duration (P50, P95, P99)",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))",
            "legendFormat": "P99"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Version Matches vs Mismatches",
        "targets": [
          {
            "expr": "rate(contextd_workflows_version_validation_matches_total[5m])",
            "legendFormat": "Matches"
          },
          {
            "expr": "rate(contextd_workflows_version_validation_mismatches_total[5m])",
            "legendFormat": "Mismatches"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Activity Errors by Type",
        "targets": [
          {
            "expr": "sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type)"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Activity Duration Heatmap",
        "targets": [
          {
            "expr": "sum(rate(contextd_workflows_activity_duration_bucket[5m])) by (le, activity)"
          }
        ],
        "type": "heatmap"
      }
    ]
  }
}
```

See `internal/workflows/version_validation_metrics_test.go` for metric testing examples.

---

## Best Practices

### 1. Workflow Design

**✅ DO:**
- Keep workflows deterministic
- Use `workflow.Now(ctx)` instead of `time.Now()`
- Use `workflow.Sleep(ctx, duration)` instead of `time.Sleep()`
- Use `workflow.NewRandom(ctx)` for randomness
- Handle errors gracefully
- Return structured results

**❌ DON'T:**
- Make direct HTTP calls in workflows
- Read files in workflows
- Use global state
- Use non-deterministic functions
- Ignore errors

### 2. Activity Design

**✅ DO:**
- Make activities idempotent (safe to retry)
- Use context for cancellation
- Implement pagination for large datasets
- Report heartbeats for long-running activities
- Use structured errors
- Log important events

**❌ DON'T:**
- Assume single execution (always retry-safe)
- Ignore context cancellation
- Load entire datasets in memory
- Ignore timeouts
- Swallow errors

### 3. Error Handling

**Activity Retry Policy:**

```go
ao := workflow.ActivityOptions{
    StartToCloseTimeout: 2 * time.Minute,
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts:    3,
        InitialInterval:    time.Second,
        MaximumInterval:    time.Minute,
        BackoffCoefficient: 2.0,
        // Retry on all errors except these:
        NonRetryableErrorTypes: []string{
            "InvalidTokenError",
            "PermissionDeniedError",
        },
    },
}
```

**Graceful Degradation:**

```go
// If non-critical activity fails, continue workflow
err := workflow.ExecuteActivity(ctx, OptionalActivity, input).Get(ctx, &result)
if err != nil {
    logger.Warn("Optional activity failed, continuing", "error", err)
    result.Errors = append(result.Errors, err.Error())
    // Don't return error - continue workflow
} else {
    // Use result
}
```

### 4. Testing

**Unit Test Workflows:**

```go
func TestPluginValidationWorkflow(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Register workflow
    env.RegisterWorkflow(PluginUpdateValidationWorkflow)

    // Mock activities
    env.OnActivity(FetchPRFilesActivity, mock.Anything, mock.Anything).
        Return([]FileChange{...}, nil)

    // Execute workflow
    env.ExecuteWorkflow(PluginUpdateValidationWorkflow, config)

    // Assert results
    require.True(t, env.IsWorkflowCompleted())
    var result PluginUpdateValidationResult
    env.GetWorkflowResult(&result)
    assert.True(t, result.NeedsUpdate)
}
```

### 5. Performance

**Parallel Activity Execution:**

```go
// Execute multiple activities in parallel
var futures []workflow.Future

for _, file := range files {
    future := workflow.ExecuteActivity(ctx, ValidateFileActivity, file)
    futures = append(futures, future)
}

// Wait for all to complete
for _, future := range futures {
    var result ValidationResult
    err := future.Get(ctx, &result)
    // Handle result
}
```

**Worker Scaling:**

```bash
# Scale workers based on queue depth
# If backlog > 100, add more workers
kubectl scale deployment contextd-worker --replicas=10

# If backlog < 10, reduce workers
kubectl scale deployment contextd-worker --replicas=3
```

---

## Summary

**Key Takeaways:**

1. **Temporal provides durable workflow execution** - Survives crashes, automatic retries, full visibility
2. **Workflows orchestrate, activities execute** - Clear separation of concerns
3. **GitHub webhooks trigger workflows** - Automatic PR validation on every push
4. **State persists to PostgreSQL** - Event sourcing enables recovery
5. **Horizontally scalable** - Add workers to handle increased load
6. **Fully testable** - Mock activities, assert workflow behavior
7. **Production-ready** - HA deployment, monitoring, error handling

**Resources:**

- Temporal Docs: https://docs.temporal.io
- Temporal Go SDK: https://github.com/temporalio/sdk-go
- contextd Workflows: `internal/workflows/`
- Deployment Guide: `deploy/TEMPORAL_DEPLOYMENT.md`

---

*Last Updated: 2025-12-29*
*Author: Claude Code (Sonnet 4.5)*
