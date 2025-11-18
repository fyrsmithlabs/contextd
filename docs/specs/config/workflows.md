# Configuration Workflows

**Parent**: [../SPEC.md](../SPEC.md)

This document provides example workflows and usage patterns for contextd configuration management.

---

## Basic Configuration

### Minimal Configuration

**Simplest working config:**

```yaml
# ~/.config/contextd/config.yaml
service:
  name: contextd

server:
  http:
    port: 8080

database:
  qdrant:
    host: localhost
    port: 6334

embedding:
  provider: tei
```

**Start contextd:**
```bash
chmod 0600 ~/.config/contextd/config.yaml
contextd
```

---

## Environment Override Workflows

### Override Specific Settings

**Development → Production:**

```yaml
# config.yaml (same for all environments)
service:
  name: contextd
  log_level: info

server:
  http:
    port: 8080
```

```bash
# Development (uses config.yaml defaults)
contextd

# Production (override via environment)
export CONTEXTD_SERVICE_ENVIRONMENT=production
export CONTEXTD_SERVICE_LOG_LEVEL=warn
export CONTEXTD_HTTP_PORT=9000
contextd
```

### TEI → OpenAI Switch

**Switch embedding provider without editing config:**

```yaml
# config.yaml
embedding:
  provider: tei
  tei:
    base_url: http://localhost:8080
```

```bash
# Override to OpenAI
export CONTEXTD_EMBEDDING_PROVIDER=openai
contextd
```

---

## Hot Reload Workflows

### SIGHUP Reload

**Update config without restart:**

```bash
# 1. Service is running
contextd &

# 2. Edit config
vim ~/.config/contextd/config.yaml

# 3. Reload (no downtime)
killall -HUP contextd

# 4. Verify reload
curl http://localhost:8080/health
```

### API Reload

**Trigger reload via HTTP:**

```bash
# Update config
vim ~/.config/contextd/config.yaml

# Reload via API
curl -X POST http://localhost:8080/admin/reload

# Response:
# {"status": "ok", "message": "Configuration reloaded"}
```

### File Watching (Automatic)

**Automatic reload on file change:**

```bash
# Start with file watching enabled (default)
contextd

# Edit config.yaml
vim ~/.config/contextd/config.yaml

# Wait 2 seconds (debounce period)
# Service automatically reloads
# Log output: [INFO] Configuration reloaded from file change
```

---

## Migration Workflows

### Generate config.yaml from Environment

**Create YAML from current env vars:**

```bash
# Current state: environment-only
export CONTEXTD_HTTP_PORT=8080
export CONTEXTD_DATABASE_QDRANT_HOST=localhost

# Generate config.yaml
contextd config generate > ~/.config/contextd/config.yaml

# Review generated config
cat ~/.config/contextd/config.yaml

# Validate
contextd config validate

# Apply
systemctl --user restart contextd
```

### Interactive Migration

**Step-by-step migration:**

```bash
contextd config migrate

# Output:
# Detected environment variables:
#   CONTEXTD_HTTP_PORT=8080
#   CONTEXTD_DATABASE_QDRANT_HOST=localhost
#   ...
#
# Generate config.yaml? [y/N]: y
# Preview:
#   [shows YAML preview]
# Apply? [y/N]: y
# Config written to ~/.config/contextd/config.yaml
# Validation: PASS
# Restart required. Restart now? [y/N]: y
```

---

## Security Workflows

### Setup Token Authentication

**Create token file:**

```bash
# Generate secure token (32+ bytes)
openssl rand -base64 48 > ~/.config/contextd/token

# Set permissions
chmod 0600 ~/.config/contextd/token

# Verify
ls -la ~/.config/contextd/token
# -rw------- 1 user user  65 Nov 18 12:00 token
```

**Configure in config.yaml:**

```yaml
server:
  security:
    token_file: ~/.config/contextd/token
    require_token: true
    token_min_length: 32
```

**Use token in requests:**

```bash
# Read token
TOKEN=$(cat ~/.config/contextd/token)

# Make authenticated request
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/checkpoints
```

### Setup OpenAI API Key

**Create key file:**

```bash
# Save API key to file
echo "sk-proj-..." > ~/.config/contextd/openai_api_key

# Set permissions
chmod 0600 ~/.config/contextd/openai_api_key

# Verify
ls -la ~/.config/contextd/openai_api_key
# -rw------- 1 user user  51 Nov 18 12:00 openai_api_key
```

**Configure in config.yaml:**

```yaml
embedding:
  provider: openai
  openai:
    api_key_file: ~/.config/contextd/openai_api_key
    model: text-embedding-3-small
```

---

## Validation Workflows

### Pre-Start Validation

**Validate before starting:**

```bash
# Validate config
contextd config validate

# Output (success):
# Configuration validation: PASS
# - Config file: ~/.config/contextd/config.yaml
# - Token file: ~/.config/contextd/token (permissions: 0600)
# - All required fields present
# - All values within valid ranges

# Output (failure):
# Configuration validation failed with 3 error(s):
#   1. server.http.port must be 1-65535 (got: 70000)
#      Fix with: Set CONTEXTD_HTTP_PORT to valid port (1-65535)
#   2. database.qdrant.port must be 1-65535 (got: 0)
#   3. server.security.token_file has wrong permissions: 0644 (expected: 0600)
#      Fix with: chmod 0600 ~/.config/contextd/token
```

### Continuous Validation

**Validate on every reload:**

```bash
# Edit config.yaml (introduce error)
vim ~/.config/contextd/config.yaml

# Trigger reload
killall -HUP contextd

# Log output:
# [ERROR] Config reload failed: validation error: server.http.port must be 1-65535 (got: 70000)
# [INFO] Continuing with current configuration
# [INFO] Fix config.yaml and reload again (no restart needed)

# Service continues running with old config
curl http://localhost:8080/health
# {"status": "ok"}
```

---

## Multi-Environment Workflows

### Development

```yaml
# config.yaml
service:
  environment: development
  log_level: debug

server:
  http:
    port: 8080
    host: 127.0.0.1  # localhost only

observability:
  traces:
    enabled: true
    endpoint: http://localhost:4318

development:
  debug:
    enabled: true
    pprof: true
    verbose_logging: true
```

### Production

```yaml
# config.yaml
service:
  environment: production
  log_level: warn

server:
  http:
    port: 8080
    host: 0.0.0.0  # all interfaces

  security:
    require_token: true

observability:
  traces:
    enabled: true
    endpoint: https://otel.production.example.com
    sample_rate: 0.1  # 10% sampling

database:
  backup:
    enabled: true
    interval: 12h
    retention_days: 90
```

---

## Troubleshooting Workflows

### Debug Configuration Loading

**Enable verbose logging:**

```bash
# Environment override
export CONTEXTD_SERVICE_LOG_LEVEL=debug
export CONTEXTD_DEVELOPMENT_DEBUG_VERBOSE_LOGGING=true

# Start and observe config loading
contextd

# Log output shows:
# [DEBUG] Loading config from ~/.config/contextd/config.yaml
# [DEBUG] Loaded YAML: {...}
# [DEBUG] Loading environment overrides
# [DEBUG] Override: CONTEXTD_SERVICE_LOG_LEVEL=debug
# [DEBUG] Final config: {...}
```

### Check Effective Configuration

**View merged config:**

```bash
# Dump current configuration
contextd config dump

# Output (YAML):
service:
  name: contextd
  environment: production  # from config.yaml
  log_level: debug         # from environment override
  ...
```

### Rollback on Invalid Config

**Automatic rollback:**

```bash
# Service running with valid config
contextd &

# Edit config (introduce error)
echo "invalid: yaml: {" >> ~/.config/contextd/config.yaml

# Trigger reload
killall -HUP contextd

# Log output:
# [ERROR] Config reload failed: yaml parse error
# [INFO] Keeping current configuration
# [INFO] Service continues running

# Service still healthy
curl http://localhost:8080/health
# {"status": "ok"}

# Fix config
vim ~/.config/contextd/config.yaml

# Reload again (succeeds)
killall -HUP contextd
# [INFO] Configuration reloaded successfully
```

---

## Performance Tuning Workflows

### Optimize for High Load

```yaml
# config.yaml
performance:
  workers:
    embedding: 8     # increase for more throughput
    indexing: 4

  cache:
    embedding_ttl: 48h
    query_ttl: 10m

  rate_limit:
    enabled: true
    requests_per_second: 1000
    burst: 2000

database:
  pool:
    max_connections: 50
    max_idle_connections: 25

server:
  http:
    read_timeout: 60s
    write_timeout: 60s
```

### Optimize for Low Memory

```yaml
# config.yaml
performance:
  workers:
    embedding: 2     # reduce workers
    indexing: 1

  cache:
    embedding_ttl: 6h  # shorter TTL
    query_ttl: 2m

embedding:
  cache:
    max_entries: 5000  # reduce cache size

database:
  pool:
    max_connections: 5
    max_idle_connections: 2
```

---

## Testing Workflows

### Test Configuration Changes

**Safe testing pattern:**

```bash
# 1. Backup current config
cp ~/.config/contextd/config.yaml ~/.config/contextd/config.yaml.backup

# 2. Make changes
vim ~/.config/contextd/config.yaml

# 3. Validate (dry run)
contextd config validate

# 4. Test with temporary instance
contextd --config ~/.config/contextd/config.yaml --port 9999 &

# 5. Test functionality
curl http://localhost:9999/health

# 6. If good: reload production instance
killall -HUP contextd

# 7. If bad: rollback
cp ~/.config/contextd/config.yaml.backup ~/.config/contextd/config.yaml
killall -HUP contextd
```

### Test Environment Overrides

**Verify override precedence:**

```yaml
# config.yaml
server:
  http:
    port: 8080
```

```bash
# Test override
export CONTEXTD_HTTP_PORT=9000
contextd config dump | grep port
# Output: port: 9000

# Verify precedence
unset CONTEXTD_HTTP_PORT
contextd config dump | grep port
# Output: port: 8080
```
