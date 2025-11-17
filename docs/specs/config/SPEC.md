# Configuration Management Specification

**Version**: 1.0.0
**Status**: Draft
**Date**: 2025-11-04
**Library**: Koanf v2

## Overview

### Purpose

The contextd configuration management system provides a secure, flexible, and maintainable approach to service configuration. It combines YAML-based configuration files with environment variable overrides and hot reload capabilities.

### Design Goals

1. **Security-First**: No sensitive values in config files, enforced file permissions
2. **Local-First**: Fast loading, minimal dependencies, file-based storage
3. **Developer-Friendly**: Clear structure, comprehensive validation, helpful errors
4. **Production-Ready**: Hot reload, backward compatibility, robust error handling
5. **Context-Optimized**: Minimal bloat, clear documentation, efficient parsing

### Key Features

- **YAML Configuration**: Human-readable structured config with comments
- **Environment Overrides**: Full environment variable support (`CONTEXTD_*` prefix)
- **Hot Reload**: Runtime updates via SIGHUP, file watching, or API
- **Strong Validation**: Comprehensive validation with clear error messages
- **Migration Support**: Smooth transition from env-only to YAML+env
- **Thread-Safe**: Concurrent access protection using RWMutex
- **Backward Compatible**: Supports env-only through v2.5

### Library Choice: Koanf

Based on comprehensive research (see [research-findings.md](research-findings.md)), **Koanf** was selected over Viper:

- **313% smaller binary** (~2MB vs ~6-8MB)
- **YAML spec compliant** (preserves case, doesn't lowercase keys)
- **Modular dependencies** (install only what you need)
- **Clean API** (explicit, predictable, less magic)
- **Production-proven** (Listmonk 15k+ stars, imgproxy, gowitness)

**Trade-offs accepted:**
- Thread safety requires explicit mutex (we need this anyway for atomic updates)
- Smaller community (mitigated: mature v2, well-documented, actively maintained)

---

## Configuration File Structure

### File Location

**Primary**: `~/.config/contextd/config.yaml` (0600 permissions)

**Search Order**:
1. Path specified via `--config` flag
2. `CONTEXTD_CONFIG_FILE` environment variable
3. `./config.yaml` (current directory)
4. `~/.config/contextd/config.yaml` (user config)
5. `/etc/contextd/config.yaml` (system config)

**Security Files** (separate from config.yaml):
- Token: `~/.config/contextd/token` (0600)
- OpenAI API Key: `~/.config/contextd/openai_api_key` (0600)

### Complete YAML Schema

```yaml
# contextd Configuration
# Location: ~/.config/contextd/config.yaml
# Permissions: 0600 (required)
#
# All settings can be overridden with environment variables:
# Format: CONTEXTD_SECTION_SUBSECTION_KEY
# Example: CONTEXTD_SERVER_SOCKET_PATH

# Service identification
service:
  name: contextd
  version: 2.0.0
  environment: development  # development, staging, production
  log_level: info           # debug, info, warn, error

# Server configuration
server:
  socket:
    path: ~/.config/contextd/api.sock
    permissions: 0600
    cleanup_on_start: true

  http:
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

## Configuration Options

### Service Options

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `service.name` | string | `contextd` | `CONTEXTD_SERVICE_NAME` | any | Service identifier |
| `service.version` | string | `2.0.0` | `CONTEXTD_SERVICE_VERSION` | semver | Service version |
| `service.environment` | string | `development` | `CONTEXTD_SERVICE_ENVIRONMENT` | development, staging, production | Deployment environment |
| `service.log_level` | string | `info` | `CONTEXTD_SERVICE_LOG_LEVEL` | debug, info, warn, error | Logging level |

**Examples:**
```bash
export CONTEXTD_SERVICE_ENVIRONMENT=production
export CONTEXTD_SERVICE_LOG_LEVEL=warn
```

### Server Options

| Option | Type | Default | Env Variable | Range/Format | Description |
|--------|------|---------|--------------|--------------|-------------|
| `server.socket.path` | string | `~/.config/contextd/api.sock` | `CONTEXTD_SERVER_SOCKET_PATH` | absolute path | Unix socket location |
| `server.socket.permissions` | octal | `0600` | `CONTEXTD_SERVER_SOCKET_PERMISSIONS` | `0600` | Socket permissions (REQUIRED) |
| `server.socket.cleanup_on_start` | bool | `true` | `CONTEXTD_SERVER_SOCKET_CLEANUP_ON_START` | true/false | Remove stale socket |
| `server.http.read_timeout` | duration | `30s` | `CONTEXTD_SERVER_HTTP_READ_TIMEOUT` | 1s-300s | Read timeout |
| `server.http.write_timeout` | duration | `30s` | `CONTEXTD_SERVER_HTTP_WRITE_TIMEOUT` | 1s-300s | Write timeout |
| `server.http.idle_timeout` | duration | `120s` | `CONTEXTD_SERVER_HTTP_IDLE_TIMEOUT` | 1s-600s | Keep-alive timeout |
| `server.http.shutdown_timeout` | duration | `10s` | `CONTEXTD_SERVER_HTTP_SHUTDOWN_TIMEOUT` | 1s-60s | Graceful shutdown |
| `server.request.max_body_size` | size | `10MB` | `CONTEXTD_SERVER_REQUEST_MAX_BODY_SIZE` | 1KB-100MB | Max request body |
| `server.request.max_header_size` | size | `1MB` | `CONTEXTD_SERVER_REQUEST_MAX_HEADER_SIZE` | 1KB-10MB | Max header size |
| `server.security.token_file` | string | `~/.config/contextd/token` | `CONTEXTD_SERVER_SECURITY_TOKEN_FILE` | absolute path | Bearer token file |
| `server.security.require_token` | bool | `true` | `CONTEXTD_SERVER_SECURITY_REQUIRE_TOKEN` | true/false | Enforce authentication |
| `server.security.token_min_length` | int | `32` | `CONTEXTD_SERVER_SECURITY_TOKEN_MIN_LENGTH` | 16-128 | Min token bytes |

**Examples:**
```bash
export CONTEXTD_SERVER_SOCKET_PATH=/tmp/contextd.sock
export CONTEXTD_SERVER_HTTP_READ_TIMEOUT=60s
```

### Database Options

| Option | Type | Default | Env Variable | Range/Format | Description |
|--------|------|---------|--------------|--------------|-------------|
| `database.qdrant.host` | string | `localhost` | `CONTEXTD_DATABASE_QDRANT_HOST` | hostname/IP | Qdrant host |
| `database.qdrant.port` | int | `6334` | `CONTEXTD_DATABASE_QDRANT_PORT` | 1-65535 | Qdrant gRPC port |
| `database.qdrant.api_key_file` | string | `""` | `CONTEXTD_DATABASE_QDRANT_API_KEY_FILE` | file path | API key file (0600) |
| `database.qdrant.use_tls` | bool | `false` | `CONTEXTD_DATABASE_QDRANT_USE_TLS` | true/false | Enable TLS |
| `database.qdrant.timeout` | duration | `30s` | `CONTEXTD_DATABASE_QDRANT_TIMEOUT` | 5s-300s | Connection timeout |
| `database.pool.max_connections` | int | `10` | `CONTEXTD_DATABASE_POOL_MAX_CONNECTIONS` | 1-100 | Max connections |
| `database.pool.max_idle_connections` | int | `5` | `CONTEXTD_DATABASE_POOL_MAX_IDLE_CONNECTIONS` | 1-max_connections | Max idle |
| `database.pool.connection_timeout` | duration | `5s` | `CONTEXTD_DATABASE_POOL_CONNECTION_TIMEOUT` | 1s-60s | Connection timeout |
| `database.multi_tenant.enabled` | bool | `true` | `CONTEXTD_DATABASE_MULTI_TENANT_ENABLED` | `true` only | Multi-tenant mode (v2.0+) |
| `database.multi_tenant.project_hash_algo` | string | `sha256` | `CONTEXTD_DATABASE_MULTI_TENANT_PROJECT_HASH_ALGO` | sha256, sha512 | Hash algorithm |
| `database.backup.enabled` | bool | `true` | `CONTEXTD_DATABASE_BACKUP_ENABLED` | true/false | Enable backups |
| `database.backup.interval` | duration | `24h` | `CONTEXTD_DATABASE_BACKUP_INTERVAL` | 1h-168h | Backup frequency |
| `database.backup.retention_days` | int | `30` | `CONTEXTD_DATABASE_BACKUP_RETENTION_DAYS` | 1-365 | Retention period |
| `database.backup.path` | string | `~/.local/share/contextd/backups` | `CONTEXTD_DATABASE_BACKUP_PATH` | directory path | Backup location |
| `database.backup.compress` | bool | `true` | `CONTEXTD_DATABASE_BACKUP_COMPRESS` | true/false | Compress backups |

**Examples:**
```bash
export CONTEXTD_DATABASE_QDRANT_HOST=qdrant.example.com
export CONTEXTD_DATABASE_QDRANT_USE_TLS=true
export CONTEXTD_DATABASE_POOL_MAX_CONNECTIONS=20
```

### Embedding Options

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `embedding.provider` | string | `tei` | `CONTEXTD_EMBEDDING_PROVIDER` | tei, openai | Embedding provider |
| `embedding.tei.base_url` | string | `http://localhost:8080` | `CONTEXTD_EMBEDDING_TEI_BASE_URL` | URL | TEI service URL |
| `embedding.tei.model` | string | `BAAI/bge-large-en-v1.5` | `CONTEXTD_EMBEDDING_TEI_MODEL` | model name | Embedding model |
| `embedding.tei.timeout` | duration | `30s` | `CONTEXTD_EMBEDDING_TEI_TIMEOUT` | 5s-300s | Request timeout |
| `embedding.tei.retry_attempts` | int | `3` | `CONTEXTD_EMBEDDING_TEI_RETRY_ATTEMPTS` | 0-10 | Retry count |
| `embedding.tei.retry_delay` | duration | `1s` | `CONTEXTD_EMBEDDING_TEI_RETRY_DELAY` | 100ms-10s | Retry delay |
| `embedding.openai.api_key_file` | string | `~/.config/contextd/openai_api_key` | `CONTEXTD_EMBEDDING_OPENAI_API_KEY_FILE` | file path | API key file (0600) |
| `embedding.openai.model` | string | `text-embedding-3-small` | `CONTEXTD_EMBEDDING_OPENAI_MODEL` | model name | OpenAI model |
| `embedding.openai.timeout` | duration | `30s` | `CONTEXTD_EMBEDDING_OPENAI_TIMEOUT` | 5s-300s | Request timeout |
| `embedding.openai.max_retries` | int | `3` | `CONTEXTD_EMBEDDING_OPENAI_MAX_RETRIES` | 0-10 | Max retries |
| `embedding.cache.enabled` | bool | `true` | `CONTEXTD_EMBEDDING_CACHE_ENABLED` | true/false | Enable cache |
| `embedding.cache.ttl` | duration | `24h` | `CONTEXTD_EMBEDDING_CACHE_TTL` | 1h-168h | Cache TTL |
| `embedding.cache.max_entries` | int | `10000` | `CONTEXTD_EMBEDDING_CACHE_MAX_ENTRIES` | 100-1000000 | Max cache entries |

**Security Requirements:**
- `openai.api_key_file` MUST have 0600 permissions
- API key MUST be in separate file, NEVER inline

**Examples:**
```bash
export CONTEXTD_EMBEDDING_PROVIDER=openai
export CONTEXTD_EMBEDDING_OPENAI_MODEL=text-embedding-3-large
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
server.socket.path            CONTEXTD_SERVER_SOCKET_PATH
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
  socket:
    path: /custom/path.sock
  http:
    read_timeout: 30s
```

```bash
# Override only read_timeout
export CONTEXTD_SERVER_HTTP_READ_TIMEOUT=60s

# Result: path=/custom/path.sock (from YAML)
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
   curl --unix-socket ~/.config/contextd/api.sock \
        -H "Authorization: Bearer $TOKEN" \
        -X POST http://localhost/admin/reload
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
- ❌ Socket path
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

## Validation

### Required Fields

**Must be present** (or have defaults):
- `service.name`
- `server.socket.path`
- `server.security.token_file`
- `database.qdrant.host`
- `database.qdrant.port`
- `embedding.provider`

### Validation Rules

**Type Validation:**
- Strings: non-empty for required fields
- Integers: within valid ranges
- Booleans: true/false only
- Durations: valid Go duration format
- Sizes: valid size format (KB, MB, GB)

**Range Validation:**
- Ports: 1-65535
- Timeouts: 1s minimum
- Sample rates: 0.0-1.0
- Retention days: 1-365

**Format Validation:**
- URLs: valid http/https
- File paths: absolute paths
- Permissions: octal 0600 only
- Environments: development, staging, production

**Logical Validation:**
- `max_idle_connections` ≤ `max_connections`
- `semantic_weight` + `string_weight` = 1.0
- `multi_tenant.enabled` must be `true` (v2.0+)

**Security Validation:**
- No API keys inline in config
- Token file has 0600 permissions
- API key files have 0600 permissions
- Socket has 0600 permissions

### Error Messages

**Format:**
```
Configuration validation failed with N error(s):
  1. server.socket.permissions must be 0600 (got: 0644)
     Fix with: chmod 0600 ~/.config/contextd/api.sock

  2. database.qdrant.port must be 1-65535 (got: 70000)

  3. embedding.openai.api_key_file does not exist: /secure/openai_key
```

**Characteristics:**
- Clear, actionable
- Include actual vs expected values
- Provide fix commands when possible
- List ALL errors, not just first
- Include context (file, line, field)

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

---

## Security

### Critical Rules

**🔒 RULE #1: NO SENSITIVE VALUES IN CONFIG FILES**

Sensitive values include:
- API keys
- Bearer tokens
- Database passwords
- TLS certificates/keys

**✅ CORRECT:**
```yaml
server:
  security:
    token_file: ~/.config/contextd/token  # ✅ Reference file

embedding:
  openai:
    api_key_file: ~/.config/contextd/openai_api_key  # ✅ Reference file
```

**❌ INCORRECT:**
```yaml
server:
  security:
    token: "Bearer abc123..."  # ❌ NEVER inline

embedding:
  openai:
    api_key: "sk-..."  # ❌ NEVER inline
```

**🔒 RULE #2: ENFORCE FILE PERMISSIONS**

All contextd files MUST have 0600 permissions:

```bash
chmod 0600 ~/.config/contextd/config.yaml
chmod 0600 ~/.config/contextd/token
chmod 0600 ~/.config/contextd/openai_api_key
chmod 0600 ~/.config/contextd/api.sock
```

**Verification:**
```bash
ls -la ~/.config/contextd/
# -rw------- 1 user user  config.yaml
# -rw------- 1 user user  token
# -rw------- 1 user user  openai_api_key
# srw------- 1 user user  api.sock
```

### Security Checklist

**Development:**
- [ ] No API keys in code
- [ ] No tokens in tests
- [ ] No secrets in error messages
- [ ] File permissions validated
- [ ] Security tests passing

**Deployment:**
- [ ] config.yaml has 0600
- [ ] token file has 0600
- [ ] api_key files have 0600
- [ ] socket has 0600
- [ ] No secrets in environment

**Operations:**
- [ ] Monitor file permissions
- [ ] Rotate credentials quarterly
- [ ] Audit access logs
- [ ] Review security alerts

---

## Testing

### Unit Tests

**Coverage Requirements:**
- Minimum: 80% overall
- Config package: 100%
- Validation: 100%
- Security checks: 100%

**Test Categories:**

1. **Parsing Tests** (~20 test cases)
   - Valid YAML
   - Invalid YAML
   - Environment overrides
   - Missing required fields
   - Type conversions

2. **Validation Tests** (~30 test cases)
   - Required field validation
   - Type validation
   - Range validation
   - Format validation
   - Security validation

3. **Override Tests** (~15 test cases)
   - String override
   - Integer override
   - Boolean override
   - Duration override
   - Nested override

### Integration Tests

**Scenarios:**

1. **Configuration Flow**
   - Create config.yaml
   - Set environment overrides
   - Load configuration
   - Verify YAML values
   - Verify environment overrides

2. **Service Startup**
   - Create minimal config
   - Create token file
   - Start service
   - Test health endpoint
   - Graceful shutdown

3. **Hot Reload**
   - Start with initial config
   - Verify initial values
   - Update config file
   - Trigger reload
   - Verify new values
   - Service still running

4. **Concurrent Access**
   - Start concurrent readers
   - Trigger reload during reads
   - Verify no nil configs
   - Verify no errors

5. **Rollback on Error**
   - Get initial config
   - Update with invalid config
   - Attempt reload (should fail)
   - Verify config unchanged
   - Service still functional

---

## Implementation Guide

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

### Koanf Implementation Pattern

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

## References

- **Research**: [research-findings.md](research-findings.md)
- **Koanf**: https://github.com/knadh/koanf
- **YAML Spec**: https://yaml.org/spec/1.2/spec.html
- **Go Validator**: https://github.com/go-playground/validator

---

**Status**: Ready for implementation with golang-pro skill

**Next Steps**:
1. Review specification
2. Create implementation tasks breakdown
3. Implement using TDD (tests first)
4. Validate with qa-engineer persona

**Version History**:
- v1.0.0 (2025-11-04): Initial specification based on Koanf research
