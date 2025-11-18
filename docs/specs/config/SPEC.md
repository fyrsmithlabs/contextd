# Configuration Management Specification

**Version**: 1.0.0
**Status**: Draft
**Last Updated**: 2025-11-04

---

## Overview

The contextd configuration management system provides a secure, flexible, and maintainable approach to service configuration. It combines YAML-based configuration files with environment variable overrides and hot reload capabilities.

### Purpose

Manage contextd service configuration through:
- YAML-based configuration files with comments
- Environment variable overrides (`CONTEXTD_*` prefix)
- Hot reload without service restart
- Strong validation with clear error messages

### Design Goals

1. **Security-First**: No sensitive values in config files, enforced file permissions
2. **Local-First**: Fast loading, minimal dependencies, file-based storage
3. **Developer-Friendly**: Clear structure, comprehensive validation, helpful errors
4. **Production-Ready**: Hot reload, backward compatibility, robust error handling
5. **Context-Optimized**: Minimal bloat, clear documentation, efficient parsing

---

## Quick Reference

**Key Facts**:
- Technology: Koanf v2 configuration library
- Location: `~/.config/contextd/config.yaml` (0600 permissions)
- Format: YAML with environment variable overrides
- Hot Reload: SIGHUP, file watching, or API endpoint
- Migration: Env-only supported through v2.5, YAML required in 0.9.0-rc-1

**Library Choice: Koanf**

Selected over Viper for:
- **313% smaller binary** (~2MB vs ~6-8MB)
- **YAML spec compliant** (preserves case, doesn't lowercase keys)
- **Modular dependencies** (install only what you need)
- **Clean API** (explicit, predictable, less magic)
- **Production-proven** (Listmonk 15k+ stars, imgproxy, gowitness)

**Trade-offs**:
- Thread safety requires explicit mutex (we need this anyway for atomic updates)
- Smaller community (mitigated: mature v2, well-documented, actively maintained)

**Components**:
- Config loader (Koanf-based)
- Validator (type, range, format, security)
- Hot reload manager (SIGHUP, file watch, API)
- Migration tools (env-only → YAML+env)
- Security checks (file permissions, credential validation)

---

## Detailed Documentation

**Requirements & Design**:
@./config/requirements.md - Configuration requirements and validation rules
@./config/architecture.md - Environment variables, hot reload, implementation patterns

**Reference & Usage**:
@./config/reference.md - Complete configuration option reference tables
@./config/workflows.md - Configuration examples and usage patterns

---

## Configuration Overview

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

### Key Configuration Sections

**Service Configuration**:
- Service identification (name, version, environment)
- Log level (debug, info, warn, error)

**Server Configuration**:
- HTTP server (port, host, timeouts)
- Request limits (body size, header size)
- Security (token file, authentication)

**Database Configuration**:
- Qdrant connection (host, port, TLS)
- Connection pool (max connections, timeouts)
- Multi-tenant mode (required in v2.0+)
- Backup settings (interval, retention)

**Embedding Configuration**:
- Provider selection (TEI or OpenAI)
- Provider-specific settings
- Cache configuration

**Observability Configuration**:
- OpenTelemetry traces and metrics
- Logging format and output

**Feature Flags**:
- Checkpoints, remediations, skills, troubleshooting, indexing

**Performance Tuning**:
- Worker counts, cache settings, rate limiting

---

## Environment Variable Override

### Naming Convention

Pattern: `CONTEXTD_SECTION_SUBSECTION_KEY`

**Examples**:
```
YAML Path                     Environment Variable
─────────────────────────────────────────────────────
service.log_level             CONTEXTD_SERVICE_LOG_LEVEL
server.http.port              CONTEXTD_HTTP_PORT
database.qdrant.host          CONTEXTD_DATABASE_QDRANT_HOST
embedding.tei.base_url        CONTEXTD_EMBEDDING_TEI_BASE_URL
```

### Precedence

1. **Environment Variables** (highest priority)
2. **YAML Configuration File**
3. **Default Values** (lowest priority)

---

## Hot Reload

### Triggers

1. **SIGHUP Signal**: `killall -HUP contextd`
2. **File Watching**: Automatic on config.yaml changes
3. **API Endpoint**: `curl -X POST http://localhost:8080/admin/reload`

### Process

- Read and parse YAML file
- Load environment overrides
- Validate new configuration
- If valid: Apply atomically (exclusive lock, <100ms)
- If invalid: Keep current config, log error

### Reloadable Settings

**Can Reload** (no restart): Log level, timeouts, worker counts, cache settings, feature flags
**Cannot Reload** (restart required): HTTP port, database host/port, embedding provider, multi-tenant mode

---

## Security

### Critical Rules

**🔒 RULE #1: NO SENSITIVE VALUES IN CONFIG FILES**

Store API keys and tokens in separate files with 0600 permissions:
- `server.security.token_file` (not inline token)
- `embedding.openai.api_key_file` (not inline key)

**🔒 RULE #2: ENFORCE FILE PERMISSIONS**

All contextd files MUST have 0600 permissions:
```bash
chmod 0600 ~/.config/contextd/config.yaml
chmod 0600 ~/.config/contextd/token
chmod 0600 ~/.config/contextd/openai_api_key
```

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

### Testing Requirements

**Coverage**: ≥80% overall, 100% for config/validation/security

**Test Categories**:
- Parsing tests (~20 cases): YAML, env overrides, type conversions
- Validation tests (~30 cases): Required fields, ranges, formats, security
- Override tests (~15 cases): All data types, nested overrides
- Integration tests: Service startup, hot reload, concurrent access, rollback

---

## Migration

### Timeline

| Version | Date | Env-Only | YAML+Env | Action |
|---------|------|----------|----------|--------|
| v2.0 | 2025-Q1 | ✅ Supported | ✅ Supported | Both work |
| v2.2 | 2025-Q2 | ⚠️ Deprecated | ✅ Supported | Warnings logged |
| v2.5 | 2025-Q3 | ⚠️ Last release | ✅ Supported | Final warning |
| 0.9.0-rc-1 | 2026-Q1 | ❌ Removed | ✅ Required | YAML required |

### Migration Tools

```bash
# Generate config.yaml from environment
contextd config generate > ~/.config/contextd/config.yaml

# Interactive migration
contextd config migrate

# Validate configuration
contextd config validate
```

---

## Summary

The configuration management system provides secure, flexible, and maintainable service configuration through YAML files with environment overrides. Key features include hot reload, comprehensive validation, and smooth migration from environment-only configuration.

**Current Status**: Draft specification ready for implementation

**Next Steps**:
1. Review specification with team
2. Implement using golang-pro skill (TDD approach)
3. Achieve ≥80% test coverage
4. Validate with qa-engineer persona
5. Create migration tools
6. Update documentation

---

## References

- **Research**: [research-findings.md](research-findings.md)
- **Koanf Documentation**: https://github.com/knadh/koanf
- **YAML Specification**: https://yaml.org/spec/1.2/spec.html
