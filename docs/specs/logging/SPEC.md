# Logging and Security Specification

> **Status:** Draft
> **Created:** 2025-01-13
> **Based On:** [2025-01-11 Comprehensive Logging Design](../../plans/2025-01-11-comprehensive-logging-security-design.md)
> **Related ADRs:** N/A (First implementation)

## 1. Purpose

Implement a production-grade logging and security system for contextd that:

1. **Prevents secret leakage** through 5 layers of defense (Claude Code hooks, MCP middleware, HTTP interceptor, audit trail)
2. **Provides structured logging** with Uber Zap (4-10x faster than stdlib)
3. **Integrates observability** via OpenTelemetry across Claude Code and contextd
4. **Enables audit compliance** with immutable event logs for GDPR/HIPAA/SOC2

**Primary Goals (Priority Order):**
1. Security: 1000% certainty for secrets detection (defense-in-depth)
2. Performance: <10ms scan latency, <1% overhead
3. Observability: Unified telemetry (logs, traces, metrics)
4. Usability: Simple installation (`ctxd hooks install`)

## 2. Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Claude Code                                                  │
│ - Sends tool calls to hooks                                 │
│ - Sends OTEL telemetry to collector                         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Claude Code Hooks (PreToolUse/PostToolUse)        │
│ - Ultra-thin: exec ctxd hooks --type={pre,post}-tool       │
│ - Exit code 2 = block, 0 = allow                           │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: ctxd CLI Hook Handler                             │
│ - Reads stdin, calls contextd HTTP API                     │
│ - POST /api/v1/hooks/{pre,post}-tool                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Contextd Hook API + Security Scanner              │
│ - Gitleaks secret scanning (800+ detectors, 88% recall)    │
│ - Policy engine (allowlist/blocklist)                      │
│ - Zap structured logging                                    │
│ - Audit event aggregation                                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 4: MCP Security Middleware                           │
│ - Wraps all MCP tool handlers                              │
│ - Scans input/output for secrets                           │
│ - Can redact or block                                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Layer 5: HTTP Client Interceptor                           │
│ - Pre-flight secret scanning BEFORE network I/O            │
│ - Blocks outbound API calls with secrets                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ OTEL Collector (Audit Trail)                               │
│ - Aggregates: Claude Code + contextd telemetry             │
│ - Exports: Grafana, Loki, Jaeger, Prometheus              │
│ - Immutable audit logs for compliance                       │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Component Relationships

```
pkg/logging/
├── logger.go          → Zap logger factory + config
├── audit.go           → Security audit event logging
├── http.go            → HTTP request/response logging middleware
└── rotation.go        → Log rotation with lumberjack

pkg/security/
├── scanner.go         → Gitleaks secret scanning
├── http_interceptor.go → HTTP client wrapper for pre-flight scanning
└── policy.go          → Policy engine (allowlist/blocklist)

internal/handlers/
└── hooks.go           → HTTP API for Claude Code hooks
                         POST /api/v1/hooks/pre-tool
                         POST /api/v1/hooks/post-tool

cmd/ctxd/
└── hooks.go           → CLI: ctxd hooks {install,execute,status,uninstall}

pkg/mcp/
└── security_middleware.go → MCP tool wrapper for input/output scanning

pkg/config/
└── logging.go         → Logging configuration schema
```

## 3. Key Components

### 3.1 pkg/logging/logger.go

**Purpose:** Centralized Zap logger with structured logging, rotation, sampling

**Public API:**

```go
package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// Config controls logging behavior
type Config struct {
    Level       string       // "debug", "info", "warn", "error"
    Encoding    string       // "json", "console"

    // Component-specific logging (independent services)
    LogHTTP          string // HTTP service log level: "debug", "info", "warn", "error", "off"
    LogHTTPPath      string // Separate log file for HTTP requests (optional)
    LogMCPCalls      string // MCP service log level: "debug", "info", "warn", "error", "off"
    LogMCPPath       string // Separate log file for MCP calls (optional)
    LogSecretStrips  bool   // Log when secrets are stripped/redacted
    LogAuditEvents   bool   // Log security audit events

    // Output
    OutputPaths      []string // ["stdout", "/var/log/contextd/app.log"]
    ErrorOutputPaths []string

    // Rotation (via lumberjack)
    MaxSizeMB   int
    MaxBackups  int
    MaxAgeDays  int
    Compress    bool

    // Performance
    Sampling    *SamplingConfig
    Development bool
}

// Note: HTTP and MCP logging now use standard log levels (debug, info, warn, error, off)
// instead of custom levels. This treats them as independent services.

type SamplingConfig struct {
    Initial    int // Log first N per second
    Thereafter int // Then 1 in N
}

// NewLogger creates configured Zap logger
func NewLogger(config Config) (*zap.Logger, error)

// NewDevelopmentLogger creates development logger (console, debug)
func NewDevelopmentLogger() (*zap.Logger, error)

// NewProductionLogger creates production logger (json, info, rotated)
func NewProductionLogger(logPath string) (*zap.Logger, error)
```

**Configuration File:** `~/.config/contextd/logging.json`

```json
{
  "level": "info",
  "encoding": "json",
  "features": {
    "http": "info",
    "http_path": "/var/log/contextd/http.log",
    "mcp_calls": "info",
    "mcp_path": "/var/log/contextd/mcp.log",
    "secret_strips": true,
    "audit_events": true
  },
  "output": {
    "paths": ["stdout", "/var/log/contextd/app.log"],
    "error_paths": ["stderr", "/var/log/contextd/error.log"],
    "rotation": {
      "max_size_mb": 100,
      "max_backups": 3,
      "max_age_days": 30,
      "compress": true
    }
  },
  "sampling": {
    "initial": 100,
    "thereafter": 100
  }
}
```

**Log Output Format (JSON):**

```json
{
  "timestamp": "2025-01-13T10:15:30.123Z",
  "level": "info",
  "logger": "contextd.hooks",
  "caller": "handlers/hooks.go:145",
  "message": "tool_allowed",
  "tool_type": "Bash",
  "command": "git status",
  "project_path": "/home/user/project",
  "scan_duration_ms": 5,
  "trace_id": "abc123def456"
}
```

### 3.2 pkg/logging/audit.go

**Purpose:** Security audit event logging with aggregation

**Public API:**

```go
package logging

import (
    "context"
    "time"
)

// AuditLogger provides security audit logging
type AuditLogger struct {
    logger     *zap.Logger
    aggregator AuditAggregator
}

// NewAuditLogger creates audit logger with external aggregator
func NewAuditLogger(logger *zap.Logger, aggregatorURL string) (*AuditLogger, error)

// LogSecretDetection logs secret detection events
func (a *AuditLogger) LogSecretDetection(ctx context.Context, event SecretDetectionEvent) error

// LogToolBlocked logs blocked tool calls
func (a *AuditLogger) LogToolBlocked(ctx context.Context, event ToolBlockedEvent) error

// SecretDetectionEvent represents a security event
type SecretDetectionEvent struct {
    Timestamp   time.Time
    Location    string   // "pre_tool.Bash", "mcp.checkpoint_save.output"
    SecretTypes []string // ["ssh-private-key", "aws-access-key"]
    Action      string   // "BLOCKED", "ALERT_ONLY", "REDACTED"
    TraceID     string
    ProjectPath string
}

// ToolBlockedEvent represents blocked tool call
type ToolBlockedEvent struct {
    Timestamp time.Time
    ToolType  string
    ToolInput map[string]interface{}
    Reason    string
    TraceID   string
}
```

**Audit Log Format:**

```json
{
  "timestamp": "2025-01-13T10:15:30.123Z",
  "level": "error",
  "logger": "contextd.audit",
  "message": "SECURITY_CRITICAL_secret_detected",
  "location": "pre_tool.Bash",
  "secret_types": ["ssh-private-key-path"],
  "action": "BLOCKED",
  "trace_id": "abc123def456",
  "project_path": "/home/user/project"
}
```

### 3.3 pkg/security/scanner.go

**Purpose:** Secret scanning using Gitleaks (800+ detectors, 88% recall)

**Public API:**

```go
package security

import (
    "context"
    "time"

    gitleaks "github.com/zricethezav/gitleaks/v8/detect"
    "go.uber.org/zap"
)

// Scanner provides secret detection
type Scanner struct {
    detector *gitleaks.Detector
    logger   *zap.Logger
    config   ScannerConfig
}

// ScannerConfig controls behavior
type ScannerConfig struct {
    ConfigPath    string   // Path to gitleaks.toml (custom patterns)
    RedactSecrets bool     // Redact vs just detect
    MinConfidence float64  // Threshold (0.0-1.0)
}

// NewScanner creates scanner with Gitleaks
func NewScanner(config ScannerConfig, logger *zap.Logger) (*Scanner, error)

// ScanText scans arbitrary text
func (s *Scanner) ScanText(ctx context.Context, text string) (*ScanResult, error)

// ScanFile scans a file
func (s *Scanner) ScanFile(ctx context.Context, filePath string) (*ScanResult, error)

// ScanResult contains detection results
type ScanResult struct {
    Found         bool
    Secrets       []Secret
    RedactedText  string
    Confidence    float64
    ScanDuration  time.Duration
}

// Secret represents a detected credential
type Secret struct {
    Type        string  // "aws_access_key", "openai_api_key"
    Match       string  // Redacted in logs
    Line        int
    Column      int
    Confidence  float64
    File        string
}
```

**Gitleaks Configuration:** `~/.config/contextd/gitleaks.toml`

```toml
# Custom patterns beyond Gitleaks defaults (800+ built-in)

[[rules]]
id = "contextd-api-key"
description = "Contextd API keys"
regex = '''(?i)(contextd[_-]?api[_-]?key)[\s:=]+['"]?([a-zA-Z0-9_\-]{32,})['"]?'''
keywords = ["contextd_api_key", "CONTEXTD_API_KEY"]

[[rules]]
id = "ssh-command-pattern"
description = "Commands reading SSH keys"
regex = '''(cat|less|more|tail|head|vim|nano)\s+.*\.ssh/'''
keywords = [".ssh", "id_rsa"]

[allowlist]
paths = [".env.example", "docs/"]
regexes = ['''example[_-]?key''', '''test[_-]?secret''']
```

### 3.4 internal/handlers/hooks.go

**Purpose:** HTTP API for Claude Code PreToolUse/PostToolUse hooks

**Public API:**

```go
package handlers

import (
    "github.com/labstack/echo/v4"
    "go.uber.org/zap"

    "github.com/axyzlabs/contextd/pkg/logging"
    "github.com/axyzlabs/contextd/pkg/security"
)

// HooksHandler handles hook API requests
type HooksHandler struct {
    scanner      *security.Scanner
    logger       *zap.Logger
    auditLogger  *logging.AuditLogger
    policyEngine *PolicyEngine
}

// NewHooksHandler creates handler
func NewHooksHandler(
    scanner *security.Scanner,
    logger *zap.Logger,
    auditLogger *logging.AuditLogger,
    policyEngine *PolicyEngine,
) *HooksHandler

// PreToolRequest from Claude Code
type PreToolRequest struct {
    ToolType  string                 `json:"tool_type"`
    ToolInput map[string]interface{} `json:"tool_input"`
    Context   map[string]string      `json:"context"`
}

// HookResponse to Claude Code
type HookResponse struct {
    Action   string   `json:"action"`    // "allow", "block", "redact"
    ExitCode int      `json:"exit_code"` // 0=allow, 2=block
    Message  string   `json:"message"`
    Reasons  []string `json:"reasons"`
}

// HandlePreToolUse handles PreToolUse hook
func (h *HooksHandler) HandlePreToolUse(c echo.Context) error

// HandlePostToolUse handles PostToolUse hook
func (h *HooksHandler) HandlePostToolUse(c echo.Context) error

// RegisterRoutes registers hook endpoints
func (h *HooksHandler) RegisterRoutes(g *echo.Group)
```

**HTTP Endpoints:**

```
POST /api/v1/hooks/pre-tool
POST /api/v1/hooks/post-tool
```

**Request Example:**

```json
{
  "tool_type": "Bash",
  "tool_input": {
    "command": "cat ~/.ssh/id_rsa",
    "description": "Read SSH private key"
  },
  "context": {
    "project_path": "/home/user/project",
    "session_id": "sess_abc123"
  }
}
```

**Response Example (Block):**

```json
{
  "action": "block",
  "exit_code": 2,
  "message": "🔒 SECURITY POLICY VIOLATION: Secrets detected\n\nFound 1 potential secret:\n  - ssh-private-key-path\n\nThis operation has been blocked to prevent credential leakage.",
  "reasons": ["ssh-private-key-path"]
}
```

**Response Example (Allow):**

```json
{
  "action": "allow",
  "exit_code": 0
}
```

### 3.5 cmd/ctxd/hooks.go

**Purpose:** CLI interface for Claude Code hook integration

**Commands:**

```bash
# Execute hook (called by hook scripts)
ctxd hooks execute --type=pre-tool < hook_input.json

# Install hooks
ctxd hooks install

# Check hook status
ctxd hooks status

# Uninstall hooks
ctxd hooks uninstall
```

**Hook Scripts (Auto-generated by install):**

`~/.claude/hooks/pre-tool-check.sh`:
```bash
#!/usr/bin/env bash
exec ctxd hooks execute --type=pre-tool
```

`~/.claude/hooks/post-tool-scan.sh`:
```bash
#!/usr/bin/env bash
exec ctxd hooks execute --type=post-tool
```

**Claude Settings Update:** `~/.claude/settings.json`

```json
{
  "hooks": {
    "preToolUse": "~/.claude/hooks/pre-tool-check.sh",
    "postToolUse": "~/.claude/hooks/post-tool-scan.sh"
  }
}
```

### 3.6 pkg/mcp/security_middleware.go

**Purpose:** MCP tool input/output scanning

**Public API:**

```go
package mcp

import (
    "context"

    "github.com/axyzlabs/contextd/pkg/security"
    "github.com/axyzlabs/contextd/pkg/logging"
)

// SecurityMiddleware wraps MCP tools with scanning
type SecurityMiddleware struct {
    scanner     *security.Scanner
    auditLogger *logging.AuditLogger
    logger      *zap.Logger
    config      SecurityConfig
}

type SecurityConfig struct {
    BlockSecretsInInput  bool
    BlockSecretsInOutput bool
    AlertOnSecrets       bool
}

// NewSecurityMiddleware creates middleware
func NewSecurityMiddleware(
    scanner *security.Scanner,
    auditLogger *logging.AuditLogger,
    logger *zap.Logger,
    config SecurityConfig,
) *SecurityMiddleware

// WrapToolHandler adds security to MCP tool
func (m *SecurityMiddleware) WrapToolHandler(
    toolName string,
    handler ToolHandler,
) ToolHandler
```

**Usage:**

```go
// Wrap all MCP tools
middleware := mcp.NewSecurityMiddleware(scanner, auditLogger, logger, config)

server.AddTool("checkpoint_save", middleware.WrapToolHandler(
    "checkpoint_save",
    handlers.HandleCheckpointSave,
))
```

### 3.7 pkg/security/http_interceptor.go

**Purpose:** Pre-flight secret scanning for outbound HTTP calls

**Public API:**

```go
package security

import (
    "net/http"

    "go.uber.org/zap"
)

// SecureHTTPClient wraps http.Client with secret scanning
type SecureHTTPClient struct {
    client        *http.Client
    logger        *zap.Logger
    scanner       *Scanner
    failOnSecrets bool
}

// NewSecureHTTPClient creates secure HTTP client
func NewSecureHTTPClient(
    client *http.Client,
    scanner *Scanner,
    logger *zap.Logger,
    failOnSecrets bool,
) *SecureHTTPClient

// Do intercepts HTTP requests for secret scanning
func (c *SecureHTTPClient) Do(req *http.Request) (*http.Response, error)
```

**Usage:**

```go
// Replace standard HTTP client
client := &http.Client{Timeout: 30 * time.Second}
secureClient := security.NewSecureHTTPClient(client, scanner, logger, true)

// All HTTP calls now pre-flight scanned
resp, err := secureClient.Do(req)
```

## 4. Data Flow

### 4.1 PreToolUse Hook Flow (Secret Detected)

```
1. User: "Read my SSH key"
2. Claude Code: Bash(cat ~/.ssh/id_rsa)
3. PreToolUse Hook: exec ctxd hooks execute --type=pre-tool
4. ctxd CLI: POST /api/v1/hooks/pre-tool with JSON input
5. Contextd API: Gitleaks scans command string
6. Gitleaks: Match "ssh-private-key-path" pattern
7. Policy Engine: BLOCK (dangerous file path)
8. Contextd API: Returns {action: "block", exit_code: 2}
9. ctxd CLI: Exit with code 2
10. PreToolUse Hook: Returns exit 2 to Claude Code
11. Claude Code: Tool execution BLOCKED
12. User sees: "🔒 SECURITY POLICY VIOLATION"
13. OTEL Collector: Audit log sent
```

### 4.2 MCP Tool Flow (Secret in Output)

```
1. Claude: mcp__contextd__checkpoint_search query="api keys"
2. MCP Handler: Executes search
3. Search Result: Contains "OPENAI_API_KEY=sk-abc123"
4. Security Middleware: Scans output JSON
5. Gitleaks: Detects "openai-api-key" pattern
6. Audit Logger: Logs SECURITY_CRITICAL event
7. Security Middleware: Redacts output if configured
8. Claude receives: "OPENAI_API_KEY=[REDACTED]"
9. OTEL Collector: Audit event exported
```

## 5. Configuration

### 5.1 Environment Variables

```bash
# Logging
CONTEXTD_LOG_LEVEL=info                    # debug, info, warn, error
CONTEXTD_LOG_ENCODING=json                 # json, console

# Component-specific logging (independent services)
CONTEXTD_LOG_HTTP=info                     # debug, info, warn, error, off
CONTEXTD_LOG_HTTP_PATH=~/.local/share/contextd/logs/http.log
CONTEXTD_LOG_MCP_CALLS=info                # debug, info, warn, error, off
CONTEXTD_LOG_MCP_PATH=~/.local/share/contextd/logs/mcp.log
CONTEXTD_LOG_SECRET_STRIPS=true
CONTEXTD_LOG_AUDIT_EVENTS=true

# Security
CONTEXTD_GITLEAKS_CONFIG=~/.config/contextd/gitleaks.toml
CONTEXTD_SECURITY_FAIL_CLOSED=true
CONTEXTD_SCAN_TIMEOUT=5s

# Hooks
CONTEXTD_HOOKS_ENABLED=true
CONTEXTD_HOOKS_TIMEOUT=5s

# OTEL
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=contextd
OTEL_ENVIRONMENT=production
```

### 5.2 File Locations

```
~/.config/contextd/
├── logging.json           # Logging configuration
├── security.json          # Security configuration
├── gitleaks.toml          # Custom Gitleaks patterns
└── hooks/
    └── policy.json        # Hook policy rules

~/.claude/
├── settings.json          # Claude Code settings (hooks config)
└── hooks/
    ├── pre-tool-check.sh  # PreToolUse hook script
    └── post-tool-scan.sh  # PostToolUse hook script

/var/log/contextd/
├── app.log                # Main application log
├── error.log              # Error-level logs
├── audit.log              # Security audit events
└── app.log.1.gz           # Rotated logs (compressed)
```

## 6. Testing Requirements

### 6.1 Unit Tests

**pkg/logging/logger_test.go:**
- Test logger creation with various configs
- Test log rotation trigger
- Test sampling behavior
- Test OTEL bridge integration

**pkg/security/scanner_test.go:**
- Test detection of 10+ secret types (SSH keys, API keys, tokens)
- Test redaction mode
- Test custom pattern loading
- Test allowlist/blocklist
- Test scan performance (<10ms for 10KB text)

**internal/handlers/hooks_test.go:**
- Test PreToolUse allow/block decisions
- Test PostToolUse alert generation
- Test policy engine rules
- Test timeout handling

### 6.2 Integration Tests

**test/integration/hooks_test.sh:**
```bash
# Start contextd
contextd &

# Install hooks
ctxd hooks install

# Test 1: Block SSH key read
echo '{"tool_type":"Bash","tool_input":{"command":"cat ~/.ssh/id_rsa"}}' \
  | ctxd hooks execute --type=pre-tool
assert_exit_code 2

# Test 2: Allow safe command
echo '{"tool_type":"Bash","tool_input":{"command":"git status"}}' \
  | ctxd hooks execute --type=pre-tool
assert_exit_code 0

# Test 3: Detect secret in output
echo '{"tool_type":"Bash","tool_output":"OPENAI_API_KEY=sk-abc123"}' \
  | ctxd hooks execute --type=post-tool
assert_audit_log_contains "openai-api-key"
```

### 6.3 Performance Benchmarks

**Requirements:**
- Scanner: <10ms per 10KB text scan
- Hook API: <50ms end-to-end latency
- HTTP interceptor: <5ms overhead per request
- Memory: <50MB additional memory usage

**Benchmark Tests:**

```go
func BenchmarkScanner_ScanText_10KB(b *testing.B) {
    scanner := security.NewScanner(config, logger)
    text := generateText(10000) // 10KB

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        scanner.ScanText(context.Background(), text)
    }
}
// Target: <10ms/op
```

## 7. Security Guarantees

### 7.1 What We CAN Guarantee

✅ **PreToolUse Hook**: Blocks dangerous commands BEFORE execution
- Pattern: `cat ~/.ssh/`, `echo $.*KEY`, `.env` files
- Exit code 2 prevents tool execution
- Claude never sees the secret

✅ **PostToolUse Hook**: Detects secrets in outputs with immediate alerts
- 88% recall rate (Gitleaks)
- 800+ detector patterns
- Alert within 5 seconds

✅ **MCP Middleware**: Redacts secrets in tool responses
- Scans all MCP tool outputs
- Configurable redaction
- Audit trail for compliance

✅ **HTTP Interceptor**: Blocks outbound API calls with secrets
- Pre-flight scanning BEFORE network I/O
- Prevents secrets from leaving machine
- OTEL trace correlation

✅ **OTEL Collector**: Immutable audit trail
- Every secret detection logged
- Correlated trace IDs across layers
- Exportable to external SIEM

### 7.2 What We CANNOT Guarantee

❌ **User Overrides**: User explicitly approves dangerous command after warning
❌ **Obfuscated Secrets**: Base64, hex-encoded, or split strings may bypass
❌ **Novel Secret Formats**: Patterns not in Gitleaks database (800+) or custom config
❌ **Claude's Internal Context**: Cannot erase secrets already in conversation memory

### 7.3 Mitigations for Limitations

**User Overrides:**
- Audit trail shows who approved
- Anomaly detection alerts
- Incident response playbook

**Obfuscated Secrets:**
- ML-based entropy detection (Gitleaks supports)
- Custom regex patterns
- Periodic manual review

**Novel Formats:**
- Continuous Gitleaks pattern updates
- Community-driven pattern sharing
- Per-project custom TOML config

## 8. Metrics and Observability

### 8.1 Key Metrics

**Security Metrics:**
```
contextd_secrets_detected_total{location="pre_tool",secret_type="ssh-key"}
contextd_secrets_detected_total{location="post_tool",secret_type="api-key"}
contextd_secrets_detected_total{location="mcp_output",secret_type="token"}
contextd_hooks_total{action="block",tool_type="Bash"}
contextd_hooks_total{action="allow",tool_type="Read"}
```

**Performance Metrics:**
```
contextd_scan_duration_seconds{layer="pre_tool"} histogram
contextd_scan_duration_seconds{layer="mcp"} histogram
contextd_hook_latency_seconds histogram
contextd_http_interceptor_overhead_seconds histogram
```

**Reliability Metrics:**
```
contextd_audit_failures_total counter
contextd_scanner_errors_total counter
contextd_hook_timeouts_total counter
```

### 8.2 Grafana Dashboard Panels

**Security Overview:**
- Secrets detected by type (pie chart)
- Blocked tool calls timeline (graph)
- Secret detection rate (gauge)
- Critical security events (table)

**Performance:**
- Scan latency P50/P95/P99 (graph)
- Hook API latency (heatmap)
- Memory usage (graph)
- CPU usage (graph)

**Audit Trail:**
- Recent security events (table with trace IDs)
- Failed audit log writes (counter)
- OTEL collector health (gauge)

### 8.3 Alerts

**Critical:**
- Secret detected in tool output → Slack critical
- Audit system unreachable → PagerDuty
- >10 blocked commands in 5 min → Slack warning

**Warning:**
- Hook scan duration >1s
- Scanner error rate >1%
- OTEL collector lag >10s

## 9. Migration and Rollout

### 9.1 Phase 1: Core Infrastructure (Week 1)

**Deliverables:**
- [ ] `pkg/logging/logger.go` with Zap integration
- [ ] `pkg/logging/audit.go` with audit event types
- [ ] `pkg/security/scanner.go` with Gitleaks integration
- [ ] Unit tests (>80% coverage)
- [ ] Configuration loading from `~/.config/contextd/`

**Success Criteria:**
- Logger creates structured JSON logs
- Scanner detects 10+ secret types
- Audit logger writes to file and OTEL

### 9.2 Phase 2: Hook System (Week 2)

**Deliverables:**
- [ ] `internal/handlers/hooks.go` with HTTP API
- [ ] `cmd/ctxd/hooks.go` with CLI commands
- [ ] Hook script generation
- [ ] Policy engine implementation
- [ ] Integration tests

**Success Criteria:**
- `ctxd hooks install` creates hook scripts
- PreToolUse blocks SSH key reads
- PostToolUse detects API keys in outputs
- End-to-end latency <50ms

### 9.3 Phase 3: MCP & HTTP Security (Week 3)

**Deliverables:**
- [ ] `pkg/mcp/security_middleware.go` wrapping all tools
- [ ] `pkg/security/http_interceptor.go` for outbound calls
- [ ] OTEL Zap bridge integration
- [ ] Performance benchmarks

**Success Criteria:**
- MCP tools scan input/output
- HTTP interceptor blocks API calls with secrets
- Logs exported to OTEL collector
- <5ms HTTP interceptor overhead

### 9.4 Phase 4: Documentation & Polish (Week 4)

**Deliverables:**
- [ ] Installation guide
- [ ] Configuration reference
- [ ] Troubleshooting guide
- [ ] Grafana dashboard JSON
- [ ] Runbook for incident response

**Success Criteria:**
- New user can install in <5 minutes
- All config options documented
- Grafana dashboard visualizes metrics

## 10. Future Enhancements

### 10.1 Short-term (Q1 2025)

**ML-based Secret Detection:**
- Train model on contextd-specific secrets
- Improve obfuscated secret detection
- Reduce false positive rate

**Enhanced Policy Engine:**
- Per-project allowlists/blocklists
- Time-based policies (e.g., block after hours)
- User-specific permissions

### 10.2 Long-term (2025)

**Distributed Tracing:**
- Trace secrets across multiple services
- Detect secret propagation chains

**Compliance Automation:**
- GDPR/HIPAA/SOC2 audit reports
- Automated compliance checks

**Secret Rotation Integration:**
- Automatic rotation when detected
- Integration with Vault, AWS Secrets Manager

## 11. Dependencies

### 11.1 Go Modules

```
go.uber.org/zap v1.27.0                           # Structured logging
gopkg.in/natefinch/lumberjack.v2 v2.2.1          # Log rotation
github.com/zricethezav/gitleaks/v8 v8.18.0       # Secret scanning
go.opentelemetry.io/otel v1.24.0                 # Observability
go.opentelemetry.io/otel/exporters/otlp v1.24.0  # OTEL exporter
```

### 11.2 External Services

**Required:**
- Qdrant (vector database) - Already in use
- OTEL Collector - New (can run in Docker)

**Optional:**
- Grafana (dashboards)
- Loki (log aggregation)
- Jaeger (trace visualization)
- Prometheus (metrics)

## 12. References

### 12.1 External Documentation

- [Claude Code Hooks Guide](https://docs.claude.com/claude-code/hooks)
- [Claude Code OTEL Monitoring](https://docs.claude.com/claude-code/monitoring)
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)
- [Uber Zap Documentation](https://pkg.go.dev/go.uber.org/zap)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)

### 12.2 Internal Documentation

- [Design Document](../../plans/2025-01-11-comprehensive-logging-security-design.md)
- [Package Guidelines](../../standards/package-guidelines.md)
- [Testing Standards](../../standards/testing-standards.md)
- [CLAUDE.md](../../../CLAUDE.md) - Project root documentation

## 13. Acceptance Criteria

### 13.1 Functional Requirements

- [ ] Logger produces structured JSON logs with rotation
- [ ] Scanner detects secrets with 88% recall (Gitleaks benchmark)
- [ ] PreToolUse hook blocks dangerous commands (exit code 2)
- [ ] PostToolUse hook alerts on secret detection
- [ ] MCP middleware scans all tool inputs/outputs
- [ ] HTTP interceptor blocks outbound calls with secrets
- [ ] Audit events exported to OTEL collector
- [ ] CLI installation: `ctxd hooks install` works end-to-end

### 13.2 Non-Functional Requirements

- [ ] Scan latency <10ms for 10KB text
- [ ] Hook API latency <50ms end-to-end
- [ ] Memory overhead <50MB
- [ ] Test coverage >80%
- [ ] Zero secrets leaked in integration tests
- [ ] Documentation complete and accurate

### 13.3 Security Requirements

- [ ] All 5 layers of defense operational
- [ ] Audit trail immutable (OTEL collector)
- [ ] Secret redaction working (no plaintext in logs)
- [ ] Hook scripts have 0600 permissions
- [ ] Config files have 0600 permissions
- [ ] No secrets in error messages

---

**Next Steps:** Create implementation plan with bite-sized tasks using `superpowers:writing-plans`.
