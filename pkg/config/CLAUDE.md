# Package: config

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides application configuration management for contextd. Loads configuration from environment variables with sensible defaults, managing server, authentication, telemetry, vector database, embedding, and backup settings.

## Specification

**Full Spec**: Configuration is documented in [`pkg/CLAUDE.md`](../CLAUDE.md)

**Quick Summary**:
- **Problem**: Centralized configuration from environment variables
- **Solution**: Struct-based config with environment variable mapping
- **Key Features**:
  - Environment variable loading with defaults
  - Multi-tenant mode configuration (always enabled)
  - Embedding provider configuration (OpenAI/TEI)

## Architecture

**Design Pattern**: Environment-based configuration with defaults

**Dependencies**:
- `os` - Environment variable access
- `pkg/vectorstore/adapter` - Vector database adapter types

**Used By**:
- `cmd/contextd` - Server initialization
- `cmd/ctxd` - Client tools
- All packages requiring configuration

## Key Components

### Main Types

```go
type Config struct {
    Server    Server
    Auth      Auth
    OTEL      OTEL
    VectorDB  VectorDB    // Vector database selection
    Qdrant    Qdrant
    Embedding Embedding
    Backup    Backup
}
```

### Main Functions

```go
// Load configuration from environment variables
func Load() *Config

// Get home directory path
func GetHomeDir() string

// Expand path with ~ to home directory
func ExpandPath(path string) string
```

## Usage Example

```go
// Load configuration
cfg := config.Load()

// Access server configuration
fmt.Println("HTTP Port:", cfg.Server.HTTPPort)
fmt.Println("HTTP Host:", cfg.Server.HTTPHost)
fmt.Println("Base URL:", cfg.Server.BaseURL)

// Check vector database selection
} else {
    fmt.Println("Using Qdrant:", cfg.Qdrant.URI)
}

// Access embedding configuration
fmt.Println("Embedding URL:", cfg.Embedding.BaseURL)
fmt.Println("Model:", cfg.Embedding.Model)
```

## Testing

**Test Coverage**: 85% (Target: ≥80%)

**Key Test Files**:
- `config_test.go` - Configuration loading, environment variables
- `merger_test.go` - Configuration merging logic

**Running Tests**:
```bash
go test ./pkg/config/
go test -cover ./pkg/config/
go test -race ./pkg/config/
```

## Configuration

**Environment Variables**:

### Server Configuration
- `CONTEXTD_HTTP_PORT` - HTTP server port (default: `8080`)
- `CONTEXTD_HTTP_HOST` - Bind address (default: `0.0.0.0` for remote access)
- `CONTEXTD_BASE_URL` - Base URL for MCP clients (default: `http://localhost:8080`)

### Authentication
- `CONTEXTD_TOKEN_PATH` - Token file path (default: `~/.config/contextd/token`)

### Vector Database
- `QDRANT_URI` - Qdrant connection string (default: `http://localhost:6333`)
- `QDRANT_API_KEY` - Qdrant API key (optional)

### Embedding
- `EMBEDDING_BASE_URL` - Embedding service URL (default: OpenAI)
- `EMBEDDING_MODEL` - Model name (default: `text-embedding-3-small`)
- `EMBEDDING_TIMEOUT` - Request timeout in seconds (default: `90`)
- `OPENAI_API_KEY` - OpenAI API key (required for OpenAI provider)

### Telemetry
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint
- `OTEL_SERVICE_NAME` - Service name for tracing (default: `contextd`)
- `OTEL_ENVIRONMENT` - Environment name (default: `development`)

### Backup
- `BACKUP_ENABLED` - Enable automatic backups (default: `true`)
- `BACKUP_INTERVAL` - Backup interval (default: `24h`)
- `BACKUP_RETENTION` - Backup retention period (default: `7d`)

## Security Considerations

**CRITICAL Security Requirements**:

1. **Token Path Permissions**:
   - Token file path defaults to 0600 permissions
   - NEVER log or expose token file contents

2. **API Keys**:
   - OpenAI API key loaded from environment ONLY
   - NEVER hardcode API keys
   - NEVER log API keys

3. **HTTP Configuration**:
   - HTTP port MUST be valid (1-65535)
   - Host binding: 0.0.0.0 for remote, 127.0.0.1 for localhost only
   - Base URL MUST match actual deployment for MCP client config

4. **Configuration Validation**:
   - Invalid configuration should fail fast at startup
   - Required fields validated before service starts

## Performance Notes

- **Load time**: <1ms (environment variable reads only)
- **Memory**: ~1KB (small configuration struct)
- **No I/O**: Configuration loading is CPU-only (environment variables)

## Related Documentation

- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
- Server: [`cmd/contextd/CLAUDE.md`](../../cmd/contextd/CLAUDE.md)
