# Config Package

Configuration loading for contextd-v2 using Koanf library.

## Features

- **Environment-based**: Load from environment variables
- **File-based**: Load from YAML config files with path validation
- **Hierarchical**: Environment variables override YAML
- **Secure**: File permission checks, path traversal prevention
- **Validated**: Configuration validation with clear error messages

## Usage

### Simple Environment-based Loading

```go
cfg := config.Load()
fmt.Println("Port:", cfg.Server.Port)
```

### File-based Loading with YAML

```go
cfg, err := config.LoadWithFile("~/.config/contextd/config.yaml")
if err != nil {
    log.Fatal(err)
}
```

## Configuration Structure

```go
type Config struct {
    Server        ServerConfig
    Observability ObservabilityConfig
    PreFetch      PreFetchConfig
    Checkpoint    CheckpointConfig
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 9090 | HTTP server port |
| `SERVER_SHUTDOWN_TIMEOUT` | 10s | Graceful shutdown timeout |
| `OTEL_ENABLE` | true | Enable OpenTelemetry |
| `OTEL_SERVICE_NAME` | contextd | Service name for traces |
| `PREFETCH_ENABLED` | true | Enable pre-fetch engine |
| `CHECKPOINT_MAX_CONTENT_SIZE_KB` | 1024 | Max checkpoint size (KB) |

## Security

### File Permissions

Config files MUST have secure permissions:
- `0600` (owner read/write only) - recommended
- `0400` (owner read only) - also accepted

### Path Validation

Only config files in allowed directories:
- `~/.config/contextd/` (user config)
- `/etc/contextd/` (system-wide)

### File Size Limit

Config files larger than 1MB are rejected.

## Testing

Run tests with:
```bash
go test ./internal/config -v
```

## Migration Notes

Ported from `contextd/pkg/config` to `contextd-v2/internal/config`:
- Updated import paths
- Removed dependencies on unused packages
- Kept Koanf-based approach
- All tests passing
