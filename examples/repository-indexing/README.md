# Repository Indexing Example

Semantic code search with intelligent grep fallback. Find code by what it does, not just what it's named.

## Overview

Repository indexing enables smart code search:

- **Index**: Build semantic embeddings of your codebase
- **Semantic Search**: Find code by meaning and intent
- **Grep Fallback**: Automatic fallback to pattern matching when needed
- **Project-Scoped**: Search within specific projects/repos

Stop grepping for function names. Search for concepts and let contextd find the code.

## Quick Start

### Prerequisites

- contextd installed and configured with Claude Code
- See [QUICKSTART.md](../../QUICKSTART.md) for setup instructions

### Pattern in Action

```
User: "Find the code that validates email addresses"

Claude: "Let me search the codebase semantically..."

[Uses MCP tool: semantic_search]
{
  "project_path": "/Users/me/my-api",
  "query": "email validation logic",
  "limit": 5
}

Response (Semantic Results):
- src/validators/email.go:45 (similarity: 0.92)
  "func ValidateEmail(email string) bool { ... }"

- src/auth/signup.go:78 (similarity: 0.85)
  "// Email validation before creating user account"

- tests/validators_test.go:120 (similarity: 0.80)
  "TestEmailValidation validates email format checking"

Claude: "Found email validation in validators/email.go..."
```

vs traditional grep:

```
$ grep -r "email" .
# Returns 500+ matches including:
# - Variable names containing "email"
# - Comments mentioning "email"
# - Strings with "email" in them
# - Completely unrelated code
```

Semantic search understands **intent**, not just text matching.

## Available MCP Tools

### repository_index

Build semantic index of a codebase for fast search.

**When to use**:
- First time searching a project
- After major code changes
- Periodically (daily/weekly) to keep index fresh

**Input**:
```json
{
  "project_path": "/Users/me/my-project",
  "project_id": "my-project",
  "include_patterns": ["*.go", "*.ts", "*.py"],
  "exclude_patterns": ["*_test.go", "*.generated.go", "node_modules/**"],
  "max_file_size_mb": 5
}
```

**Output**:
```json
{
  "files_indexed": 247,
  "total_chunks": 1834,
  "skipped_files": 12,
  "index_size_mb": 15.4,
  "duration_seconds": 8.2
}
```

**Indexing behavior**:
- Chunks large files into smaller segments
- Respects `.gitignore`, `.dockerignore`, `.contextdignore`
- Skips binary files automatically
- Creates semantic embeddings for each chunk
- Stores metadata (file, line numbers, function names)

---

### semantic_search

Search indexed codebase by meaning, with automatic grep fallback.

**When to use**: Whenever you need to find code by what it does

**Input**:
```json
{
  "project_path": "/Users/me/my-project",
  "query": "authentication middleware that checks JWT tokens",
  "limit": 10,
  "min_similarity": 0.6
}
```

**Output**:
```json
{
  "results": [
    {
      "file": "src/middleware/auth.go",
      "line_start": 45,
      "line_end": 67,
      "content": "func JWTAuthMiddleware(next http.Handler) http.Handler { ... }",
      "similarity": 0.91,
      "chunk_id": "chunk_abc123"
    }
  ],
  "search_type": "semantic",
  "fallback_used": false,
  "total_results": 3
}
```

**Automatic Fallback**:
If no semantic results above `min_similarity`, contextd automatically tries grep:
```json
{
  "results": [...],
  "search_type": "grep",
  "fallback_used": true,
  "grep_pattern": "authentication|auth|JWT"
}
```

---

### repository_search

Alias for `semantic_search` - same functionality.

---

## Real-World Examples

### Example 1: Finding Authentication Logic

```
User: "Where do we check if a user is authenticated?"

# Step 1: Semantic search
Claude uses: semantic_search(
  project_path="/Users/me/api",
  query="authentication check user logged in",
  limit=5
)

Results (Semantic):
1. middleware/auth.go:34 (0.94) - "func RequireAuth()"
2. middleware/auth.go:56 (0.89) - "func CheckUserSession()"
3. handlers/protected.go:12 (0.82) - "// Protected routes require authentication"
4. auth/session.go:78 (0.75) - "func ValidateSession()"

Claude: "Authentication is checked in middleware/auth.go with RequireAuth()..."
```

**Why semantic search wins**:
- Found relevant code even though "authenticated" vs "authentication" difference
- Ranked by relevance, not alphabetical
- Included related concepts (session validation)
- Skipped irrelevant matches (logs about auth, comments, etc.)

---

### Example 2: Finding Error Handling Patterns

```
User: "Show me how we handle database connection errors"

# Semantic search
Claude uses: semantic_search(
  project_path="/Users/me/api",
  query="database connection error handling retry logic",
  limit=5
)

Results:
1. database/pool.go:89 (0.93)
   ```go
   func (p *Pool) acquireConnection() (*sql.Conn, error) {
       for attempt := 0; attempt < maxRetries; attempt++ {
           conn, err := p.db.Conn(context.Background())
           if err == nil {
               return conn, nil
           }
           if isTransientError(err) {
               time.Sleep(backoffDuration(attempt))
               continue
           }
           return nil, fmt.Errorf("acquiring connection: %w", err)
       }
       return nil, ErrMaxRetriesExceeded
   }
   ```

2. database/health.go:45 (0.87) - Health check with error handling

3. config/database.go:23 (0.80) - Connection string validation
```

**What it found**:
- Actual retry logic (not just error handling)
- Connection acquisition (related concept)
- Health checks (monitoring aspect)

vs `grep "error"` which would return 1000+ useless matches.

---

### Example 3: Grep Fallback Example

```
User: "Find all files that import the 'jwt' package"

# Semantic search tries first
Claude uses: semantic_search(
  project_path="/Users/me/api",
  query="import jwt package",
  limit=10
)

# Semantic results don't match well (import statements are too generic)
# contextd automatically falls back to grep

Results (Grep Fallback):
search_type: "grep"
fallback_used: true
grep_pattern: "import.*jwt|jwt.*import"

Files found:
- middleware/auth.go:5 - "import \"github.com/golang-jwt/jwt/v5\""
- handlers/token.go:7 - "import jwt \"github.com/golang-jwt/jwt/v5\""
- tests/auth_test.go:9 - "import \"github.com/golang-jwt/jwt/v5\""

Claude: "Found 3 files importing the jwt package..."
```

**When fallback helps**:
- Exact string matches (imports, function names)
- Regex patterns needed
- Very specific technical terms
- No semantic context available

---

### Example 4: Complex Query

```
User: "Find code that makes HTTP requests to external APIs with retry logic"

# Semantic search
Claude uses: semantic_search(
  project_path="/Users/me/api",
  query="HTTP client external API calls with retry and backoff logic",
  limit=5
)

Results:
1. client/http.go:67 (0.95)
   ```go
   // RetryableClient wraps http.Client with exponential backoff
   func (c *RetryableClient) Do(req *http.Request) (*http.Response, error) {
       var lastErr error
       for attempt := 0; attempt < c.maxRetries; attempt++ {
           resp, err := c.client.Do(req)
           if err == nil && resp.StatusCode < 500 {
               return resp, nil
           }
           lastErr = err
           time.Sleep(c.backoff.Duration(attempt))
       }
       return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
   }
   ```

2. integrations/stripe.go:34 (0.89) - Stripe API calls with retry
3. integrations/sendgrid.go:56 (0.85) - SendGrid API with retry
4. client/backoff.go:12 (0.82) - Exponential backoff implementation

Claude: "Found the retryable HTTP client in client/http.go, and it's used by Stripe and SendGrid integrations..."
```

**Power of semantic search**:
- Combined multiple concepts (HTTP + external + retry + backoff)
- Found implementations, not just definitions
- Ranked by relevance to full query
- Included usage examples

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────┐
│         Need to Find Code                       │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Is project indexed? │
        └──────────┬───────────┘
                   │
          ┌────────┴─────────┐
          │                  │
          ▼                  ▼
    ┌──────────┐      ┌────────────────┐
    │   Yes    │      │  No - Index it │
    └────┬─────┘      └────────┬───────┘
         │                     │
         │                     ▼
         │          ┌──────────────────────┐
         │          │  repository_index    │
         │          └──────────┬───────────┘
         │                     │
         └──────────┬──────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  semantic_search     │  ← Search by meaning
         └──────────┬───────────┘
                    │
           ┌────────┴─────────┐
           │                  │
           ▼                  ▼
    ┌────────────┐     ┌────────────┐
    │Found Results│    │No Good     │
    │(Semantic)  │    │Matches     │
    └──────┬─────┘     └──────┬─────┘
           │                  │
           │                  ▼
           │          ┌────────────────┐
           │          │Auto Grep       │ ← Fallback
           │          │Fallback        │
           │          └────────┬───────┘
           │                   │
           └────────┬──────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  Review Results      │
         └──────────────────────┘
```

## Indexing Best Practices

### Include Patterns

**Common patterns**:
```json
{
  "include_patterns": [
    "*.go",      // Go source
    "*.py",      // Python
    "*.ts",      // TypeScript
    "*.tsx",     // React TSX
    "*.js",      // JavaScript
    "*.java",    // Java
    "*.rs",      // Rust
    "*.sql",     // SQL schemas
    "*.proto",   // Protocol buffers
    "README.md", // Documentation
    "*.yaml"     // Configurations
  ]
}
```

### Exclude Patterns

**Common exclusions**:
```json
{
  "exclude_patterns": [
    "*_test.go",        // Test files (optional)
    "*.generated.go",   // Generated code
    "*.pb.go",          // Protobuf generated
    "node_modules/**",  // Dependencies
    "vendor/**",        // Go vendor
    ".git/**",          // Git internals
    "dist/**",          // Build outputs
    "build/**",         // Build artifacts
    "*.min.js",         // Minified JS
    "coverage/**"       // Test coverage
  ]
}
```

### Reindexing Strategy

| Scenario | When to Reindex |
|----------|-----------------|
| **Daily development** | Once daily (morning) |
| **Major refactor** | Immediately after |
| **Adding new service** | Before searching new code |
| **CI/CD** | After each deploy |
| **Large team** | Multiple times daily |

### Index Size Management

```
Small project (< 100 files):
  Index size: ~5-10 MB
  Index time: 1-2 seconds
  Search time: < 100ms

Medium project (100-1000 files):
  Index size: ~50-100 MB
  Index time: 5-15 seconds
  Search time: 100-300ms

Large project (> 1000 files):
  Index size: 100-500 MB
  Index time: 15-60 seconds
  Search time: 300-1000ms
```

## Best Practices

### ✅ DO

- **Index before searching**: Can't search what isn't indexed
- **Use semantic search for concepts**: "rate limiting logic", not "RateLimiter"
- **Use descriptive queries**: More context = better results
- **Reindex regularly**: Keep index fresh with code changes
- **Review multiple results**: Top result isn't always the best
- **Combine with grep for exact matches**: Let fallback happen naturally

### ❌ DON'T

- **Don't index everything**: Exclude tests, generated code, dependencies
- **Don't use exact function names**: That's what grep is for
- **Don't set min_similarity too high**: 0.6-0.7 is usually good
- **Don't forget to reindex**: Stale index misses new code
- **Don't index binary files**: Wastes space and time

## Troubleshooting

### "Project not indexed" error

**Cause**: Haven't run `repository_index` yet

**Fix**:
```
Claude uses: repository_index(
  project_path="/Users/me/project",
  project_id="project"
)
```

---

### Semantic search returns nothing, grep fallback also empty

**Cause**: Query too specific or using wrong terms

**Fix**: Try broader queries:
```
# Instead of:
semantic_search(query="validateEmailAddressFormat")

# Try:
semantic_search(query="email validation")
```

---

### Search is slow (> 2 seconds)

**Possible causes**:
1. **Large index**: Consider excluding more files
2. **First search**: Embeddings load into memory (subsequent searches faster)
3. **Max file size too high**: Reduce `max_file_size_mb`

**Fix**:
```json
{
  "include_patterns": ["*.go"],  // Narrow patterns
  "exclude_patterns": ["*_test.go", "vendor/**"],  // Exclude more
  "max_file_size_mb": 2  // Lower limit
}
```

---

### Results don't match expectations

**Cause**: Semantic embeddings learn from code patterns

**Fix**:
- Use technical domain terms from your codebase
- Add more context to query
- Lower `min_similarity` to see more results
- Try multiple query phrasings

## Integration with Other Features

- **Remediation**: Use semantic_search to find where errors occur, then [remediation_search](../remediation/) for fixes
- **Session Lifecycle**: Record [memory_record](../session-lifecycle/) about useful code patterns you find
- **Context-Folding**: Use [branch_create](../context-folding/) for exploratory searches without cluttering main context

## Next Steps

- Learn [context-folding](../context-folding/) for isolated search sessions
- Try [session-lifecycle](../session-lifecycle/) for remembering code patterns
- Explore [remediation](../remediation/) for error-specific searches

---

**Remember**: Semantic search finds code by what it **does**, not what it's **named**. Think in concepts, not keywords.
