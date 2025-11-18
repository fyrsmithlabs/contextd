# MCP Workflows

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the lifecycle, initialization sequence, and common usage workflows for contextd's MCP integration.

---

## Server Lifecycle

### Initialization Sequence

1. **Parse Command-Line Flags**: Check for `--mcp` flag
2. **Load Configuration**: Environment variables and defaults
3. **Initialize Services**: Vector store, embedding, checkpoint, remediation, skills, troubleshooting
4. **Initialize OpenTelemetry**: Traces and metrics
5. **Create MCP Server**: Register all 16 tools
6. **Start HTTP Server**: Begin accepting requests on `/mcp` endpoint (port 8080)

**Initialization Code**:
```go
// Create MCP services struct
mcpServices := &mcp.Services{
    Checkpoint:      services.Checkpoint,
    Remediation:     services.Remediation,
    Troubleshooting: services.Troubleshooting,
    Skills:          services.Skills,
    Analytics:       services.Analytics,
}

// Create MCP server
mcpServer, err := mcp.NewServer(mcpServices)
if err != nil {
    return fmt.Errorf("failed to create MCP server: %w", err)
}

// Run server (blocking)
if err := mcpServer.Run(ctx); err != nil {
    return fmt.Errorf("server error: %w", err)
}
```

### Health Checks

The server performs health checks on initialization:

```go
if err := services.Checkpoint.Health(ctx); err != nil {
    return fmt.Errorf("checkpoint service health check failed: %w", err)
}
```

Health status is available via the `status` tool during runtime.

### Graceful Shutdown

1. **Signal Handling**: Graceful shutdown on SIGINT/SIGTERM
2. **Context Cancellation**: Cancel all in-flight operations
3. **Service Cleanup**: Close database connections
4. **Telemetry Flush**: Flush pending traces and metrics (5s timeout)
5. **Exit**: Clean process termination

**Shutdown Code**:
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

select {
case <-sigChan:
    cancel() // Cancel context
    // Cleanup via defer statements
}
```

---

## Claude Code Configuration

### MCP Server Setup

Add contextd MCP server to `~/.claude/config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "/usr/local/bin/contextd",
      "args": ["--mcp"],
      "env": {
        "EMBEDDING_BASE_URL": "http://localhost:8080/v1",
        "EMBEDDING_MODEL": "BAAI/bge-small-en-v1.5",
        "QDRANT_URI": "http://localhost:6333",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318"
      }
    }
  }
}
```

### Remote Access Configuration

For remote/distributed teams:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "/usr/local/bin/contextd",
      "args": ["--mcp"],
      "env": {
        "CONTEXTD_HTTP_PORT": "8080",
        "CONTEXTD_HTTP_HOST": "0.0.0.0",
        "EMBEDDING_BASE_URL": "http://localhost:8080/v1",
        "EMBEDDING_MODEL": "BAAI/bge-small-en-v1.5",
        "QDRANT_URI": "http://localhost:6333"
      }
    }
  }
}
```

**Production Recommendations**:
- Deploy behind reverse proxy with TLS (nginx/Caddy)
- Add authentication (Bearer token, JWT, OAuth)
- Use VPN or SSH tunnel for security
- Implement rate limiting and DDoS protection

---

## Common Workflows

### Workflow 1: Session Checkpoint Management

**Use Case**: Save and resume work sessions.

**Steps**:
1. **Save Checkpoint**: Store current session state
   ```javascript
   checkpoint_save({
     summary: "Implemented user authentication",
     description: "Added JWT-based auth with refresh tokens",
     project_path: "/home/user/project",
     tags: ["auth", "security", "backend"]
   })
   ```

2. **Search Checkpoints**: Find relevant past work
   ```javascript
   checkpoint_search({
     query: "authentication implementation",
     top_k: 5,
     project_path: "/home/user/project"
   })
   ```

3. **List Recent Checkpoints**: Browse recent sessions
   ```javascript
   checkpoint_list({
     limit: 10,
     project_path: "/home/user/project",
     sort_by: "created_at"
   })
   ```

---

### Workflow 2: Error Resolution

**Use Case**: Store and search error solutions.

**Steps**:
1. **Save Error Solution**: Store remediation after fixing error
   ```javascript
   remediation_save({
     error_message: "dial tcp 127.0.0.1:8080: connect: connection refused",
     error_type: "ConnectionError",
     solution: "Start the server: ./server",
     severity: "medium",
     tags: ["networking", "server"]
   })
   ```

2. **Search for Solutions**: Find similar errors when encountering new issue
   ```javascript
   remediation_search({
     error_message: "connection refused",
     limit: 5,
     min_score: 0.6
   })
   ```

3. **Review Match Details**: Examine match scores for transparency
   ```javascript
   // Result includes:
   // - match_score: Combined score (70% semantic + 30% string)
   // - semantic_score: Vector similarity score
   // - string_score: String matching score
   // - stack_trace_match: Whether stack traces match
   // - error_type_match: Whether error types match
   ```

---

### Workflow 3: AI-Powered Troubleshooting

**Use Case**: Diagnose complex errors with AI assistance.

**Steps**:
1. **Troubleshoot Error**: Submit error for AI analysis
   ```javascript
   troubleshoot({
     error_message: "panic: runtime error: invalid memory address",
     stack_trace: "goroutine 1 [running]:\nmain.processRequest(...)",
     context: {
       "go_version": "1.21",
       "os": "linux"
     },
     mode: "auto"
   })
   ```

2. **Review Diagnosis**: Examine root cause, hypotheses, and similar issues
   ```javascript
   // Result includes:
   // - root_cause: Identified root cause
   // - confidence: high|medium|low
   // - hypotheses: Ranked list of possible causes
   // - similar_issues: Matched problems from knowledge base
   // - recommended_actions: Step-by-step diagnostic/fix actions
   // - diagnostic_steps: Additional diagnostic steps
   ```

3. **Browse Patterns**: Learn from common issues
   ```javascript
   list_patterns({
     category: "concurrency",
     severity: "high",
     min_success_rate: 0.8,
     limit: 10
   })
   ```

---

### Workflow 4: Skills Management

**Use Case**: Create and apply reusable workflows.

**Steps**:
1. **Create Skill**: Document successful workflow
   ```javascript
   skill_create({
     name: "Debug Go Race Conditions",
     description: "Systematic approach to debugging race conditions in Go",
     content: "# Debug Race Conditions\n\n1. Run with -race flag...",
     version: "1.0.0",
     author: "contextd team",
     category: "debugging",
     tags: ["go", "concurrency", "debugging"]
   })
   ```

2. **Search Skills**: Find relevant workflow for current problem
   ```javascript
   skill_search({
     query: "debugging race conditions",
     category: "debugging",
     top_k: 3
   })
   ```

3. **Apply Skill**: Use skill and track success
   ```javascript
   skill_apply({
     id: "skill-123",
     success: true
   })
   ```

4. **Update Skill**: Improve skill based on feedback
   ```javascript
   skill_update({
     id: "skill-123",
     content: "# Debug Race Conditions\n\n1. Run with -race flag...\n2. Add data race detector annotations...",
     version: "1.1.0"
   })
   ```

5. **List Skills**: Browse available workflows
   ```javascript
   skill_list({
     category: "debugging",
     sort_by: "success_rate",
     limit: 20
   })
   ```

---

### Workflow 5: Repository Indexing

**Use Case**: Index codebase for semantic search.

**Steps**:
1. **Index Repository**: Create searchable checkpoints from files
   ```javascript
   index_repository({
     path: "/home/user/project",
     include_patterns: ["*.md", "*.go", "*.txt"],
     exclude_patterns: ["vendor/**", "*.log"],
     max_file_size: 1048576  // 1MB
   })
   ```

2. **Search Indexed Files**: Find relevant documentation/code
   ```javascript
   checkpoint_search({
     query: "authentication middleware implementation",
     project_path: "/home/user/project",
     top_k: 10
   })
   ```

---

### Workflow 6: Analytics & Monitoring

**Use Case**: Track usage and performance.

**Steps**:
1. **Get Analytics**: Review context optimization metrics
   ```javascript
   analytics_get({
     period: "weekly",
     project_path: "/home/user/project"
   })
   ```

2. **Review Metrics**: Examine token reduction, time saved, feature adoption
   ```javascript
   // Result includes:
   // - avg_token_reduction_pct: % reduction in token usage
   // - total_time_saved_min: Estimated time saved
   // - search_precision: Search quality metric
   // - estimated_cost_save_usd: Cost savings from reduced tokens
   // - top_features: Most-used features with success rates
   // - performance: Latency and success rate metrics
   ```

3. **Check Service Status**: Monitor health
   ```javascript
   status({})
   ```

---

## Rate Limiting Behavior

### Per-Connection, Per-Tool Limits

**Default Limits**:
- Default RPS: 10 requests per second per tool
- Default burst: 20 requests

**Tool-Specific Overrides**:
- `troubleshoot`: 2 RPS, 5 burst (expensive AI operations)
- `skill_create`: 5 RPS, 10 burst (large embeddings)

**Rate Limit Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "error": {
    "code": -32603,
    "message": "rate limit exceeded for tool: troubleshoot"
  }
}
```

**Strategy**: Implement exponential backoff when rate limited.

---

## Timeout Handling

### Operation-Specific Timeouts

| Operation | Timeout | Behavior on Timeout |
|-----------|---------|---------------------|
| checkpoint_save | 30s | Returns timeout error, operation canceled |
| checkpoint_search | 10s | Returns timeout error, partial results not returned |
| remediation_search | 10s | Returns timeout error, partial results not returned |
| troubleshoot | 60s | Returns timeout error, diagnosis incomplete |
| skill_create | 120s | Returns timeout error, skill not created |
| index_repository | 300s | Returns timeout error, indexing incomplete |

**Timeout Error Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "error": {
    "code": -32603,
    "message": "[timeout] checkpoint creation timed out",
    "data": {
      "category": "timeout",
      "message": "checkpoint creation timed out",
      "details": {}
    }
  }
}
```

---

## Summary

**Key Workflows**:
- ✅ Session checkpoint management (save, search, list)
- ✅ Error resolution (save solutions, search matches)
- ✅ AI-powered troubleshooting (diagnose, review patterns)
- ✅ Skills management (create, search, apply, update)
- ✅ Repository indexing (semantic search across codebase)
- ✅ Analytics & monitoring (track usage, performance)

**Lifecycle**:
- ✅ Initialization with health checks
- ✅ Graceful shutdown with telemetry flush
- ✅ Rate limiting per connection and tool
- ✅ Operation-specific timeouts
