# Troubleshooting Requirements

**Parent**: [../SPEC.md](../SPEC.md)

## Core Philosophy

**Primary Goals**:
1. **Automated Learning**: Continuously learn from resolved errors to build knowledge base
2. **Intelligent Diagnosis**: Use AI to identify root causes and generate actionable recommendations
3. **Context Efficiency**: Reduce trial-and-error debugging time through semantic pattern matching
4. **Safety-First**: Detect and warn about destructive operations in recommended solutions

## Features and Capabilities

### 1. AI-Powered Error Diagnosis

- Semantic error pattern matching
- Root cause identification
- Hypothesis generation with probability scoring
- Verification step recommendations
- Solution generation based on similar issues

### 2. Troubleshooting Knowledge Base

- Store error patterns with solutions
- Organize by category (configuration, resource, dependency, permission, logic, network, storage)
- Classify by severity (critical, high, medium, low)
- Track success rates and usage patterns
- Tag for improved searchability

### 3. Intelligent Pattern Recognition

- Semantic search using vector embeddings
- Hybrid scoring combining similarity, success rate, and usage
- Metadata filtering (category, severity, tags)
- Reranking for optimal result ordering

### 4. Interactive Troubleshooting Sessions

- Track complete diagnostic sessions
- Record actions performed
- Capture resolution outcomes
- Store feedback for continuous learning

### 5. Observability and Monitoring

- OpenTelemetry instrumentation
- Metrics for diagnosis performance
- Pattern match tracking
- Success rate monitoring
- Hypothesis generation tracking

## Key Differentiators

- **Hybrid Matching**: Combines semantic similarity (60%), success rate (30%), and usage frequency (10%) for intelligent ranking
- **Progressive Disclosure**: Returns information based on confidence level - high confidence includes detailed timeline and affected resources
- **Safety Detection**: Automatically identifies destructive operations (delete, remove, drop, kill, restart) and adds warnings
- **Multi-Tenant Isolation**: Stores global troubleshooting knowledge in shared database accessible to all projects
- **Feedback Loop**: Tracks success rates and usage patterns to improve recommendations over time

## MCP Tool Integration

The troubleshooting service exposes two MCP tools:

### 1. troubleshoot

AI-powered error diagnosis.

- Analyzes error messages and stack traces
- Searches similar issues in knowledge base
- Generates hypotheses with evidence
- Recommends diagnostic steps and solutions
- Returns session ID for tracking

### 2. list_patterns

Browse troubleshooting patterns.

- Filter by category, severity, success rate
- Paginated results
- Useful for learning from past solutions

## Performance Requirements

### Target Response Times

| Operation | Target | Typical |
|-----------|--------|---------|
| Diagnose (full) | < 2s | 1.5s |
| Search Similar Issues | < 300ms | 200ms |
| Store Resolution | < 500ms | 300ms |
| List Patterns | < 100ms | 50ms |

### Scalability Requirements

**Vector Search Performance**:
- Small KB (<1000 patterns): <50ms search time
- Medium KB (1000-10000 patterns): 50-200ms search time
- Large KB (>10000 patterns): 200-500ms search time

**Resource Requirements**:
- Service base: ~50MB
- Per 1000 patterns: ~20MB (vectors + metadata)
- Storage per pattern: ~5KB (vector + metadata)

## Security Requirements

### Input Validation

1. **Error Message**: Max 10,000 characters, prevent injection
2. **Stack Trace**: Max 50,000 characters, sanitize paths
3. **Context**: Max 100 key-value pairs, sanitize values
4. **Category**: Enum validation (only predefined categories)
5. **Severity**: Enum validation (only critical/high/medium/low)
6. **Tags**: Max 20 tags, max 50 chars each

### Sensitive Data Handling

**Automatic Redaction**:
- API keys (pattern: `sk-...`, `key-...`)
- Tokens (pattern: `Bearer ...`, `token: ...`)
- Passwords (pattern: `password=...`, `pwd=...`)
- File paths with usernames (`/home/username/` → `/home/***/`)

### Filter Injection Prevention

- Troubleshooting patterns stored in isolated shared database
- No cross-project data access possible
- Database-level isolation prevents filter injection
- All filter values sanitized before building expressions

## Data Categories and Severity Levels

### Categories

```go
const (
    CategoryConfiguration = "configuration" // Config errors, missing env vars
    CategoryResource      = "resource"      // Out of memory, disk full
    CategoryDependency    = "dependency"    // Missing library, version mismatch
    CategoryPermission    = "permission"    // Access denied, file permissions
    CategoryLogic         = "logic"         // Nil pointer, index out of bounds
    CategoryNetwork       = "network"       // Connection refused, timeout
    CategoryStorage       = "storage"       // Database errors, file I/O
    CategoryGeneral       = "general"       // Uncategorized
)
```

### Severity Levels

```go
const (
    SeverityCritical = "critical" // Service crash, data loss, security breach
    SeverityHigh     = "high"     // Major feature broken, workaround exists
    SeverityMedium   = "medium"   // Minor feature broken, inconvenient
    SeverityLow      = "low"      // Cosmetic, edge case, minor annoyance
)
```
