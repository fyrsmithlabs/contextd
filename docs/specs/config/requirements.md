# Configuration Requirements

**Parent**: [../SPEC.md](../SPEC.md)

This document defines the functional and non-functional requirements for the contextd configuration management system.

---

## Design Goals

1. **Security-First**: No sensitive values in config files, enforced file permissions
2. **Local-First**: Fast loading, minimal dependencies, file-based storage
3. **Developer-Friendly**: Clear structure, comprehensive validation, helpful errors
4. **Production-Ready**: Hot reload, backward compatibility, robust error handling
5. **Context-Optimized**: Minimal bloat, clear documentation, efficient parsing

## Key Features

- **YAML Configuration**: Human-readable structured config with comments
- **Environment Overrides**: Full environment variable support (`CONTEXTD_*` prefix)
- **Hot Reload**: Runtime updates via SIGHUP, file watching, or API
- **Strong Validation**: Comprehensive validation with clear error messages
- **Migration Support**: Smooth transition from env-only to YAML+env
- **Thread-Safe**: Concurrent access protection using RWMutex
- **Backward Compatible**: Supports env-only through v2.5

## Library Selection: Koanf

Based on comprehensive research (see [research-findings.md](research-findings.md)), **Koanf** was selected over Viper:

**Why Koanf:**
- **313% smaller binary** (~2MB vs ~6-8MB)
- **YAML spec compliant** (preserves case, doesn't lowercase keys)
- **Modular dependencies** (install only what you need)
- **Clean API** (explicit, predictable, less magic)
- **Production-proven** (Listmonk 15k+ stars, imgproxy, gowitness)

**Trade-offs accepted:**
- Thread safety requires explicit mutex (we need this anyway for atomic updates)
- Smaller community (mitigated: mature v2, well-documented, actively maintained)

---

## File Requirements

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

---

## Required Fields

**Must be present** (or have defaults):
- `service.name`
- `server.http.port`
- `server.security.token_file`
- `database.qdrant.host`
- `database.qdrant.port`
- `embedding.provider`

---

## Validation Requirements

### Type Validation
- Strings: non-empty for required fields
- Integers: within valid ranges
- Booleans: true/false only
- Durations: valid Go duration format
- Sizes: valid size format (KB, MB, GB)

### Range Validation
- Ports: 1-65535
- Timeouts: 1s minimum
- Sample rates: 0.0-1.0
- Retention days: 1-365

### Format Validation
- URLs: valid http/https
- File paths: absolute paths
- Permissions: octal 0600 only
- Environments: development, staging, production

### Logical Validation
- `max_idle_connections` ≤ `max_connections`
- `semantic_weight` + `string_weight` = 1.0
- `multi_tenant.enabled` must be `true` (v2.0+)

### Security Validation
- No API keys inline in config
- Token file has 0600 permissions
- API key files have 0600 permissions
- Config file has 0600 permissions

---

## Error Message Requirements

**Format:**
```
Configuration validation failed with N error(s):
  1. server.http.port must be 1-65535 (got: 70000)
     Fix with: Set CONTEXTD_HTTP_PORT to valid port (1-65535)

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

## Security Requirements

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
```

**Verification:**
```bash
ls -la ~/.config/contextd/
# -rw------- 1 user user  config.yaml
# -rw------- 1 user user  token
# -rw------- 1 user user  openai_api_key
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
- [ ] No secrets in environment

**Operations:**
- [ ] Monitor file permissions
- [ ] Rotate credentials quarterly
- [ ] Audit access logs
- [ ] Review security alerts

---

## Testing Requirements

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
