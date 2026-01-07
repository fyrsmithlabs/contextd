# Configuration Specification

**Feature**: Configuration Management
**Status**: Implemented
**Created**: 2025-11-23
**Updated**: 2026-01-06

---

## Overview

Unified configuration for contextd using Koanf with layered sources. Supports YAML config files, environment variable overrides, and custom validation.

| Goal | Description |
|------|-------------|
| **Layered** | Defaults -> YAML file -> env vars (later overrides earlier) |
| **Type-safe** | Unmarshal to Go structs with validation |
| **12-Factor** | Environment variables for production deployment |
| **Fail-fast** | Validate on startup, reject invalid config |
| **No secrets in files** | Secrets via env vars only |
| **Secure** | File permissions validated, path traversal protection |

### Non-Goals (MVP)

- Remote configuration (Consul, Vault) - Phase 2
- Live reload in production - restart for config changes
- GUI configuration
- CLI flag overrides - use env vars or YAML

---

## Quick Reference

| Component | Technology |
|-----------|------------|
| Library | Koanf v2 |
| Format | YAML |
| Validation | Custom validation functions |
| Secret handling | Custom `Secret` type (auto-redacted) |

### Config File Locations

**Supported paths** (validated for security):
1. `~/.config/contextd/config.yaml` (default)
2. `/etc/contextd/config.yaml` (system-wide)

**Security constraints**:
- Only paths within allowed directories accepted
- File permissions must be 0600 or 0400 (owner read/write only)
- Max file size: 1MB
- Symlinks resolved to prevent escape attacks

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| Koanf over Viper | 3x smaller binary, modular, correct key handling |
| YAML only | Simplicity, readable, widely supported |
| No env prefix | Direct env var names (e.g., `SERVER_PORT`, not `CONTEXTD_SERVER_PORT`) |
| Custom validation | Simpler than validator library, tailored to needs |
| Restricted paths | Security - prevent arbitrary file reads |

---

## Package Structure

```
internal/config/
├── config.go               # Root Config struct, Load(), Validate()
├── loader.go               # LoadWithFile(), path validation, Koanf setup
├── types.go                # Duration, Secret types with marshaling
├── config_test.go          # Config loading tests
├── loader_test.go          # File loading tests
├── production_test.go      # Production mode validation tests
├── env_validation_test.go  # Environment variable validation tests
└── path_validation_test.go # Path security tests
```

**Note**: All configuration sections (Server, Qdrant, Embeddings, etc.) are defined in `config.go` rather than separate files for simplicity.

---

## Architecture

@./architecture.md

---

## Detailed Documentation

| Document | Contents |
|----------|----------|
| @./schema.md | Full YAML configuration reference |
| @./structs.md | Go struct definitions |
| @./validation.md | Validation rules, custom validators |
| @./environment.md | Environment variable mapping |
| @./testing.md | Test helpers |

---

## Configuration Sections

| Section | Purpose |
|---------|---------|
| `production` | Production mode, security settings |
| `server` | HTTP server port and shutdown timeout |
| `observability` | OpenTelemetry configuration |
| `prefetch` | Pre-fetch engine rules and caching |
| `checkpoint` | Checkpoint size limits |
| `vectorstore` | Vector database provider selection (chromem/qdrant) |
| `qdrant` | Qdrant-specific configuration |
| `embeddings` | Embeddings provider (fastembed/tei) |
| `repository` | Repository indexing patterns |
| `statusline` | Claude Code statusline display |

---

## Configuration Loading

Two methods are available for loading configuration:

### 1. `config.Load()` - Environment-only (Recommended for Production)

```go
cfg := config.Load()
```

- Loads from environment variables only
- Uses hardcoded defaults for missing values
- Fast, simple, no file I/O
- Best for containerized deployments (12-factor app)

### 2. `config.LoadWithFile(path)` - YAML + Environment

```go
cfg, err := config.LoadWithFile("")  // Use default path
// or
cfg, err := config.LoadWithFile("/path/to/config.yaml")
```

- Loads from YAML file first, then overrides with environment variables
- Default path: `~/.config/contextd/config.yaml`
- File must be in allowed directories: `~/.config/contextd/` or `/etc/contextd/`
- File permissions must be 0600 or 0400
- Best for development and configuration-heavy deployments

**Precedence** (highest to lowest):
1. Environment variables
2. YAML file values
3. Hardcoded defaults

---

## Environment Variables

**Production & Security:**
- `CONTEXTD_PRODUCTION_MODE` - Enable production mode (default: `false`)
- `CONTEXTD_LOCAL_MODE` - Allow development features in production (default: `false`)
- `CONTEXTD_REQUIRE_AUTH` - Require authentication (default: `false`)
- `CONTEXTD_REQUIRE_TLS` - Require TLS for external services (default: `false`)
- `CONTEXTD_ALLOW_NO_ISOLATION` - Allow NoIsolation mode (default: `false`, always false in production)

**Server:**
- `SERVER_PORT` - HTTP server port (default: `9090`)
- `SERVER_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout (default: `10s`)

**Observability:**
- `OTEL_ENABLE` - Enable OpenTelemetry (default: `false`)
- `OTEL_SERVICE_NAME` - Service name for traces (default: `contextd`)

**Vector Store:**
- `CONTEXTD_VECTORSTORE_PROVIDER` - Provider: `chromem` or `qdrant` (default: `chromem`)
- `CONTEXTD_VECTORSTORE_CHROMEM_PATH` - Chromem storage path (default: `~/.config/contextd/vectorstore`)
- `CONTEXTD_VECTORSTORE_CHROMEM_COMPRESS` - Enable compression (default: `false`)
- `CONTEXTD_VECTORSTORE_CHROMEM_COLLECTION` - Collection name (default: `contextd_default`)
- `CONTEXTD_VECTORSTORE_CHROMEM_VECTOR_SIZE` - Embedding dimensions (default: `384`)

**Qdrant:**
- `QDRANT_HOST` - Qdrant host (default: `localhost`)
- `QDRANT_PORT` - Qdrant gRPC port (default: `6334`)
- `QDRANT_HTTP_PORT` - Qdrant HTTP port (default: `6333`)
- `QDRANT_COLLECTION` - Collection name (default: `contextd_default`)
- `QDRANT_VECTOR_SIZE` - Vector dimensions (default: `384`)
- `CONTEXTD_DATA_PATH` - Base data path (default: `/data`)

**Embeddings:**
- `EMBEDDINGS_PROVIDER` - Provider: `fastembed` or `tei` (default: `fastembed`)
- `EMBEDDINGS_MODEL` - Model name (default: `BAAI/bge-small-en-v1.5`)
- `EMBEDDING_BASE_URL` - TEI endpoint (default: `http://localhost:8080`)
- `EMBEDDINGS_CACHE_DIR` - Model cache directory (default: `./local_cache`)
- `EMBEDDINGS_ONNX_VERSION` - ONNX runtime version override (optional)

**Repository:**
- `REPOSITORY_IGNORE_FILES` - Comma-separated ignore file names (default: `.gitignore,.dockerignore,.contextdignore`)
- `REPOSITORY_FALLBACK_EXCLUDES` - Comma-separated exclude patterns (default: `.git/**,node_modules/**,vendor/**,__pycache__/**`)

**Checkpoint:**
- `CHECKPOINT_MAX_CONTENT_SIZE_KB` - Max checkpoint size in KB (default: `1024`)

**Pre-fetch:**
- `PREFETCH_ENABLED` - Enable pre-fetch engine (default: `true`)
- `PREFETCH_CACHE_TTL` - Cache TTL (default: `5m`)
- `PREFETCH_CACHE_MAX_ENTRIES` - Max cache entries (default: `100`)
- `PREFETCH_BRANCH_DIFF_ENABLED` - Enable branch diff rule (default: `true`)
- `PREFETCH_BRANCH_DIFF_MAX_FILES` - Max files for branch diff (default: `10`)
- `PREFETCH_BRANCH_DIFF_MAX_SIZE_KB` - Max size per file (default: `50`)
- `PREFETCH_BRANCH_DIFF_TIMEOUT_MS` - Timeout in ms (default: `1000`)
- `PREFETCH_RECENT_COMMIT_ENABLED` - Enable recent commit rule (default: `true`)
- `PREFETCH_RECENT_COMMIT_MAX_SIZE_KB` - Max commit message size (default: `20`)
- `PREFETCH_RECENT_COMMIT_TIMEOUT_MS` - Timeout in ms (default: `500`)
- `PREFETCH_COMMON_FILES_ENABLED` - Enable common files rule (default: `true`)
- `PREFETCH_COMMON_FILES_MAX_FILES` - Max common files (default: `3`)
- `PREFETCH_COMMON_FILES_TIMEOUT_MS` - Timeout in ms (default: `500`)

**Statusline:**
- `CONTEXTD_STATUSLINE_ENABLED` - Enable statusline (default: `true`)
- `CONTEXTD_STATUSLINE_ENDPOINT` - HTTP endpoint (default: `http://localhost:9090`)
- `CONTEXTD_STATUSLINE_SHOW_SERVICE` - Show service status (default: `true`)
- `CONTEXTD_STATUSLINE_SHOW_MEMORIES` - Show memory count (default: `true`)
- `CONTEXTD_STATUSLINE_SHOW_CHECKPOINTS` - Show checkpoint count (default: `true`)
- `CONTEXTD_STATUSLINE_SHOW_CONTEXT` - Show context usage (default: `true`)
- `CONTEXTD_STATUSLINE_SHOW_CONFIDENCE` - Show confidence (default: `true`)
- `CONTEXTD_STATUSLINE_SHOW_COMPRESSION` - Show compression ratio (default: `true`)
- `CONTEXTD_STATUSLINE_CONTEXT_WARNING` - Warning threshold % (default: `70`)
- `CONTEXTD_STATUSLINE_CONTEXT_CRITICAL` - Critical threshold % (default: `85`)

---

## Functional Requirements

| ID | Requirement | Status |
|----|-------------|--------|
| FR-001 | Load from defaults -> file -> env (no CLI flags) | ✅ Implemented |
| FR-002 | YAML format for config files | ✅ Implemented |
| FR-003 | All keys settable via env vars (mixed prefixes) | ✅ Implemented |
| FR-004 | Unmarshal to strongly-typed Go structs | ✅ Implemented |
| FR-005 | Validate on load, reject invalid values | ✅ Implemented |
| FR-006 | Redact secrets in logs/serialization | ✅ Implemented (Secret type) |
| FR-007 | Sensible defaults for development | ✅ Implemented |
| FR-008 | Cross-field validation (production mode) | ✅ Implemented |
| FR-009 | Clear validation error messages | ✅ Implemented |
| FR-010 | Secure file path validation | ✅ Implemented |
| FR-011 | File permission validation (0600/0400) | ✅ Implemented (Unix only) |
| FR-012 | Path traversal protection | ✅ Implemented |
| FR-013 | Environment variable input validation | ✅ Implemented |

---

## Success Criteria

| ID | Criterion |
|----|-----------|
| SC-001 | Config loading < 100ms |
| SC-002 | 0 secret values in log output |
| SC-003 | All fields have appropriate validation |
| SC-004 | > 80% test coverage |

---

## Implementation Status

| Phase | Status | Contents |
|-------|--------|----------|
| **1** | ✅ Complete | Koanf setup, YAML loading, struct definitions |
| **2** | ✅ Complete | Environment variables, Secret type |
| **3** | ⏭️ Skipped | CLI flags (not needed - use env vars or YAML) |
| **4** | ✅ Complete | Validation, custom validators |
| **5** | ✅ Complete | Test helpers, comprehensive test coverage |

**Current Status**: Fully implemented and in production use.

---

## Security Features

The configuration system includes multiple security layers:

### File Security

| Feature | Implementation |
|---------|----------------|
| **Path validation** | Only `~/.config/contextd/` and `/etc/contextd/` allowed |
| **Symlink resolution** | Symlinks resolved to prevent directory escape |
| **File permissions** | Must be 0600 or 0400 (owner-only read/write) |
| **Size limit** | Max 1MB to prevent resource exhaustion |
| **TOCTOU protection** | File opened once, validated via descriptor |

### Input Validation

| Feature | Implementation |
|---------|----------------|
| **Hostname validation** | Reject shell metacharacters (`;`, `$`, `` ` ``, `|`, etc.) |
| **Path traversal** | Reject `..` sequences, validate cleaned paths |
| **URL validation** | Only `http://` and `https://` schemes allowed |
| **Port range** | Server ports must be 1-65535 |

### Production Mode

Production mode enforces additional security constraints:

```go
if production.Enabled {
    // NoIsolation mode always blocked
    AllowNoIsolation = false

    // Authentication required unless local override
    RequireAuthentication = !LocalModeAcknowledged

    // TLS required unless local override
    RequireTLS = !LocalModeAcknowledged
}
```

**Environment variables:**
- `CONTEXTD_PRODUCTION_MODE=1` - Enable production mode
- `CONTEXTD_LOCAL_MODE=1` - Override for local development (use cautiously)

### Secret Handling

The `Secret` type ensures secrets are never logged or serialized:

```go
type Secret string

func (s Secret) String() string { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal("[REDACTED]") }
func (s Secret) Value() string { return string(s) } // Only use when needed
```

**Usage:**
```yaml
qdrant:
  api_key: ""  # Don't put secrets in files!
```

Set via environment instead:
```bash
export QDRANT_API_KEY="secret_value_here"
```

---

## Future Capabilities (Post-MVP)

Koanf natively supports for future integration:

| Provider | Use Case | Priority |
|----------|----------|----------|
| Consul | Non-secret config, service discovery | P2 |
| Vault | Secrets management, rotation | P2 |
| etcd | Distributed config | P3 |
| AWS Parameter Store | Cloud-native secrets | P3 |

---

## References

- [Koanf Documentation](https://github.com/knadh/koanf)
- [go-playground/validator](https://github.com/go-playground/validator)
- [12-Factor App Config](https://12factor.net/config)
