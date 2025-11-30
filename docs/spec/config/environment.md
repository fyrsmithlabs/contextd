# Environment Variables

**Parent**: @./SPEC.md

Environment variable mapping and secrets handling.

---

## Naming Convention

```
CONTEXTD_{SECTION}_{KEY}
CONTEXTD_{SECTION}_{SUBSECTION}_{KEY}
```

**Transformation**:
- Prefix: `CONTEXTD_`
- Delimiter: `_` maps to `.` in config
- Case: UPPERCASE in env, lowercase in config

---

## Common Mappings

| Config Path | Environment Variable |
|-------------|---------------------|
| `server.grpc.port` | `CONTEXTD_SERVER_GRPC_PORT` |
| `server.http.port` | `CONTEXTD_SERVER_HTTP_PORT` |
| `qdrant.host` | `CONTEXTD_QDRANT_HOST` |
| `qdrant.port` | `CONTEXTD_QDRANT_PORT` |
| `qdrant.api_key` | `CONTEXTD_QDRANT_API_KEY` |
| `telemetry.enabled` | `CONTEXTD_TELEMETRY_ENABLED` |
| `telemetry.endpoint` | `CONTEXTD_TELEMETRY_ENDPOINT` |
| `logging.level` | `CONTEXTD_LOGGING_LEVEL` |
| `logging.format` | `CONTEXTD_LOGGING_FORMAT` |
| `tenancy.enabled` | `CONTEXTD_TENANCY_ENABLED` |
| `tenancy.jwt.secret` | `CONTEXTD_TENANCY_JWT_SECRET` |

---

## Secrets (MUST Use Environment Variables)

| Secret | Environment Variable | Never in file |
|--------|---------------------|---------------|
| Qdrant API key | `CONTEXTD_QDRANT_API_KEY` | Yes |
| JWT secret | `CONTEXTD_TENANCY_JWT_SECRET` | Yes |
| Embedding API key | `CONTEXTD_MEMORY_EMBEDDING_API_KEY` | Yes |

**Security**: These values MUST NOT appear in config files. Use `Secret` type in structs to auto-redact in logs.

---

## Environment Provider Implementation

```go
// Koanf env provider setup
envProvider := env.Provider("CONTEXTD_", ".", func(s string) string {
    // CONTEXTD_SERVER_GRPC_PORT -> server.grpc.port
    return strings.Replace(
        strings.ToLower(strings.TrimPrefix(s, "CONTEXTD_")),
        "_", ".", -1,
    )
})

if err := k.Load(envProvider, nil); err != nil {
    return nil, fmt.Errorf("load env vars: %w", err)
}
```

---

## Override Priority

Environment variables override config file values:

```
Defaults < File < Environment < Flags
```

Example:
```yaml
# config.yaml
server:
  grpc:
    port: 50051
```

```bash
# Override via environment
export CONTEXTD_SERVER_GRPC_PORT=9090
# Result: port = 9090
```

---

## Boolean Values

| Value | Interpreted As |
|-------|----------------|
| `true`, `1`, `yes` | true |
| `false`, `0`, `no` | false |

```bash
CONTEXTD_TELEMETRY_ENABLED=false
CONTEXTD_TENANCY_ENABLED=1
```

---

## Duration Values

Use Go duration format:

| Format | Example |
|--------|---------|
| Seconds | `30s` |
| Minutes | `5m` |
| Hours | `1h` |
| Combined | `1h30m` |

```bash
CONTEXTD_SERVER_SHUTDOWN_TIMEOUT=60s
CONTEXTD_SESSION_IDLE_TIMEOUT=30m
```

---

## Array Values

Not directly supported via environment. Use config file for arrays.

For critical array overrides, use comma-separated custom parsing:

```bash
# Custom handling required
CONTEXTD_TOOLS_BASH_BLOCKED_COMMANDS="rm -rf /,mkfs,dd"
```

---

## Development vs Production

### Development

```bash
# Minimal environment
export CONTEXTD_LOGGING_LEVEL=debug
export CONTEXTD_LOGGING_FORMAT=console
```

### Production

```bash
# Required secrets
export CONTEXTD_QDRANT_API_KEY=your-key
export CONTEXTD_TENANCY_JWT_SECRET=your-secret
export CONTEXTD_MEMORY_EMBEDDING_API_KEY=your-key

# Production overrides
export CONTEXTD_TELEMETRY_INSECURE=false
export CONTEXTD_LOGGING_LEVEL=info
```

---

## Docker Example

```dockerfile
ENV CONTEXTD_SERVER_GRPC_PORT=50051
ENV CONTEXTD_SERVER_HTTP_PORT=8080
ENV CONTEXTD_QDRANT_HOST=qdrant
ENV CONTEXTD_QDRANT_PORT=6334
ENV CONTEXTD_TELEMETRY_ENDPOINT=otel-collector:4317
```

---

## Kubernetes Secret Reference

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: contextd
    env:
    - name: CONTEXTD_QDRANT_API_KEY
      valueFrom:
        secretKeyRef:
          name: contextd-secrets
          key: qdrant-api-key
    - name: CONTEXTD_TENANCY_JWT_SECRET
      valueFrom:
        secretKeyRef:
          name: contextd-secrets
          key: jwt-secret
```
