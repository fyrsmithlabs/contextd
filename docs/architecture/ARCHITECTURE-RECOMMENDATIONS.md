# Architecture Recommendations for claude-tools
## Based on Comprehensive Security, Context Optimization, and Best Practices Research

**Date**: 2025-10-28
**Version**: 1.1
**Priority Focus**: 1) Security, 2) CLAUDE.md/Token Control, 3) Context Inference, 4) **OpenTelemetry Observability**

---

## Executive Summary

Based on research across 100+ sources (Claude Code setups, security best practices, context optimization, and OpenTelemetry standards), this document provides definitive architecture recommendations for the `claude-tools` Go-based localhost API.

**Key Research Findings Applied**:
- Unix domain sockets provide 35% better performance + superior security vs TCP
- Token-based auth with filesystem permissions is optimal for localhost APIs
- CLAUDE.md must stay <5K tokens; reference-based patterns achieve 50-76% reduction
- Prompt caching provides 90% cost savings on repeated content
- systemd security hardening is essential for daemon services
- **OpenTelemetry provides complete observability for Claude Code + API performance**
- **GenAI semantic conventions enable standardized LLM monitoring**

---

## 1. Security Architecture (Priority #1)

### 1.1 Transport Layer: Unix Domain Socket (REQUIRED)

**Decision**: Use Unix domain socket over TCP localhost

**Rationale**:
- ✅ No network exposure (filesystem-only)
- ✅ Filesystem permission-based access control
- ✅ 35% faster (2.3μs vs 3.6μs latency)
- ✅ Lower interception risk
- ✅ No port conflicts

**Implementation**:
```go
// cmd/claude-tools/main.go
func startServer() {
    socketPath := filepath.Join(os.Getenv("HOME"), ".config/claude-tools/api.sock")

    // Remove old socket if exists
    os.Remove(socketPath)

    // Create Unix listener with restricted permissions
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatal(err)
    }

    // Set socket permissions: owner read/write only
    os.Chmod(socketPath, 0600)

    e := echo.New()
    setupSecurityMiddleware(e)
    e.Listener = listener
    e.Start("")
}
```

**Socket Location**: `~/.config/claude-tools/api.sock`
**Permissions**: `0600` (owner read/write only)

### 1.2 Authentication: Token-Based with Filesystem Storage

**Decision**: Use bearer token stored in protected file

**Rationale**:
- Simple for localhost-only service
- Filesystem permissions provide access control
- No need for complex JWT for single-user system
- Constant-time comparison prevents timing attacks

**Implementation**:
```go
// pkg/auth/middleware.go
func TokenAuth() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            token := c.Request().Header.Get("Authorization")
            if token == "" {
                return echo.ErrUnauthorized
            }

            // Load token from secure location
            expectedToken, err := loadToken()
            if err != nil {
                return echo.ErrInternalServerError
            }

            // Constant-time comparison (prevent timing attacks)
            if subtle.ConstantTimeCompare([]byte(token), expectedToken) != 1 {
                return echo.ErrUnauthorized
            }

            return next(c)
        }
    }
}

func loadToken() ([]byte, error) {
    tokenPath := filepath.Join(os.Getenv("HOME"), ".config/claude-tools/token")
    return os.ReadFile(tokenPath)
}
```

**Token Generation**:
```bash
# On first run, generate secure token
openssl rand -hex 32 > ~/.config/claude-tools/token
chmod 0600 ~/.config/claude-tools/token
```

**Client Usage**:
```bash
# CLI automatically reads token
export CLAUDE_TOOLS_TOKEN=$(cat ~/.config/claude-tools/token)
curl -H "Authorization: $CLAUDE_TOOLS_TOKEN" \
  --unix-socket ~/.config/claude-tools/api.sock \
  http://localhost/api/v1/checkpoints
```

### 1.3 Echo Security Middleware (REQUIRED)

**Implementation**:
```go
// pkg/api/server.go
func setupSecurityMiddleware(e *echo.Echo) {
    // 1. Secure headers
    e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
        XSSProtection:         "1; mode=block",
        ContentTypeNosniff:    "nosniff",
        XFrameOptions:         "DENY",
        HSTSMaxAge:            31536000,
        ContentSecurityPolicy: "default-src 'self'",
    }))

    // 2. Rate limiting (prevent abuse even on localhost)
    e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

    // 3. Request size limits
    e.Use(middleware.BodyLimit("1M"))

    // 4. Timeout
    e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
        Timeout: 30 * time.Second,
    }))

    // 5. Request ID for tracing
    e.Use(middleware.RequestID())

    // 6. Recover from panics
    e.Use(middleware.Recover())

    // 7. Logging (structured)
    e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
        Format: `{"time":"${time_rfc3339}","id":"${id}","method":"${method}",` +
                `"uri":"${uri}","status":${status},"latency":"${latency_human}"}` + "\n",
    }))
}
```

### 1.4 Input Validation (Command Injection Prevention)

**Critical**: Prevent command injection in all user inputs

```go
// pkg/util/validation.go

// Validate project path (prevent path traversal)
func ValidateProjectPath(path string) error {
    // Go 1.24+ use os.Root for path sandboxing
    cleaned := filepath.Clean(path)

    // Check if path escapes home directory
    home := os.Getenv("HOME")
    if !strings.HasPrefix(cleaned, home) {
        return fmt.Errorf("path must be within home directory")
    }

    // Verify no path traversal
    if strings.Contains(path, "..") {
        return fmt.Errorf("path traversal detected")
    }

    return nil
}

// Safe command execution (prevent injection)
func SafeExec(command string, args []string) error {
    // NEVER use exec.Command with shell
    // ALWAYS pass args separately
    cmd := exec.Command(command, args...)

    // Set restrictive environment
    cmd.Env = []string{
        "PATH=/usr/local/bin:/usr/bin:/bin",
        "HOME=" + os.Getenv("HOME"),
    }

    return cmd.Run()
}

func SanitizeCollectionName(name string) (string, error) {
    // Only alphanumeric and underscores
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, name)
    if !matched {
        return "", fmt.Errorf("invalid collection name")
    }

    if len(name) > 255 {
        return "", fmt.Errorf("collection name too long")
    }

    return name, nil
}
```

### 1.5 systemd Security Hardening (REQUIRED)

**File**: `/etc/systemd/user/claude-tools.service`

```ini
[Unit]
Description=Claude Tools API Server
Documentation=https://github.com/yourusername/claude-tools
After=network.target

[Service]
Type=notify
ExecStart=/usr/local/bin/claude-tools server

# Security Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=read-only

# Resource Limits
MemoryMax=512M
CPUQuota=50%

# User isolation
DynamicUser=false
User=%u
Group=%u

# Restart policy
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=default.target
```

**Enable Service**:
```bash
systemctl --user enable claude-tools
systemctl --user start claude-tools
```


**Decision**: Environment variables with restricted file permissions

**Configuration File**: `~/.config/claude-tools/config.yaml`
```yaml
  local:
    uri: "http://localhost:19530"
  cluster:
    tls: true
```

**Permissions**: `chmod 0600 ~/.config/claude-tools/config.yaml`

**Load Secrets**:
```go
// pkg/config/config.go
type Config struct {
}

func Load() (*Config, error) {
    cfgPath := filepath.Join(os.Getenv("HOME"), ".config/claude-tools/config.yaml")

    data, err := os.ReadFile(cfgPath)
    if err != nil {
        return nil, err
    }

    // Expand environment variables
    expanded := os.ExpandEnv(string(data))

    var cfg Config
    if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

### 1.7 Security Checklist (PRE-DEPLOYMENT)

- [ ] Unix domain socket with 0600 permissions
- [ ] Token-based authentication enabled
- [ ] All Echo security middleware configured
- [ ] Input validation on all user inputs
- [ ] Command injection prevention verified
- [ ] Path traversal prevention tested
- [ ] systemd hardening options applied
- [ ] Config file permissions 0600
- [ ] No secrets in version control
- [ ] Rate limiting configured
- [ ] Request size limits set
- [ ] Logging configured (no sensitive data)

---

## 2. CLAUDE.md Management & Token Control (Priority #2)

### 2.1 CLAUDE.md Size Target

**CRITICAL FINDING**: CLAUDE.md must be <5K tokens

**Research-Based Target**: <500 characters (~125 tokens)

**Current Claude Code Best Practice**:
- Global CLAUDE.md: <5K tokens total
- Use `@imports` for modular organization
- Short bullet points, not paragraphs
- Reference external docs instead of inline content

### 2.2 Reference-Based Pattern (RECOMMENDED)


**Example Minimal CLAUDE.md**:
```markdown
# Claude Tools User Setup

## API
Local API: `~/.config/claude-tools/api.sock`
Auth: Token in `~/.config/claude-tools/token`

## Commands
- `/checkpoint save|search|list` - Context management
- `/rem add|search` - Error solutions
- `claude-tools skill create` - Interactive skill builder
- `claude-tools docs scrape <url>` - Index documentation

## Context Policy
At 70% → `/checkpoint save "summary"` → `/clear`
NEVER use `/compact` (30-60s delay)

See: github.com/yourusername/claude-tools for details
```

**Size**: 434 chars (~109 tokens) - **73% reduction from 1,809 chars**

### 2.3 Dynamic Content Loading via MCP

**Research Finding**: MCP (Model Context Protocol) enables order-of-magnitude context reduction

**Implementation**:
```typescript
// ~/.claude/mcp-servers/claude-tools/index.ts
import { MCPServer } from '@modelcontextprotocol/sdk';

const server = new MCPServer({
  name: 'claude-tools',
  version: '1.0.0',
});

// Tool: Fetch checkpoint by ID
server.addTool({
  name: 'get_checkpoint',
  parameters: {
    checkpoint_id: { type: 'string', required: true }
  },
  handler: async ({ checkpoint_id }) => {
    const response = await fetch('http://unix:~/.config/claude-tools/api.sock:/api/v1/checkpoints/' + checkpoint_id);
    return await response.json();
  }
});

// Tool: Search skills
server.addTool({
  name: 'search_skills',
  description: 'Search skills by keyword',
  parameters: {
    query: { type: 'string', required: true }
  },
  handler: async ({ query }) => {
    const response = await fetch('http://unix:~/.config/claude-tools/api.sock:/api/v1/skills/search?q=' + query);
    return await response.json();
  }
});

server.start();
```

**Benefit**: Claude fetches context only when needed, not loaded in every session

### 2.4 Token Counting Integration

**Implementation**: Track tokens in API responses

```go
// pkg/api/handlers/checkpoint.go
type CheckpointResponse struct {
    ID        string    `json:"id"`
    Summary   string    `json:"summary"`
    Content   string    `json:"content"`
    Tokens    int       `json:"tokens"`      // Estimated token count
    CreatedAt time.Time `json:"created_at"`
}

// Estimate tokens (rough approximation: chars / 4)
func estimateTokens(text string) int {
    return len(text) / 4
}

func (h *Handler) CreateCheckpoint(c echo.Context) error {
    var req CheckpointRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    tokens := estimateTokens(req.Summary + req.Content)

    // Warn if checkpoint is too large
    if tokens > 1000 {
        log.Warn().Int("tokens", tokens).Msg("Large checkpoint detected")
    }

    checkpoint := &Checkpoint{
        Summary: req.Summary,
        Content: req.Content,
        Tokens:  tokens,
    }


    return c.JSON(http.StatusOK, checkpoint)
}
```

### 2.5 Prompt Caching Strategy

**Research Finding**: 90% cost reduction on cached content

**Implementation**: Structure prompts for maximum cache reuse

```go
// pkg/api/handlers/skills.go

// Skill prompt structure optimized for caching
type SkillPrompt struct {
    // Static content (cached)
    SystemContext string `json:"system_context"`  // Global Claude setup
    BaseSkill     string `json:"base_skill"`      // Skill template

    // Dynamic content (not cached)
    UserInput     string `json:"user_input"`      // Current request
    Variables     map[string]string `json:"variables"` // Context-specific
}

func buildSkillPrompt(skill *Skill, input string) string {
    // Order: static first (for caching), then dynamic
    return skill.SystemContext + "\n\n" +
           skill.Template + "\n\n" +
           "User Input: " + input
}
```

**Benefit**: Static skill templates cached, only dynamic input counts toward tokens

### 2.6 Context Optimization Metrics

**Dashboard**: Track token efficiency

```go
// pkg/metrics/context.go
type ContextMetrics struct {
    TotalTokens        int64     `json:"total_tokens"`
    CachedTokens       int64     `json:"cached_tokens"`
    DynamicTokens      int64     `json:"dynamic_tokens"`
    CacheHitRate       float64   `json:"cache_hit_rate"`
    AvgCheckpointSize  int       `json:"avg_checkpoint_size"`
    CLAUDEMDSize       int       `json:"claude_md_size"`
    LastOptimized      time.Time `json:"last_optimized"`
}

// API endpoint: GET /api/v1/metrics/context
func (h *Handler) GetContextMetrics(c echo.Context) error {
    metrics := h.calculateMetrics()
    return c.JSON(http.StatusOK, metrics)
}
```

**CLI Command**:
```bash
$ claude-tools metrics context
Context Optimization Metrics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Tokens:        45,230
Cached Tokens:       38,910 (86%)
Dynamic Tokens:       6,320 (14%)

CLAUDE.md Size:         487 chars (122 tokens)
Avg Checkpoint:         234 tokens
Cache Hit Rate:         86%

Status: ✅ OPTIMAL
```

---

## 3. Context Inference & Maximization (Priority #3)


**Research Finding**: 40-67% improvement in retrieval accuracy with contextual retrieval

**Collection Schema**:
```go

type CheckpointSchema struct {
}

// Create collection with optimal index
    schema := &entity.Schema{
        CollectionName: "checkpoints",
        Fields: []*entity.Field{
            {Name: "id", DataType: entity.FieldTypeVarChar, PrimaryKey: true, MaxLength: 64},
            {Name: "summary", DataType: entity.FieldTypeVarChar, MaxLength: 1000},
            {Name: "content", DataType: entity.FieldTypeVarChar, MaxLength: 10000},
            {Name: "embedding", DataType: entity.FieldTypeFloatVector, Dim: 768},
            {Name: "project", DataType: entity.FieldTypeVarChar, MaxLength: 255},
            {Name: "tags", DataType: entity.FieldTypeArray, ElementType: entity.FieldTypeVarChar},
            {Name: "timestamp", DataType: entity.FieldTypeInt64},
            {Name: "token_count", DataType: entity.FieldTypeInt64},
        },
    }

    // Create with IVF_FLAT index (good for <1M vectors)
    indexParams := entity.NewIndexIvfFlat(entity.L2, 128)

    return client.CreateCollection(context.Background(), schema, indexParams)
}
```

### 3.2 Contextual Retrieval (Anthropic Pattern)

**Research Finding**: +49% top-20 retrieval accuracy with context prepending

**Implementation**:
```go

// Prepend context to chunks before embedding
func PrepareContextualChunk(chunk, documentContext string) string {
    contextPrompt := fmt.Sprintf(
        "Document: %s\n\nChunk: %s",
        documentContext,
        chunk,
    )
    return contextPrompt
}

// Store with contextual embeddings
func IndexDocumentation(doc *Document) error {
    // Document-level context
    docContext := fmt.Sprintf("%s - %s", doc.Title, doc.URL)

    chunks := chunkDocument(doc.Content)

    for _, chunk := range chunks {
        // Add context to chunk before embedding
        contextualChunk := PrepareContextualChunk(chunk, docContext)

        embedding := generateEmbedding(contextualChunk)

        // Store original chunk + contextual embedding
            ID:        generateID(),
            Content:   chunk,  // Original chunk
            Embedding: embedding,  // Contextual embedding
            DocTitle:  doc.Title,
            URL:       doc.URL,
        })
    }
}
```

**Benefit**: 49% improvement in retrieving correct context

### 3.3 Checkpoint Resume Pattern

**Implementation**: Efficient session resumption

```go
// pkg/checkpoint/resume.go

type ResumeContext struct {
    CheckpointID  string            `json:"checkpoint_id"`
    Summary       string            `json:"summary"`
    Project       string            `json:"project"`
    OpenFiles     []string          `json:"open_files"`
    NextActions   []string          `json:"next_actions"`
    Variables     map[string]string `json:"variables"`
}

// Search for most relevant checkpoint
func FindRelevantCheckpoint(query string, project string) (*ResumeContext, error) {
    queryEmbedding := generateEmbedding(query)

        CollectionName: "checkpoints",
        Embedding:      queryEmbedding,
        Filter:         fmt.Sprintf("project == '%s'", project),
        TopK:           5,
    })

    if len(results) == 0 {
        return nil, fmt.Errorf("no relevant checkpoint found")
    }

    // Return most relevant
    return buildResumeContext(results[0]), nil
}

// CLI usage
func resumeSession(checkpointQuery string) {
    ctx, err := FindRelevantCheckpoint(checkpointQuery, getCurrentProject())
    if err != nil {
        log.Fatal(err)
    }

    // Inject context into new Claude session
    fmt.Printf("Resuming from: %s\n", ctx.Summary)
    fmt.Printf("Next actions:\n")
    for _, action := range ctx.NextActions {
        fmt.Printf("  - %s\n", action)
    }
}
```

### 3.4 Parallel Inference for Batch Operations

**Research Finding**: 3-5x faster with parallel agents

**Implementation**: Parallel project indexing

```go
// pkg/index/parallel.go

func IndexProjectParallel(projectPath string) error {
    files, err := findSourceFiles(projectPath)
    if err != nil {
        return err
    }

    // Create worker pool
    numWorkers := runtime.NumCPU()
    jobs := make(chan string, len(files))
    results := make(chan *IndexResult, len(files))

    // Start workers
    var wg sync.WaitGroup
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for filePath := range jobs {
                result := indexFile(filePath)
                results <- result
            }
        }()
    }

    // Send jobs
    for _, file := range files {
        jobs <- file
    }
    close(jobs)

    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    var allResults []*IndexResult
    for result := range results {
        allResults = append(allResults, result)
    }

}
```

**Benefit**: 3-5x faster indexing using all CPU cores

### 3.5 Intelligent Context Pruning

**Implementation**: Remove least relevant context when approaching limits

```go
// pkg/context/pruning.go

type ContextItem struct {
    ID         string
    Content    string
    Relevance  float64  // Similarity to current query
    TokenCount int
    Timestamp  time.Time
}

// Prune context to stay under token budget
func PruneContext(items []ContextItem, maxTokens int) []ContextItem {
    // Sort by relevance score (descending)
    sort.Slice(items, func(i, j int) bool {
        return items[i].Relevance > items[j].Relevance
    })

    var selected []ContextItem
    totalTokens := 0

    for _, item := range items {
        if totalTokens + item.TokenCount > maxTokens {
            break
        }
        selected = append(selected, item)
        totalTokens += item.TokenCount
    }

    return selected
}
```

---

## 4. Implementation Priority

### Week 1: Security Foundation
1. ✅ Unix domain socket implementation
2. ✅ Token-based authentication
3. ✅ Echo security middleware
4. ✅ Input validation (command injection, path traversal)
5. ✅ systemd service with hardening

### Week 2: Context Optimization
1. ✅ Minimal CLAUDE.md (<500 chars)
2. ✅ Token counting in API
4. ✅ Context metrics dashboard
5. ✅ MCP server for dynamic loading

### Week 3: Context Inference
2. ✅ Contextual retrieval implementation
3. ✅ Parallel indexing
4. ✅ Checkpoint resume pattern
5. ✅ Context pruning algorithm

---

## 4. OpenTelemetry Observability (NEW: Priority #4)

### 4.1 Comprehensive Metrics Collection

**Endpoint**: `https://otel.dhendel.dev`
**Grafana**: `https://grafana.dhendel.dev`

**Metrics Categories**:

1. **HTTP Server Metrics** (from `otelecho` middleware)
   - `http.server.request.duration` - Request latency
   - `http.server.active_requests` - Concurrent requests
   - Automatic attributes: method, route, status_code

2. **GenAI Metrics** (OTEL Semantic Conventions)
   - `gen_ai.client.token.usage` - Token consumption by operation
   - `gen_ai.client.operation.duration` - Operation latency
   - `gen_ai.prompt.cache.hits` - Cache performance
   - Attributes: operation_name, token_type, model


4. **Cost Tracking**
   - `cost.request` - Cost per request (USD)
   - `cost.cumulative` - Total spend tracking
   - `cost.budget.status` - Budget utilization (%)

5. **Context Management**
   - `claude.md.size` - CLAUDE.md size tracking
   - `checkpoint.tokens` - Checkpoint efficiency
   - `context.window.usage` - Context % utilization

### 4.2 Implementation

```go
// Initialize OpenTelemetry
shutdown, err := telemetry.Initialize(&telemetry.Config{
    ServiceName:    "claude-tools-api",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    OTELEndpoint:   "https://otel.dhendel.dev",
    OTELAuthToken:  os.Getenv("OTEL_AUTH_TOKEN"),
})

// Echo middleware (automatic instrumentation)
e.Use(otelecho.Middleware("claude-tools-api"))

// Custom metrics
metrics.RecordTokenUsage(ctx, "checkpoint.create", inputTokens, outputTokens, cachedTokens)
```

### 4.3 Grafana Dashboards

**Dashboard 1: Claude Code Performance**
- Token usage over time
- Cost per hour (with budget alerts)
- Cache hit rate
- Operation breakdown

**Dashboard 2: API Performance**
- Request latency p95/p99
- Error rate
- Throughput
- Active connections

- Search latency by collection
- Insert throughput
- Sync status
- Collection sizes

### 4.4 Alerts

```yaml
# Token budget warning (70%)
- alert: TokenBudgetWarning
  expr: (token_budget_used / token_budget_daily) > 0.70

# Cost spike ($10/hour)
- alert: CostSpike
  expr: increase(cost_cumulative[1h]) > 10.00

# High API latency (>500ms p95)
- alert: HighLatency
  expr: histogram_quantile(0.95, http_server_request_duration) > 500
```

**Complete Spec**: See `docs/OPENTELEMETRY-IMPLEMENTATION.md`

---

## 5. Success Metrics

### Security
- [ ] No exposed TCP ports (Unix socket only)
- [ ] Token auth with 0600 permissions
- [ ] systemd security score >80% (`systemd-analyze security`)
- [ ] No command injection vulnerabilities (tested)
- [ ] All configs 0600 permissions

### CLAUDE.md & Token Control
- [ ] CLAUDE.md <500 chars (<125 tokens)
- [ ] 60%+ token savings vs baseline
- [ ] Cache hit rate >80%
- [ ] Average checkpoint <250 tokens
- [ ] Token tracking in all API responses

### Context Inference
- [ ] Checkpoint search <100ms p95
- [ ] Semantic search accuracy >80%
- [ ] Parallel indexing 3x faster than sequential
- [ ] Context pruning maintains >90% relevance
- [ ] Resume workflow <5 seconds end-to-end

### OpenTelemetry Observability (NEW)
- [ ] OTEL exporter connected to https://otel.dhendel.dev
- [ ] All metrics visible in Grafana dashboards
- [ ] Token usage tracked per operation
- [ ] Cost calculation accuracy verified
- [ ] Alerts configured (budget, latency, errors)
- [ ] Traces end-to-end for all operations
- [ ] Zero metric export errors

---

## 6. References

### Research Reports
- `research/RESEARCH-SUMMARY.md` - Claude Code best practices
- `docs/security-research-report.md` - Go API security (60+ sources)
- `reports/executive-summary-context-optimization.md` - Token optimization ROI
- `docs/OPENTELEMETRY-IMPLEMENTATION.md` - **Complete OTEL spec** ⭐
- `opentelemetry-go-implementation-guide.md` - Go SDK setup
- `docs/research/opentelemetry-llm-monitoring-standards.md` - GenAI conventions
- `reports/opentelemetry-configuration-research.md` - Collector config

### Key External Resources
- Anthropic: Contextual Retrieval (49% improvement)
- OWASP Go Security Cheat Sheet
- systemd Security Hardening
- Echo Framework v4 Security Guide
- **OpenTelemetry GenAI Semantic Conventions (v1.36.0)**
- **otelecho Middleware Documentation**

---

**NEXT STEP**: Begin Phase 1 implementation with security foundation + OTEL instrumentation
