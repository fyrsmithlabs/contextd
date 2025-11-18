# Configuration Reference

**Parent**: [../SPEC.md](../SPEC.md)

This document provides complete reference tables for all configuration options.

---

## Service Options

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

---

## Server Options

| Option | Type | Default | Env Variable | Range/Format | Description |
|--------|------|---------|--------------|--------------|-------------|
| `server.http.port` | int | `8080` | `CONTEXTD_HTTP_PORT` | 1-65535 | HTTP server port |
| `server.http.host` | string | `0.0.0.0` | `CONTEXTD_HTTP_HOST` | IP/hostname | Bind address (0.0.0.0=all, 127.0.0.1=localhost) |
| `server.http.base_url` | string | `http://localhost:8080` | `CONTEXTD_BASE_URL` | URL | Base URL for MCP client configuration |
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
export CONTEXTD_HTTP_PORT=8080
export CONTEXTD_HTTP_HOST=0.0.0.0
export CONTEXTD_BASE_URL=http://localhost:8080
export CONTEXTD_SERVER_HTTP_READ_TIMEOUT=60s
```

---

## Database Options

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

---

## Embedding Options

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

## Observability Options

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `observability.traces.enabled` | bool | `true` | `CONTEXTD_OBSERVABILITY_TRACES_ENABLED` | true/false | Enable tracing |
| `observability.traces.endpoint` | string | `http://localhost:4318` | `CONTEXTD_OBSERVABILITY_TRACES_ENDPOINT` | URL | OTLP endpoint |
| `observability.traces.sample_rate` | float | `1.0` | `CONTEXTD_OBSERVABILITY_TRACES_SAMPLE_RATE` | 0.0-1.0 | Sample rate |
| `observability.traces.batch_timeout` | duration | `5s` | `CONTEXTD_OBSERVABILITY_TRACES_BATCH_TIMEOUT` | 1s-60s | Batch timeout |
| `observability.traces.max_batch_size` | int | `512` | `CONTEXTD_OBSERVABILITY_TRACES_MAX_BATCH_SIZE` | 1-10000 | Max batch size |
| `observability.metrics.enabled` | bool | `true` | `CONTEXTD_OBSERVABILITY_METRICS_ENABLED` | true/false | Enable metrics |
| `observability.metrics.endpoint` | string | `http://localhost:4318` | `CONTEXTD_OBSERVABILITY_METRICS_ENDPOINT` | URL | OTLP endpoint |
| `observability.metrics.interval` | duration | `60s` | `CONTEXTD_OBSERVABILITY_METRICS_INTERVAL` | 10s-300s | Export interval |
| `observability.logging.format` | string | `json` | `CONTEXTD_OBSERVABILITY_LOGGING_FORMAT` | json, text | Log format |
| `observability.logging.output` | string | `stdout` | `CONTEXTD_OBSERVABILITY_LOGGING_OUTPUT` | stdout, stderr, path | Output destination |
| `observability.logging.add_source` | bool | `true` | `CONTEXTD_OBSERVABILITY_LOGGING_ADD_SOURCE` | true/false | Include source location |

---

## Feature Flags

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `features.mcp_mode` | bool | `false` | `CONTEXTD_FEATURES_MCP_MODE` | true/false | MCP protocol mode |
| `features.checkpoints.enabled` | bool | `true` | `CONTEXTD_FEATURES_CHECKPOINTS_ENABLED` | true/false | Enable checkpoints |
| `features.checkpoints.auto_save_interval` | duration | `5m` | `CONTEXTD_FEATURES_CHECKPOINTS_AUTO_SAVE_INTERVAL` | 1m-60m | Auto-save interval |
| `features.checkpoints.max_per_project` | int | `1000` | `CONTEXTD_FEATURES_CHECKPOINTS_MAX_PER_PROJECT` | 1-10000 | Max per project |
| `features.remediations.enabled` | bool | `true` | `CONTEXTD_FEATURES_REMEDIATIONS_ENABLED` | true/false | Enable remediations |
| `features.remediations.semantic_weight` | float | `0.7` | `CONTEXTD_FEATURES_REMEDIATIONS_SEMANTIC_WEIGHT` | 0.0-1.0 | Semantic weight |
| `features.remediations.string_weight` | float | `0.3` | `CONTEXTD_FEATURES_REMEDIATIONS_STRING_WEIGHT` | 0.0-1.0 | String weight |
| `features.remediations.min_similarity` | float | `0.6` | `CONTEXTD_FEATURES_REMEDIATIONS_MIN_SIMILARITY` | 0.0-1.0 | Min similarity |
| `features.skills.enabled` | bool | `true` | `CONTEXTD_FEATURES_SKILLS_ENABLED` | true/false | Enable skills |
| `features.skills.auto_index` | bool | `true` | `CONTEXTD_FEATURES_SKILLS_AUTO_INDEX` | true/false | Auto-index skills |
| `features.troubleshooting.enabled` | bool | `true` | `CONTEXTD_FEATURES_TROUBLESHOOTING_ENABLED` | true/false | Enable troubleshooting |
| `features.troubleshooting.max_context_tokens` | int | `4000` | `CONTEXTD_FEATURES_TROUBLESHOOTING_MAX_CONTEXT_TOKENS` | 1000-10000 | Max context tokens |
| `features.indexing.enabled` | bool | `true` | `CONTEXTD_FEATURES_INDEXING_ENABLED` | true/false | Enable indexing |
| `features.indexing.max_file_size` | size | `1MB` | `CONTEXTD_FEATURES_INDEXING_MAX_FILE_SIZE` | 100KB-10MB | Max file size |
| `features.indexing.batch_size` | int | `100` | `CONTEXTD_FEATURES_INDEXING_BATCH_SIZE` | 10-1000 | Batch size |

---

## Performance Options

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `performance.workers.embedding` | int | `4` | `CONTEXTD_PERFORMANCE_WORKERS_EMBEDDING` | 1-16 | Embedding workers |
| `performance.workers.indexing` | int | `2` | `CONTEXTD_PERFORMANCE_WORKERS_INDEXING` | 1-16 | Indexing workers |
| `performance.cache.embedding_ttl` | duration | `24h` | `CONTEXTD_PERFORMANCE_CACHE_EMBEDDING_TTL` | 1h-168h | Embedding cache TTL |
| `performance.cache.query_ttl` | duration | `5m` | `CONTEXTD_PERFORMANCE_CACHE_QUERY_TTL` | 1m-60m | Query cache TTL |
| `performance.rate_limit.enabled` | bool | `false` | `CONTEXTD_PERFORMANCE_RATE_LIMIT_ENABLED` | true/false | Enable rate limiting |
| `performance.rate_limit.requests_per_second` | int | `100` | `CONTEXTD_PERFORMANCE_RATE_LIMIT_REQUESTS_PER_SECOND` | 1-10000 | RPS limit |
| `performance.rate_limit.burst` | int | `200` | `CONTEXTD_PERFORMANCE_RATE_LIMIT_BURST` | 1-20000 | Burst size |

---

## Development Options

| Option | Type | Default | Env Variable | Valid Values | Description |
|--------|------|---------|--------------|--------------|-------------|
| `development.debug.enabled` | bool | `false` | `CONTEXTD_DEVELOPMENT_DEBUG_ENABLED` | true/false | Enable debug mode |
| `development.debug.pprof` | bool | `false` | `CONTEXTD_DEVELOPMENT_DEBUG_PPROF` | true/false | Enable pprof |
| `development.debug.verbose_logging` | bool | `false` | `CONTEXTD_DEVELOPMENT_DEBUG_VERBOSE_LOGGING` | true/false | Verbose logs |
| `development.profile.cpu` | bool | `false` | `CONTEXTD_DEVELOPMENT_PROFILE_CPU` | true/false | CPU profiling |
| `development.profile.memory` | bool | `false` | `CONTEXTD_DEVELOPMENT_PROFILE_MEMORY` | true/false | Memory profiling |
| `development.profile.output_dir` | string | `/tmp/contextd/profiles` | `CONTEXTD_DEVELOPMENT_PROFILE_OUTPUT_DIR` | directory path | Profile output |
| `development.mocks.embedding` | bool | `false` | `CONTEXTD_DEVELOPMENT_MOCKS_EMBEDDING` | true/false | Mock embedding |
| `development.mocks.database` | bool | `false` | `CONTEXTD_DEVELOPMENT_MOCKS_DATABASE` | true/false | Mock database |
