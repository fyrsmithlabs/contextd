# internal/http

HTTP API server for contextd, exposing read-only status and health endpoints for the `ctxd statusline` and external monitoring.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET`  | `/health` | Process health + optional metadata-integrity status |
| `GET`  | `/api/v1/health/metadata` | Full metadata health detail (loopback-only) |
| `GET`  | `/metrics` | Prometheus metrics |
| `GET`  | `/api/v1/status` | Service status, version, collection counts, optional memory stats |

## Removed endpoints

Earlier versions exposed `POST /api/v1/scrub`, `POST /api/v1/threshold`, and a `POST /checkpoint/*` family. These were removed:

- **`/api/v1/scrub`** and **`/api/v1/threshold`** — replaced by MCP tool flows. Agents should use the MCP `secrets_scrub` and `checkpoint_save` paths instead of a separate HTTP surface.
- **`/checkpoint/*`** — removed in CVE-2025-CONTEXTD-001 fix. Use the `checkpoint_save`, `checkpoint_list`, and `checkpoint_resume` MCP tools.

## Configuration

| Setting | Default | Notes |
|---------|---------|-------|
| `Host`  | `localhost` | Bind address |
| `Port`  | `9090`    | Bind port |
| `HealthChecker` | `nil` | Optional `vectorstore.MetadataHealthChecker` for the `/health` integrity field |

## Middleware

- Request ID injection (`X-Request-ID`)
- Panic recovery
- Request logging (zap)
- OTEL metrics middleware

## Testing

```bash
go test ./internal/http/...
```
