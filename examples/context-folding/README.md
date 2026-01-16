# Context-Folding Example

Execute isolated subtasks with dedicated token budgets. Achieve 90%+ context compression by keeping only essential results.

## Overview

Context-folding creates isolated branches for complex subtasks:

- **Branch Create**: Start isolated subtask with token budget
- **Branch Execute**: Work happens in isolated context
- **Branch Return**: Return scrubbed summary (not full execution)
- **Budget Enforcement**: Automatic termination when budget exhausted

Think of it like function calls - the caller doesn't need to see every step, just the result.

## Quick Start

### Prerequisites

- contextd installed and configured with Claude Code
- See [QUICKSTART.md](../../QUICKSTART.md) for setup instructions

### Pattern in Action

```
User: "Search through all API handlers to find rate limiting code"

Claude: "This will require reading many files. Let me use a branch to keep main context clean..."

# Main context: 45,000 tokens used

[Uses MCP tool: branch_create]
{
  "name": "search-rate-limiting",
  "description": "Search all handler files for rate limiting implementation",
  "budget": 10000
}

Response: Branch created (ID: branch_abc123)

# Inside branch (isolated from main context):
Claude:
  - Reads handlers/user.go (1,200 tokens)
  - Reads handlers/auth.go (980 tokens)
  - Reads handlers/api.go (1,450 tokens)
  - Reads middleware/ratelimit.go (800 tokens)
  - Searches 6 more files (4,000 tokens)

  Total used in branch: 8,430 tokens

Claude: "Found it! Rate limiting in middleware/ratelimit.go using token bucket algorithm"

[Uses MCP tool: branch_return]
{
  "result": "Rate limiting found in middleware/ratelimit.go:34. Uses token bucket algorithm with redis backend. Config: 100 requests/minute per IP."
}

# Back to main context:
# Branch consumed 8,430 tokens but main context only grows by ~150 tokens (the result)!

Main context: 45,150 tokens (grew by 150 instead of 8,430 - 98% compression!)

Claude: "Found the rate limiting code without filling up the conversation..."
```

## Available MCP Tools

### branch_create

Start an isolated context branch with token budget.

**When to use**:
- Exploring multiple files to find something specific
- Researching API documentation
- Trying multiple debugging approaches
- Any task where the **process is verbose** but the **result is concise**

**Input**:
```json
{
  "name": "descriptive-branch-name",
  "description": "What you're trying to accomplish in this branch",
  "budget": 5000,
  "parent_branch": null
}
```

**Output**:
```json
{
  "branch_id": "branch_abc123",
  "budget": 5000,
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Budget guidelines**:
- **Small task** (read 1-2 files): 2,000-3,000 tokens
- **Medium task** (read 5-10 files): 5,000-8,000 tokens
- **Large task** (read 20+ files): 10,000-15,000 tokens
- **Research** (fetch docs, try approaches): 15,000-20,000 tokens

---

### branch_return

Exit branch and return scrubbed summary to parent context.

**When to use**: When you've completed the branch task

**Input**:
```json
{
  "result": "Concise summary of what you found/accomplished. Only include essential information.",
  "metadata": {
    "files_read": 12,
    "tokens_used": 8430,
    "success": true
  }
}
```

**Output**:
```json
{
  "branch_id": "branch_abc123",
  "tokens_consumed": 8430,
  "result_length": 152,
  "compression_ratio": 0.98,
  "scrubbing_performed": true
}
```

**Secret scrubbing**: All return content is automatically scrubbed for:
- API keys
- Passwords
- Tokens
- Private keys
- Connection strings
- Other secrets (via gitleaks rules)

---

### branch_status

Check branch budget and usage.

**When to use**: Monitor budget during long-running branch tasks

**Input**:
```json
{}
```

**Output**:
```json
{
  "branch_id": "branch_abc123",
  "name": "search-rate-limiting",
  "budget": 10000,
  "used": 6250,
  "remaining": 3750,
  "depth": 1,
  "parent_branch": null
}
```

---

## Real-World Examples

### Example 1: File Exploration

```
User: "Find the function that generates PDF receipts"

# Without context-folding (BAD):
Claude reads:
  - handlers/checkout.go (2,000 tokens)
  - services/payment.go (1,800 tokens)
  - services/receipt.go (1,500 tokens) ← Found it!
  - utils/pdf.go (1,200 tokens)
  - ... (explores 8 files total)

  Total: 10,500 tokens added to main context
  Result: "Found in services/receipt.go line 67"

# With context-folding (GOOD):
Claude uses: branch_create(
  name="find-pdf-receipt",
  description="Search for PDF receipt generation function",
  budget=12000
)

[In branch - reads same 8 files]

Claude uses: branch_return(
  result="PDF receipt generation found in services/receipt.go:67 - GeneratePDFReceipt() function. Uses go-pdf library, templates in assets/receipts/."
)

Main context grows by: ~120 tokens (not 10,500!)
Compression: 99%
```

---

### Example 2: API Documentation Research

```
User: "How do we implement OAuth2 with the Stripe API?"

# Main context: 30,000 tokens

Claude: "Let me research Stripe's OAuth2 docs..."

branch_create(
  name="research-stripe-oauth",
  description="Research Stripe OAuth2 implementation requirements",
  budget=15000
)

# Inside branch:
Claude:
  - Fetches Stripe OAuth2 guide (4,000 tokens)
  - Reads Stripe API reference (3,500 tokens)
  - Checks Stripe SDK docs (2,800 tokens)
  - Reviews example code (2,000 tokens)
  - Explores error handling docs (1,500 tokens)

  Total: 13,800 tokens

branch_return(
  result="Stripe OAuth2 requires: 1) Register app in dashboard for client_id/secret 2) Redirect users to /oauth/authorize 3) Exchange code for access token via /oauth/token 4) Store refresh token for long-term access 5) Use 'Stripe-Account' header for connected account requests. SDK: stripe-go v74+ has built-in OAuth helpers."
)

# Main context grows by: ~200 tokens
# Avoided: 13,800 tokens of documentation
# Compression: 98.5%
```

---

### Example 3: Trial-and-Error Debugging

```
User: "The test TestUserLogin is failing"

Claude: "Let me debug this in a branch to avoid cluttering context..."

branch_create(
  name="debug-test-failure",
  description="Try multiple approaches to fix TestUserLogin",
  budget=10000
)

# Inside branch:
Claude:
  # Approach 1: Check test setup
  [Reads test file - 800 tokens]
  [Tries adding setup code - didn't work - 500 tokens]

  # Approach 2: Check mock configuration
  [Reads mock setup - 600 tokens]
  [Tries fixing mocks - didn't work - 700 tokens]

  # Approach 3: Check async timing
  [Reads async code - 500 tokens]
  [Finds race condition - this is it! - 400 tokens]
  [Verifies fix works - 300 tokens]

  Total: 3,800 tokens (including dead ends)

branch_return(
  result="Fixed! Race condition in test setup - server.Start() is async but test runs immediately. Added channel-based sync: serverReady := make(chan bool). Fix verified with 10 test runs."
)

# Main context grows by: ~100 tokens
# Avoided: 3,800 tokens of trial-and-error
# Compression: 97%
```

---

### Example 4: Nested Branches

```
User: "Find all database queries that don't use prepared statements"

# Main context: 40,000 tokens

Claude: "This requires searching many files. Using a branch..."

branch_create(
  name="find-unsafe-queries",
  description="Search codebase for SQL queries without prepared statements",
  budget=15000
)

# Inside branch:
Claude: "I'll search by service to stay organized..."

  # Sub-branch 1: User service
  branch_create(
    name="search-user-service",
    description="Check user service for unsafe queries",
    budget=5000,
    parent_branch="find-unsafe-queries"
  )

  [Reads 8 files, finds 2 unsafe queries]

  branch_return(
    result="User service: 2 unsafe queries found in user_repo.go lines 45, 89"
  )

  # Sub-branch 2: Payment service
  branch_create(
    name="search-payment-service",
    budget=5000,
    parent_branch="find-unsafe-queries"
  )

  [Reads 6 files, finds 1 unsafe query]

  branch_return(
    result="Payment service: 1 unsafe query in payment_repo.go line 123"
  )

branch_return(
  result="Found 3 unsafe SQL queries total: user_repo.go:45, user_repo.go:89, payment_repo.go:123. All are string concatenation instead of prepared statements. Fix: use db.Prepare() or parameterized queries."
)

# Main context grows by: ~150 tokens
# Branch consumed: ~11,000 tokens (including nested branches)
# Compression: 98.6%
```

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────┐
│         Main Context (40K tokens)                │
│                                                  │
│  User: "Find auth middleware"                   │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │  branch_create(budget=5000)                │ │
│  └───────────────────┬────────────────────────┘ │
│                      │                           │
│         ┌────────────▼──────────────┐           │
│         │   Branch Context (0K)    │           │
│         │                           │           │
│         │  • Read file 1 (1.2K)    │           │
│         │  • Read file 2 (0.9K)    │           │
│         │  • Search 5 more (2.8K)  │           │
│         │  • Total used: 4.9K      │           │
│         │                           │           │
│         │  Found in: auth.go:34    │           │
│         └────────────┬──────────────┘           │
│                      │                           │
│  ┌───────────────────▼────────────────────────┐ │
│  │  branch_return(result="Found in auth.go") │ │
│  └────────────────────────────────────────────┘ │
│                                                  │
│  Main Context: 40.1K tokens (grew by 100, not 4900!) │
│                                                  │
│  Compression: 98% ✓                             │
└─────────────────────────────────────────────────┘
```

## Use Cases by Category

### ✅ Use Context-Folding For

| Task | Why | Example Budget |
|------|-----|----------------|
| **File exploration** | Read many files, return one location | 5,000-10,000 |
| **API research** | Fetch docs, return summary | 10,000-15,000 |
| **Trial-and-error debugging** | Try approaches, return solution | 8,000-12,000 |
| **Multi-file search** | Search codebase, return matches | 10,000-15,000 |
| **Comparative analysis** | Compare options, return recommendation | 8,000-12,000 |

### ❌ Don't Use Context-Folding For

| Task | Why | Better Approach |
|------|-----|-----------------|
| **Single file edit** | Overhead not worth it | Direct edit in main context |
| **User needs to see process** | Defeats the purpose | Show work in main context |
| **Already simple task** | No benefit from isolation | Work in main context |
| **Critical debugging** | User wants full visibility | Main context with checkpoints |

## Budget Management

### Monitoring Budget

```
# Check remaining budget during long task
Claude uses: branch_status()

Response:
{
  "budget": 10000,
  "used": 7500,
  "remaining": 2500  # 25% left - might need to wrap up soon
}
```

### Budget Exhaustion

When budget is exhausted:
1. Branch is automatically force-returned
2. Partial results returned to parent
3. Warning included in return metadata

```json
{
  "result": "[PARTIAL] Found 3 of estimated 10 files before budget exhausted: auth.go, user.go, token.go...",
  "metadata": {
    "budget_exhausted": true,
    "tokens_used": 10000,
    "completion_estimate": "30%"
  }
}
```

### Budget Optimization

**Too small**: Task can't complete
```
budget: 2000
task: Read 10 files (each ~800 tokens)
result: Budget exhausted after 2 files
```

**Too large**: Wastes budget allocation
```
budget: 20000
task: Read 1 file (~1000 tokens)
result: 95% of budget unused
```

**Right-sized**: Completes with ~20% buffer
```
budget: 8000
task: Read 8 files (~6000 tokens) + exploration (~1500 tokens)
result: Completes with 500 tokens remaining
```

## Security Features

### Automatic Secret Scrubbing

All `branch_return()` content is scrubbed for:

| Secret Type | Pattern Examples |
|-------------|------------------|
| **API Keys** | `api_key=sk_live_...`, `STRIPE_KEY=...` |
| **Passwords** | `password=...`, `pwd=...` |
| **Tokens** | `token=ghp_...`, `bearer eyJ...` |
| **Private Keys** | `-----BEGIN RSA PRIVATE KEY-----` |
| **Connection Strings** | `postgres://user:pass@host/db` |
| **AWS Credentials** | `AKIA...`, `aws_secret_access_key=...` |

**Example**:
```
# Before scrubbing:
"Database connection: postgres://admin:MyP@ssw0rd!@db.example.com/prod"

# After scrubbing:
"Database connection: postgres://[REDACTED]@db.example.com/prod"
```

### Rate Limiting

- **Max concurrent branches**: 10 per session
- **Max nesting depth**: 3 levels
- **Max branch lifetime**: 30 minutes
- **Max budget per branch**: 50,000 tokens

Prevents resource exhaustion and runaway processes.

## Best Practices

### ✅ DO

- **Use for verbose exploration**: When process is noisy but result is simple
- **Set appropriate budgets**: Estimate token needs + 20% buffer
- **Return concise results**: Only essential findings, not full execution log
- **Name branches descriptively**: Clear names help debugging
- **Monitor large branches**: Use `branch_status()` for long tasks
- **Use nested branches for organization**: Helps structure complex searches

### ❌ DON'T

- **Don't overuse**: Not every task needs a branch
- **Don't return full process**: Defeats compression purpose
- **Don't set huge budgets "just in case"**: Wastes allocation
- **Don't nest too deeply**: Max 3 levels for clarity
- **Don't forget to return**: Branch results aren't visible until return

## Troubleshooting

### "Budget exceeded" error mid-task

**Cause**: Budget too small for task

**Fix**: Either:
1. **Increase budget** on next attempt
2. **Break into smaller branches** (nested approach)
3. **Return partial results** when nearing limit

---

### Branch results not visible in main context

**Cause**: Forgot to call `branch_return()`

**Fix**: Always close branches with return:
```
branch_create(...)
[do work]
branch_return(result="summary")  ← Required!
```

---

### "Max nesting depth exceeded"

**Cause**: Branches nested more than 3 levels deep

**Fix**: Flatten structure or return intermediate results:
```
# Instead of:
main → branch1 → branch2 → branch3 → branch4 (✗ Too deep)

# Use:
main → branch1 (completes, returns)
main → branch2 (completes, returns)
main → branch3 (completes, returns)
```

---

### Secret accidentally leaked in return

**Cause**: Scrubbing missed an unusual format

**Fix**: Manual scrub before return:
```
result = "Found config: database=prod_db, auth=[REDACTED]"
branch_return(result=result)
```

## Integration with Other Features

- **Repository Search**: Use [semantic_search](../repository-indexing/) inside branches to explore code
- **Checkpoints**: Save [checkpoint](../checkpoints/) before major branch-heavy operations
- **Session Lifecycle**: Record [memory_record](../session-lifecycle/) about useful patterns found in branches

## Next Steps

- Try [repository-indexing](../repository-indexing/) for semantic code search
- Learn [checkpoints](../checkpoints/) for longer-term context preservation
- Explore [session-lifecycle](../session-lifecycle/) for cross-session memory

---

**Remember**: Context-folding is like function scope - the caller doesn't need to see local variables, just the return value. Keep branches focused and results concise.
