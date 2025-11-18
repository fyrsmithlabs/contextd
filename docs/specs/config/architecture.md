# Configuration Architecture

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the configuration architecture, environment variables, and implementation patterns.

---

## Configuration Structure

### Complete YAML Schema

```yaml
# contextd Configuration
# Location: ~/.config/contextd/config.yaml
# Permissions: 0600 (required)
#
# All settings can be overridden with environment variables:
# Format: CONTEXTD_SECTION_SUBSECTION_KEY
# Example: CONTEXTD_HTTP_PORT

# Service identification
service:
  name: contextd
  version: 2.0.0
  environment: development  # development, staging, production
  log_level: info           # debug, info, warn, error

# Server configuration
server:
  http:
    port: 8080
    host: 0.0.0.0
    base_url: http://localhost:8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
    shutdown_timeout: 10s

  request:
    max_body_size: 10MB
    max_header_size: 1MB

  security:
    token_file: ~/.config/contextd/token
    require_token: true
    token_min_length: 32

# Database (Qdrant)
database:
  qdrant:
    host: localhost
    port: 6334
    api_key_file: ""
    use_tls: false
    timeout: 30s

  pool:
    max_connections: 10
    max_idle_connections: 5
    connection_timeout: 5s

  multi_tenant:
    enabled: true  # REQUIRED in v2.0+
    project_hash_algo: sha256

  backup:
    enabled: true
    interval: 24h
    retention_days: 30
    path: ~/.local/share/contextd/backups
    compress: true

# Embedding service
embedding:
  provider: tei  # tei or openai

  tei:
    base_url: http://localhost:8080
    model: BAAI/bge-large-en-v1.5
    timeout: 30s
    retry_attempts: 3
    retry_delay: 1s

  openai:
    api_key_file: ~/.config/contextd/openai_api_key
    model: text-embedding-3-small
    timeout: 30s
    max_retries: 3

  cache:
    enabled: true
    ttl: 24h
    max_entries: 10000

# Observability (OpenTelemetry)
observability:
  traces:
    enabled: true
    endpoint: http://localhost:4318
    sample_rate: 1.0
    batch_timeout: 5s
    max_batch_size: 512

  metrics:
    enabled: true
    endpoint: http://localhost:4318
    interval: 60s

  logging:
    format: json  # json or text
    output: stdout  # stdout, stderr, or file path
    add_source: true

  resource:
    service_name: contextd
    service_version: 2.0.0
    deployment_environment: development

# Feature flags
features:
  mcp_mode: false

  checkpoints:
    enabled: true
    auto_save_interval: 5m
    max_per_project: 1000

  remediations:
    enabled: true
    semantic_weight: 0.7
    string_weight: 0.3
    min_similarity: 0.6

  skills:
    enabled: true
    auto_index: true

  troubleshooting:
    enabled: true
    max_context_tokens: 4000

  indexing:
    enabled: true
    max_file_size: 1MB
    batch_size: 100

# Performance tuning
performance:
  workers:
    embedding: 4
    indexing: 2

  cache:
    embedding_ttl: 24h
    query_ttl: 5m

  rate_limit:
    enabled: false
    requests_per_second: 100
    burst: 200

# Development/debugging
development:
  debug:
    enabled: false
    pprof: false
    verbose_logging: false

  profile:
    cpu: false
    memory: false
    output_dir: /tmp/contextd/profiles

  mocks:
    embedding: false
    database: false
```

---

## Environment Variable Override

### Naming Convention

Pattern: `CONTEXTD_SECTION_SUBSECTION_KEY`

**Rules:**
1. Prefix: `CONTEXTD_`
2. Uppercase
3. Dots (`.`) → Underscores (`_`)
4. Nested sections separated by `_`

**Examples:**
```
YAML Path                     Environment Variable
───────────────────────────────────────────────────────
service.log_level             CONTEXTD_SERVICE_LOG_LEVEL
server.http.port              CONTEXTD_HTTP_PORT
database.qdrant.host          CONTEXTD_DATABASE_QDRANT_HOST
embedding.tei.base_url        CONTEXTD_EMBEDDING_TEI_BASE_URL
observability.traces.enabled  CONTEXTD_OBSERVABILITY_TRACES_ENABLED
```

### Precedence Rules

Configuration loaded in order (highest to lowest priority):

1. **Environment Variables** ← highest priority
2. **YAML Configuration File**
3. **Default Values** ← lowest priority

**Merge Behavior:**
- Environment variables override specific keys only
- Unset environment variables don't affect YAML
- Missing YAML uses all defaults
- Partial YAML merges with defaults

**Example:**
```yaml
# config.yaml
server:
  http:
    port: 8080
    read_timeout: 30s
```

```bash
# Override only read_timeout
export CONTEXTD_SERVER_HTTP_READ_TIMEOUT=60s

# Result: port=8080 (from YAML)
#         read_timeout=60s (from env)
```

### Type Conversion

Koanf automatically converts string environment variables:

**String:**
```bash
export CONTEXTD_SERVICE_NAME=myservice
```

**Integer:**
```bash
export CONTEXTD_DATABASE_QDRANT_PORT=6335
```

**Boolean:**
```bash
export CONTEXTD_FEATURES_MCP_MODE=true  # or: 1, yes
export CONTEXTD_OBSERVABILITY_TRACES_ENABLED=false  # or: 0, no
```

**Duration:**
```bash
export CONTEXTD_SERVER_HTTP_READ_TIMEOUT=60s
export CONTEXTD_DATABASE_BACKUP_INTERVAL=12h
```

**Size:**
```bash
export CONTEXTD_SERVER_REQUEST_MAX_BODY_SIZE=20MB
export CONTEXTD_FEATURES_INDEXING_MAX_FILE_SIZE=2MB
```

**Float:**
```bash
export CONTEXTD_OBSERVABILITY_TRACES_SAMPLE_RATE=0.1
export CONTEXTD_FEATURES_REMEDIATIONS_SEMANTIC_WEIGHT=0.8
```

---

## Hot Reload

### Reload Triggers

1. **SIGHUP Signal**
   ```bash
   killall -HUP contextd
   # or
   systemctl --user reload contextd
   ```

2. **File Watching** (automatic)
   - Watches config.yaml for changes
   - Debounced: 2-second delay after last write
   - Filters: Ignores temp/swap files

3. **API Endpoint**
   ```bash
   curl -X POST http://localhost:8080/admin/reload
   ```

### Reload Process

**Sequence:**
1. Receive reload signal/request
2. Acquire configuration write lock (blocks reads)
3. Read and parse YAML file
4. Load environment overrides
5. Validate new configuration
6. If valid: Apply atomically
7. If invalid: Keep current config, log error
8. Release lock
9. Log completion

**Timing:**
- File read: ~1-5ms
- Validation: ~10-50ms
- Application: ~5-10ms
- **Total: <100ms typical**

### Thread Safety (Koanf-Specific)

**RWMutex Protection Required:**

```go
type ConfigManager struct {
    mu sync.RWMutex
    k  *koanf.Koanf
}

// Read (concurrent safe)
func (c *ConfigManager) Get(key string) interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.k.Get(key)
}

// Reload (exclusive)
func (c *ConfigManager) Reload() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Load new config
    newK := koanf.New(".")
    if err := newK.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
        return err
    }

    // Validate
    if err := validate(newK); err != nil {
        return err
    }

    // Atomic swap
    c.k = newK
    return nil
}
```

### Reloadable vs Non-Reloadable

**Can Reload (No Restart):**
- ✅ Log level
- ✅ Timeouts
- ✅ Worker counts
- ✅ Cache settings
- ✅ Feature flags
- ✅ Rate limits
- ✅ Backup settings

**Cannot Reload (Requires Restart):**
- ❌ HTTP server port
- ❌ Database host/port
- ❌ Embedding provider
- ❌ Multi-tenant mode
- ❌ Token file path

### Error Handling

**Reload Failure:**
- Service continues running
- Keeps current configuration
- Logs ERROR with details
- Returns HTTP 500 on API reload

**Example:**
```
[ERROR] Config reload failed: yaml: line 15: did not find expected key
[INFO] Continuing with current configuration
[INFO] Fix config.yaml and reload again (no restart needed)
```

---

## Implementation Pattern

### Package Structure

```
pkg/config/
├── config.go        # Config structs and constants
├── loader.go        # Koanf-based loading
├── validator.go     # Validation logic
├── reload.go        # Hot reload implementation
├── security.go      # Security checks
├── migrate.go       # Migration tools
└── config_test.go   # Comprehensive tests
```

### Koanf Implementation

```go
package config

import (
    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/env"
    "github.com/knadh/koanf/providers/file"
    "github.com/knadh/koanf/v2"
)

type Manager struct {
    mu sync.RWMutex
    k  *koanf.Koanf
}

func New() (*Manager, error) {
    k := koanf.New(".")

    // Load YAML
    if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
        // Config file not found is OK (use defaults)
        if !os.IsNotExist(err) {
            return nil, err
        }
    }

    // Load env vars with CONTEXTD_ prefix
    k.Load(env.Provider("CONTEXTD_", ".", func(s string) string {
        return strings.Replace(strings.ToLower(
            strings.TrimPrefix(s, "CONTEXTD_")), "_", ".", -1)
    }), nil)

    return &Manager{k: k}, nil
}
```

### Key Implementation Points

1. **Security First**: Validate file permissions on startup
2. **Clear Errors**: All errors include context and fix suggestions
3. **Thread Safe**: Use RWMutex for concurrent access
4. **Backward Compatible**: Support env-only through v2.5
5. **Well Tested**: Achieve ≥80% coverage, 100% on critical paths

---

## Migration

### From Environment-Only to YAML+Env

**Current State (v1.x):**
```bash
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
export EMBEDDING_PROVIDER=tei
```

**Target State (v2.x):**
```yaml
# config.yaml
database:
  qdrant:
    host: localhost
    port: 6334

embedding:
  provider: tei
```

### Migration Phases

| Version | Date | Env-Only | YAML+Env | Action |
|---------|------|----------|----------|--------|
| v2.0 | 2025-Q1 | ✅ Supported | ✅ Supported | Both work |
| v2.2 | 2025-Q2 | ⚠️ Deprecated | ✅ Supported | Warnings logged |
| v2.5 | 2025-Q3 | ⚠️ Last release | ✅ Supported | Final warning |
| 0.9.0-rc-1 | 2026-Q1 | ❌ Removed | ✅ Required | YAML required |

### Migration Tools

**Generate config.yaml from environment:**
```bash
contextd config generate > ~/.config/contextd/config.yaml
contextd config validate
systemctl --user restart contextd
```

**Interactive migration:**
```bash
contextd config migrate
# Guides through: detect env vars → generate YAML → test → apply
```

**Validate configuration:**
```bash
contextd config validate [--config FILE]
```
