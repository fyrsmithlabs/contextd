# Remediation Example

This example demonstrates the error remediation pattern in contextd: **record → search → reuse**. Remediation enables you to capture error fix patterns and automatically surface similar solutions when errors occur again.

## Overview

Remediation provides error pattern tracking by:

1. **Record** error fix patterns (problem, root cause, solution, code diff)
2. **Search** for similar errors using semantic similarity
3. **Reuse** proven solutions from past fixes
4. **Learn** through feedback to improve confidence scores

This is how contextd prevents teams from solving the same problem twice.

## The Pattern

```
┌─────────────────────────────────────────────────────────┐
│            Developer Encounters Error                    │
└────────────────────┬───────────────────────────────────┘
                     │
                     ▼
         ┌──────────────────────┐
         │  1. Search for Fix   │  ← remediation_search
         │  (Semantic lookup)   │      (error message/stack trace)
         └──────────┬───────────┘
                    │
                    ├─── Match Found ──────────┐
                    │                          │
                    │                          ▼
                    │              ┌──────────────────────┐
                    │              │  2. Apply Solution   │
                    │              │  (Reuse past fix)    │
                    │              └──────────┬───────────┘
                    │                         │
                    │                         ▼
                    │              ┌──────────────────────┐
                    │              │  3. Provide Feedback │  ← remediation_feedback
                    │              │  (helpful/outdated)  │
                    │              └──────────────────────┘
                    │
                    └─── No Match ────────────┐
                                             │
                                             ▼
                                 ┌──────────────────────┐
                                 │  4. Debug & Fix      │
                                 │  (Solve problem)     │
                                 └──────────┬───────────┘
                                            │
                                            ▼
                                 ┌──────────────────────┐
                                 │  5. Record Fix       │  ← remediation_record
                                 │  (Capture pattern)   │
                                 └──────────────────────┘
```

## Why Remediation?

### The Problem

Teams waste time solving the same errors repeatedly:

- Developer A fixes "database connection pool exhausted" in Service X
- 3 months later, Developer B encounters same issue in Service Y
- Developer B spends hours debugging (solution already existed!)
- Same pattern repeats across team, wasting collective time

### The Solution

Remediation captures institutional knowledge:

1. **Semantic Search**: Find similar errors even with different wording
2. **Confidence Scores**: Learn which solutions actually work through feedback
3. **Scope Hierarchy**: Share fixes at project/team/org level
4. **Code Diffs**: See exact code changes that fixed the issue

### Real-World Impact

| Metric | Without contextd | With contextd |
|--------|-----------------|---------------|
| Time to fix recurring errors | 2-4 hours | 5-10 minutes |
| Knowledge loss on team changes | High | Low (captured) |
| Solution quality | Inconsistent | Improves over time |
| Cross-team learning | Minimal | Automatic |

## Quick Start

### Prerequisites

- contextd running (either as MCP server or standalone)
- Go 1.25+ installed

### Run the Example

```bash
# From the examples/remediation directory
go run main.go

# Or build first
go build -o remediation
./remediation
```

### Expected Output

```
Remediation Example - Demonstrating record->search->reuse workflow
===================================================================

Step 1: Recording error fix - Nil pointer dereference...
✓ Recorded remediation: "Nil pointer dereference in user handler" (ID: abc12345)

Step 2: Recording error fix - Database connection timeout...
✓ Recorded remediation: "Database connection pool exhaustion" (ID: def67890)

Step 3: Recording error fix - Test flakiness...
✓ Recorded remediation: "Flaky integration test: TestUserLogin" (ID: ghi34567)

Step 4: Encountering similar error...
Error: panic: runtime error: invalid memory address
Looking for similar fixes...

Found 1 similar fixes:

1. [abc12345] Nil pointer dereference in user handler (score: 0.89, confidence: 0.50)
   Problem: Application crashes with panic: runtime error: invalid memory address...
   Solution: Add nil check in middleware before calling handler. Return 401 if user...

Step 5: Applying solution from "Nil pointer dereference in user handler"...
Applied fix successfully! ✓
Providing feedback...

✓ Provided positive feedback (confidence increased)

Step 6: Searching for performance-related fixes...
Found 1 performance-related fixes:
  - [def67890] Database connection pool exhaustion (tags: [database performance connection-pool])

Step 7: Demonstrating scope hierarchy...
Found 3 fixes across all scopes:
  - [abc12345] Nil pointer dereference in user handler (scope: project)
  - [def67890] Database connection pool exhaustion (scope: team)
  - [ghi34567] Flaky integration test: TestUserLogin (scope: org)

✓ Remediation workflow complete!
```

## How It Works

### 1. Record Error Fix

After solving an error, record the fix pattern:

```go
// Record the fix
recordReq := &remediation.RecordRequest{
    Title:    "Nil pointer dereference in user handler",
    Problem:  "Application crashes with panic: runtime error: invalid memory address",
    Symptoms: []string{
        "Server returns 500 error",
        "Panic stack trace shows user.go:45",
        "Only happens when user is not authenticated",
    },
    RootCause: "User middleware doesn't check if context.User is nil",
    Solution:  "Add nil check in middleware before calling handler",
    CodeDiff: `diff --git a/middleware/auth.go b/middleware/auth.go
--- a/middleware/auth.go
+++ b/middleware/auth.go
@@ -10,6 +10,10 @@ func AuthMiddleware(next http.Handler) http.Handler {
     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
         user := getUserFromContext(r.Context())
+        if user == nil {
+            http.Error(w, "Unauthorized", http.StatusUnauthorized)
+            return
+        }
         next.ServeHTTP(w, r)
     })
 }`,
    AffectedFiles: []string{"middleware/auth.go"},
    Category:      remediation.ErrorRuntime,
    Tags:          []string{"panic", "authentication", "middleware"},
    Scope:         remediation.ScopeProject,
    TenantID:      tenant,
    TeamID:        teamID,
    ProjectPath:   projectPath,
}

rem, err := service.Record(ctx, recordReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Recorded: %s (ID: %s)\n", rem.Title, rem.ID)
```

**When to record?**
- ✅ After solving a non-trivial error
- ✅ When the solution might help others
- ✅ For recurring error patterns
- ✅ When you learned something valuable
- ❌ For obvious typos or trivial fixes

### 2. Search for Similar Errors

When encountering an error, search for similar fixes:

```go
// Search using error message and context
searchReq := &remediation.SearchRequest{
    Query:            "panic nil pointer error user context",
    Limit:            5,
    MinConfidence:    0.3, // Minimum confidence threshold
    Category:         remediation.ErrorRuntime, // Filter by category
    TenantID:         tenant,
    TeamID:           teamID,
    ProjectPath:      projectPath,
    Scope:            remediation.ScopeProject,
    IncludeHierarchy: true, // Search project → team → org
}

results, err := service.Search(ctx, searchReq)
if err != nil {
    log.Fatal(err)
}

// Review results
for _, result := range results {
    fmt.Printf("[%s] %s (score: %.2f)\n",
        result.ID[:8], result.Title, result.Score)
    fmt.Printf("Solution: %s\n", result.Solution)
    if result.CodeDiff != "" {
        fmt.Printf("Code diff:\n%s\n", result.CodeDiff)
    }
}
```

**Search tips:**
- Use error message text directly
- Include stack trace context
- Use category filters to narrow results
- Set appropriate confidence thresholds
- Enable hierarchy to search parent scopes

### 3. Provide Feedback

After applying a solution, provide feedback to improve confidence:

```go
// Mark as helpful if it solved the problem
feedbackReq := &remediation.FeedbackRequest{
    RemediationID: remediationID,
    TenantID:      tenant,
    Rating:        remediation.RatingHelpful, // or NotHelpful/Outdated
    SessionID:     sessionID,
    Comment:       "Fixed the issue immediately",
}

err := service.Feedback(ctx, feedbackReq)
if err != nil {
    log.Fatal(err)
}
```

**Feedback ratings:**

| Rating | Effect | When to Use |
|--------|--------|-------------|
| `RatingHelpful` | Confidence +0.1 | Solution worked perfectly |
| `RatingNotHelpful` | Confidence -0.1 | Solution didn't help |
| `RatingOutdated` | Confidence -0.2 | Solution is no longer relevant |

Confidence scores range from 0.1 to 1.0 and improve over time based on feedback.

## Error Categories

Remediation supports categorizing errors for better organization:

| Category | Use Case | Examples |
|----------|----------|----------|
| `ErrorCompile` | Compile-time errors | Type errors, syntax errors |
| `ErrorRuntime` | Runtime errors | Panics, nil pointers, index out of bounds |
| `ErrorTest` | Test failures | Flaky tests, assertion failures |
| `ErrorLint` | Linter errors | golangci-lint, eslint violations |
| `ErrorSecurity` | Security issues | SQL injection, XSS, auth bypass |
| `ErrorPerformance` | Performance issues | Slow queries, memory leaks |
| `ErrorOther` | Other errors | Anything else |

## Scope Hierarchy

Remediation supports three scope levels for sharing fixes:

```
┌──────────────────────────────────────────────────────┐
│                   Organization                        │
│  (Share across all teams in org)                     │
│                                                       │
│  ┌────────────────────────────────────────────────┐  │
│  │              Team                               │  │
│  │  (Share across team projects)                  │  │
│  │                                                 │  │
│  │  ┌──────────────────────────────────────────┐  │  │
│  │  │          Project                         │  │  │
│  │  │  (Project-specific fixes)                │  │  │
│  │  └──────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

### Choosing a Scope

| Scope | When to Use | Example |
|-------|-------------|---------|
| **Project** | Fix is specific to this project | "Fix auth bug in user service" |
| **Team** | Fix applies to team's shared patterns | "Database connection pool config" |
| **Org** | Fix applies to all projects | "Flaky test timeout adjustment" |

### Hierarchy Search

When `IncludeHierarchy: true`, searches cascade up:

- **Project scope**: searches project → team → org
- **Team scope**: searches team → org
- **Org scope**: searches org only

This ensures you find relevant fixes at any level.

## Real-World Usage

### Example 1: New Team Member Onboarding

```go
// New developer encounters mysterious test failure
searchReq := &remediation.SearchRequest{
    Query:       "test timeout context deadline exceeded",
    Category:    remediation.ErrorTest,
    TenantID:    tenant,
    TeamID:      teamID,
    Scope:       remediation.ScopeTeam,
    IncludeHierarchy: true,
}

results, _ := service.Search(ctx, searchReq)

// Finds: "Flaky integration test: increase timeout to 30s"
// New developer solves issue in 5 minutes instead of 2 hours
```

### Example 2: Cross-Service Learning

```go
// Developer A fixes connection pool issue in Service X
recordReq := &remediation.RecordRequest{
    Title:       "Database connection pool exhaustion",
    Problem:     "pq: connection pool exhausted under load",
    Solution:    "Increase pool size to 50 and add connection lifetime",
    Scope:       remediation.ScopeTeam, // Share with team
    Category:    remediation.ErrorPerformance,
    Tags:        []string{"database", "performance"},
    // ... other fields
}
service.Record(ctx, recordReq)

// 2 weeks later, Developer B encounters same issue in Service Y
searchReq := &remediation.SearchRequest{
    Query:    "database connection pool timeout",
    Category: remediation.ErrorPerformance,
    Scope:    remediation.ScopeTeam,
}

results, _ := service.Search(ctx, searchReq)
// Instantly finds Developer A's fix, applies it to Service Y
```

### Example 3: Security Vulnerability Pattern

```go
// Security team discovers SQL injection pattern
recordReq := &remediation.RecordRequest{
    Title:    "SQL injection in user query",
    Problem:  "User input not sanitized in raw SQL query",
    RootCause: "String concatenation instead of parameterized query",
    Solution: "Use parameterized queries with $1, $2 placeholders",
    CodeDiff: `diff --git a/handlers/users.go
-    query := "SELECT * FROM users WHERE name = '" + name + "'"
+    query := "SELECT * FROM users WHERE name = $1"
+    rows, err := db.Query(query, name)`,
    Category: remediation.ErrorSecurity,
    Scope:    remediation.ScopeOrg, // Critical: share org-wide
    Tags:     []string{"security", "sql-injection"},
}

service.Record(ctx, recordReq)

// Any developer searching for SQL issues will find this
// Prevents repeating the same vulnerability across projects
```

### Example 4: CI/CD Pipeline Debugging

```go
// CI pipeline fails with "permission denied" error
searchReq := &remediation.SearchRequest{
    Query: "ci pipeline error permission denied docker",
    Tags:  []string{"ci", "docker"},
    TenantID: tenant,
    Scope: remediation.ScopeOrg,
}

results, _ := service.Search(ctx, searchReq)

// Finds previous fix: "Add docker group to CI user"
// Solution applied in 2 minutes instead of debugging for hours
```

## Integration with MCP

When running as an MCP server, these operations are exposed as tools:

| Go Method | MCP Tool | Purpose |
|-----------|----------|---------|
| `service.Record()` | `remediation_record` | Save error fix pattern |
| `service.Search()` | `remediation_search` | Find similar errors |
| `service.Feedback()` | `remediation_feedback` | Improve confidence scores |

Example MCP tool call:

```json
{
  "tool": "remediation_search",
  "arguments": {
    "query": "panic nil pointer error",
    "limit": 5,
    "min_confidence": 0.3,
    "category": "runtime",
    "tenant_id": "myorg",
    "team_id": "backend-team",
    "project_path": "/path/to/project",
    "scope": "project",
    "include_hierarchy": true
  }
}
```

## Best Practices

### Do's ✅

1. **Record after solving** - Capture fixes when the solution is fresh in your mind
2. **Write clear problems** - Describe symptoms and error messages accurately
3. **Include root causes** - Explain why the error occurred, not just how to fix it
4. **Add code diffs** - Show exact changes that fixed the issue
5. **Use appropriate scopes** - Project for specific, team for patterns, org for critical
6. **Provide feedback** - Rate solutions to improve confidence over time
7. **Use semantic queries** - Search with natural language, not exact error text
8. **Tag appropriately** - Add tags for better categorization and filtering

### Don'ts ❌

1. **Don't record trivial fixes** - Typos and obvious errors aren't worth capturing
2. **Don't skip root causes** - Understanding "why" is as important as "how to fix"
3. **Don't use vague titles** - "Fix bug" is useless; "Fix nil pointer in auth middleware" is good
4. **Don't forget symptoms** - They help match similar errors semantically
5. **Don't over-scope** - Don't make project-specific fixes org-wide
6. **Don't ignore feedback** - Confidence scores are only useful if you provide feedback
7. **Don't record unverified fixes** - Only record solutions that actually worked

## Confidence Scoring

Remediation uses confidence scores to surface the best solutions:

| Confidence | Meaning | Typical State |
|------------|---------|---------------|
| 0.9 - 1.0 | Highly trusted | 10+ helpful ratings, 0 outdated |
| 0.7 - 0.9 | Reliable | 5+ helpful ratings, few not-helpful |
| 0.5 - 0.7 | Decent | Default score, some positive feedback |
| 0.3 - 0.5 | Questionable | Mixed feedback, may be outdated |
| 0.1 - 0.3 | Low trust | Multiple not-helpful ratings |

**Feedback impact:**
- Helpful: +0.1 confidence
- Not helpful: -0.1 confidence
- Outdated: -0.2 confidence

Use `MinConfidence` in search requests to filter low-quality results.

## Troubleshooting

### "No remediations found" on search

**Cause**: No similar errors recorded yet, or query is too specific.

**Solution**:
```go
// Broaden search query
searchReq.Query = "error database" // Instead of exact error message

// Lower confidence threshold
searchReq.MinConfidence = 0.1 // Instead of 0.5

// Enable hierarchy search
searchReq.IncludeHierarchy = true

// Remove category filter
searchReq.Category = "" // Search all categories
```

### Search returns irrelevant results

**Cause**: Query is too broad or semantic matching is loose.

**Solution**:
```go
// Add category filter
searchReq.Category = remediation.ErrorRuntime

// Add tags filter
searchReq.Tags = []string{"database", "timeout"}

// Increase confidence threshold
searchReq.MinConfidence = 0.7 // Only high-confidence fixes

// Use more specific query
searchReq.Query = "postgres connection pool exhausted timeout"
```

### Confidence score not changing

**Cause**: Feedback not being recorded correctly.

**Solution**:
```go
// Verify remediation ID is correct
rem, err := service.Get(ctx, tenant, remediationID)
if err != nil {
    log.Printf("Remediation not found: %v", err)
}

// Ensure tenant ID matches
feedbackReq.TenantID = rem.TenantID

// Check feedback was recorded
err := service.Feedback(ctx, feedbackReq)
if err != nil {
    log.Printf("Feedback failed: %v", err)
}
```

### "Tenant ID required" error

**Cause**: Multi-tenant isolation requires tenant context.

**Solution**:
```go
// Always provide tenant, team, and project IDs
searchReq := &remediation.SearchRequest{
    TenantID:    "myorg",      // Required
    TeamID:      "myteam",     // Required for team/project scope
    ProjectPath: "/my/project", // Required for project scope
    // ... other fields
}
```

## Advanced: Tags and Filtering

Use tags for fine-grained categorization:

```go
// Record with descriptive tags
recordReq := &remediation.RecordRequest{
    Title: "Redis connection timeout",
    Tags: []string{
        "redis",
        "cache",
        "timeout",
        "infrastructure",
        "production",
    },
    // ... other fields
}

// Search by multiple tags
searchReq := &remediation.SearchRequest{
    Query: "cache error",
    Tags:  []string{"redis", "timeout"}, // Match any tag
    // ... other fields
}
```

**Tag strategies:**
- **Technology tags**: redis, postgres, kafka, grpc
- **Component tags**: authentication, authorization, api, database
- **Environment tags**: production, staging, development
- **Severity tags**: critical, high-priority, low-priority
- **Team tags**: backend, frontend, devops, security

## Next Steps

- **Session Lifecycle**: See `examples/session-lifecycle/` for memory search/record patterns
- **Checkpoints**: See `examples/checkpoints/` for context snapshot management
- **Repository Indexing**: See `examples/repository-indexing/` for semantic code search
- **Context-Folding**: See `examples/context-folding/` for isolated subtask execution

## See Also

- Remediation spec: `docs/spec/remediation/`
- Remediation service: `internal/remediation/service.go`
- MCP handlers: `internal/mcp/handlers/remediation.go`
- Integration tests: `test/integration/framework/workflow.go`
