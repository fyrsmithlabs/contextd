# Remediation Example

Learn from past errors and apply proven fixes automatically. Remediation tracks error patterns and their solutions, saving debugging time on recurring issues.

## Overview

Remediation enables error pattern reuse:

- **Record**: Save error patterns and their fixes
- **Search**: Find similar past errors and their solutions
- **Apply**: Use proven fixes for new occurrences
- **Feedback**: Track which fixes work in practice

Stop solving the same problems repeatedly. Let contextd remember what worked.

## Quick Start

### Prerequisites

- contextd installed and configured with Claude Code
- See [QUICKSTART.md](../../QUICKSTART.md) for setup instructions

### Pattern in Action

```
User: "The app crashed with: panic: runtime error: invalid memory address"

Claude: "Let me search for similar errors we've fixed before..."

[Uses MCP tool: remediation_search]
{
  "project_id": "my-app",
  "error_message": "panic: runtime error: invalid memory address",
  "limit": 3
}

Response:
- "nil pointer dereference in user handler" (confidence: 0.92, similarity: 0.88)
  Fix: "Add nil check before accessing user.Profile"

- "missing database connection check" (confidence: 0.85, similarity: 0.75)
  Fix: "Initialize db connection in init() function"

Claude: "I found a similar issue! We fixed this before by adding nil checks..."

[Applies fix - adds nil check]
[Tests pass]

Claude: "Fixed! Let me record this specific variant..."

[Uses MCP tool: remediation_record]
{
  "project_id": "my-app",
  "error_type": "NilPointerError",
  "error_message": "panic: invalid memory address accessing payment.Method",
  "context": {
    "file": "handlers/payment.go",
    "line": 45,
    "function": "ProcessPayment"
  },
  "fix_description": "Added nil check for payment object before accessing Method field",
  "fix_code": "if payment == nil || payment.Method == \"\" { return ErrInvalidPayment }",
  "category": "null-safety",
  "severity": "high"
}

Result: Remediation recorded (ID: rem_abc123)

[Uses MCP tool: remediation_feedback]
{
  "remediation_id": "rem_def456",
  "helpful": true
}

Claude: "Marked the original fix as helpful (confidence boosted to 0.95)"
```

## Available MCP Tools

### remediation_search

Find past fixes for similar errors.

**When to use**: When encountering an error, before debugging manually

**Input**:
```json
{
  "project_id": "my-project",
  "error_message": "The error message you're seeing",
  "error_type": "NullPointerException",
  "category": "database",
  "limit": 5,
  "min_confidence": 0.6
}
```

**Output**:
```json
{
  "remediations": [
    {
      "id": "rem_abc123",
      "error_type": "NullPointerException",
      "error_message": "Cannot read property 'id' of null",
      "fix_description": "Added optional chaining and default values",
      "fix_code": "const userId = user?.id ?? 'anonymous'",
      "confidence": 0.88,
      "similarity": 0.92,
      "category": "null-safety",
      "times_helpful": 15,
      "created_at": "2024-01-10T12:00:00Z"
    }
  ]
}
```

**Search parameters**:
- `error_message`: Semantic search on error text
- `error_type`: Filter by type (NullPointer, Network, Database, etc.)
- `category`: Filter by category (authentication, database, api, etc.)
- `min_confidence`: Minimum confidence threshold (0.0-1.0)

---

### remediation_record

Save a new error fix for future reuse.

**When to use**: After successfully fixing an error

**Input**:
```json
{
  "project_id": "my-project",
  "error_type": "DatabaseError",
  "error_message": "connection timeout to postgres after 30s",
  "context": {
    "file": "database/connection.go",
    "line": 67,
    "function": "Connect",
    "stack_trace": "Optional full stack trace..."
  },
  "fix_description": "Increased connection timeout and added retry logic",
  "fix_code": "conn, err := sql.Open(\"postgres\", dsn, sql.ConnMaxLifetime(90*time.Second), sql.ConnRetries(3))",
  "root_cause": "Default 30s timeout too low for production load",
  "prevention": "Set higher timeout in production config",
  "category": "database",
  "severity": "high",
  "tags": ["postgres", "timeout", "connection-pool"]
}
```

**Output**:
```json
{
  "remediation_id": "rem_xyz789",
  "recorded": true
}
```

**Best practices**:
- Include specific error message (helps semantic search)
- Provide actual fix code, not just description
- Note root cause for understanding
- Add prevention tips
- Use consistent categories and tags

---

### remediation_feedback

Mark a remediation as helpful or not.

**When to use**: After applying a remediation fix

**Input**:
```json
{
  "remediation_id": "rem_abc123",
  "helpful": true,
  "notes": "Fixed the issue completely, no side effects"
}
```

**Effect**:
- `helpful: true` → Confidence increases, ranks higher in searches
- `helpful: false` → Confidence decreases, may eventually be archived

---

## Real-World Examples

### Example 1: Nil Pointer Panic

```
# Error occurs
User: "App crashed: panic: runtime error: invalid memory address or nil pointer dereference"

# Step 1: Search for similar fixes
Claude uses: remediation_search(
  project_id="api",
  error_message="nil pointer dereference",
  category="null-safety",
  limit=3
)

Results:
- "nil check missing in user.GetProfile()" (0.90, 87% similar)
  Fix: "Add guard: if user == nil { return ErrUserNotFound }"

# Step 2: Apply adapted fix
Claude: "Adding nil check before accessing object..."

[Adds check]
[Tests pass]

# Step 3: Record this specific variant
Claude uses: remediation_record(
  error_message="panic: nil pointer at payment.ProcessTransaction",
  fix_description="Added nil check for payment object and card details",
  fix_code="if payment == nil || payment.Card == nil { return ErrInvalidPayment }",
  category="null-safety",
  severity="high"
)

# Step 4: Mark original as helpful
Claude uses: remediation_feedback(
  remediation_id="original-fix-id",
  helpful=true,
  notes="Same pattern, worked perfectly"
)
```

---

### Example 2: Database Connection Issues

```
User: "Database queries failing with: pq: connection refused"

# Step 1: Search
Claude uses: remediation_search(
  project_id="api",
  error_message="pq: connection refused database",
  category="database",
  limit=5
)

Results:
- "Database container not started" (0.88)
  Fix: "docker-compose up -d postgres"

- "Wrong connection string in .env" (0.82)
  Fix: "Verify DATABASE_URL format"

- "Connection pool exhausted" (0.75)
  Fix: "Increase max_connections in postgresql.conf"

# Step 2: Check each possibility
Claude: "Let me verify the database container..."

[Checks - container is running]

Claude: "Container is up. Checking connection string..."

[Checks .env - format is correct]

Claude: "Connection string looks right. Let me check pool settings..."

[Finds: max_connections = 5, but app has 10 workers]

Claude: "Found it! Connection pool too small for number of workers."

# Step 3: Apply fix
[Updates postgresql.conf: max_connections = 20]
[Restarts database]
[Tests pass]

# Step 4: Record specific fix
Claude uses: remediation_record(
  error_message="pq: connection refused - too many clients",
  fix_description="Increased max_connections to match worker count",
  fix_code="max_connections = 20  # In postgresql.conf",
  root_cause="Connection pool size (5) < worker count (10)",
  prevention="Set max_connections = workers * 2 as baseline",
  category="database",
  severity="high"
)
```

---

### Example 3: Flaky Test

```
User: "TestUserLogin fails randomly, but only in CI"

# Step 1: Search for flaky test fixes
Claude uses: remediation_search(
  project_id="api",
  error_message="TestUserLogin fails in CI",
  error_type="FlakyTest",
  limit=3
)

Results:
- "Race condition in test setup" (0.85)
  Fix: "Add time.Sleep(100ms) after starting test server"

- "Async operation not awaited" (0.80)
  Fix: "Use sync.WaitGroup for goroutines in tests"

# Step 2: Investigate
Claude: "Let me check for race conditions..."

[Examines test code]
[Finds: test server starts in goroutine, test runs immediately]

Claude: "Found race! Test runs before server ready."

# Step 3: Apply fix
[Adds channel-based synchronization]

```go
serverReady := make(chan bool)
go func() {
    server.Start()
    serverReady <- true
}()
<-serverReady  // Wait for server

// Now run test
response := client.Login(...)
```

[Runs test 20 times - all pass]

# Step 4: Record solution
Claude uses: remediation_record(
  error_type="FlakyTest",
  error_message="TestUserLogin intermittent failures in CI",
  fix_description="Added channel-based synchronization for test server startup",
  fix_code="<full code above>",
  root_cause="Race condition - test ran before server ready to accept connections",
  prevention="Always synchronize async test setup with channels or WaitGroups",
  category="testing",
  severity="medium",
  tags=["flaky-test", "race-condition", "ci"]
)
```

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────┐
│            Error Encountered                    │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  remediation_search  │  ← "Have we seen this before?"
        │  (Find past fixes)   │
        └──────────┬───────────┘
                   │
          ┌────────┴─────────┐
          │                  │
          ▼                  ▼
   ┌────────────┐     ┌────────────┐
   │ Found Fix  │     │ No Matches │
   └──────┬─────┘     └──────┬─────┘
          │                  │
          ▼                  ▼
   ┌────────────┐     ┌────────────┐
   │ Apply Fix  │     │Debug Fresh │
   └──────┬─────┘     └──────┬─────┘
          │                  │
          └────────┬─────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Did it work?        │
        └──────────┬───────────┘
                   │
          ┌────────┴─────────┐
          │                  │
          ▼                  ▼
    ┌──────────┐      ┌──────────┐
    │ Success  │      │ Failed   │
    └────┬─────┘      └────┬─────┘
         │                 │
         ▼                 └─────► Try next fix
  ┌──────────────────────┐         or debug
  │ remediation_record   │         manually
  │ (Save for future)    │
  └──────────┬───────────┘
             │
             ▼
  ┌──────────────────────┐
  │ remediation_feedback │  ← Mark original
  │ (Rate helpfulness)   │    as helpful
  └──────────────────────┘
```

## Error Categories

Organize remediations by category for better findability:

| Category | Examples |
|----------|----------|
| `null-safety` | Nil pointer dereferences, undefined access |
| `database` | Connection issues, query errors, migrations |
| `authentication` | Login failures, token issues, session problems |
| `api` | HTTP errors, timeout issues, rate limits |
| `network` | Connection refused, DNS failures, proxy issues |
| `testing` | Flaky tests, race conditions, mock issues |
| `performance` | Memory leaks, slow queries, N+1 problems |
| `security` | SQL injection, XSS, CSRF vulnerabilities |
| `deployment` | Container issues, environment variables, config |

## Severity Levels

Set appropriate severity for prioritization:

- **critical**: Production down, data loss risk
- **high**: Feature broken, significant user impact
- **medium**: Degraded functionality, workaround exists
- **low**: Minor issue, aesthetic problems
- **info**: Informational, no fix required

## Best Practices

### ✅ DO

- **Search before debugging**: Check for past fixes first
- **Record all fixes**: Even "obvious" ones help future you
- **Include code**: Show actual fix, not just description
- **Note root cause**: Explain why the error occurred
- **Add prevention tips**: How to avoid this in the future
- **Use consistent categories**: Easier to find later
- **Provide feedback**: Mark fixes as helpful/not helpful

### ❌ DON'T

- **Don't skip the search**: You might waste time solving a known problem
- **Don't record without fix code**: Description alone isn't enough
- **Don't forget context**: File, line, function help locate the issue
- **Don't use vague error messages**: Be specific for better semantic search
- **Don't ignore feedback**: System learns from your ratings

## Troubleshooting

### "No similar errors found" but I know we fixed this before

**Cause**: Error message text doesn't match semantically

**Fix**: Try different search terms:
```
# Instead of:
remediation_search(error_message="error in handler")

# Try:
remediation_search(
  error_message="nil pointer exception user handler",
  error_type="NilPointerError"
)
```

Use specific technical terms that likely appear in stored remediations.

---

### Found fix doesn't work for my case

**Cause**: Similar error, different root cause

**Fix**:
1. Mark the remediation as `helpful: false` with notes
2. Continue debugging
3. Record your actual fix as a new remediation
4. Reference the misleading one in notes

---

### Too many low-quality remediations

**Cause**: Recording every tiny fix

**Fix**: Only record reusable patterns:
- ✅ Record: "Nil check pattern for API responses"
- ❌ Don't record: "Fixed typo in variable name"

---

### Confidence scores seem wrong

**Cause**: Not providing feedback after applying fixes

**Fix**: Always call `remediation_feedback` after trying a fix:
- Worked perfectly → `helpful: true`
- Didn't work → `helpful: false` with notes

## Integration with Other Features

- **Session Lifecycle**: Combine with [memory_record](../session-lifecycle/) for general learnings
- **Checkpoints**: Save [checkpoint](../checkpoints/) before applying experimental fixes
- **Repository Search**: Use [semantic_search](../repository-indexing/) to find error locations in code

## Next Steps

- Try [repository-indexing](../repository-indexing/) for semantic code search
- Explore [context-folding](../context-folding/) for isolated debugging sessions
- Learn [session-lifecycle](../session-lifecycle/) for cross-session memory

---

**Remember**: Every error you fix is an investment. Record it, and you'll never solve it twice.
