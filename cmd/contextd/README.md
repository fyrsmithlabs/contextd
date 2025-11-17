# contextd Server

**Status**: Production ready

The contextd server with HTTP/SSE transport.

See [../../docs/specs/v3-rebuild/SPEC.md](../../docs/specs/v3-rebuild/SPEC.md) for complete specification.

## Architecture

- HTTP/SSE transport (no stdio)
- MCP over HTTP endpoints
- Owner-scoped multi-tenancy
- langchaingo abstractions
- Gitleaks-based secret scrubbing

## Build

```bash
go build -o contextd ./cmd/contextd/
```

## Run

```bash
# Start HTTP server
./contextd serve

# Health check
curl http://localhost:8080/health
```

## Configuration

See [../../docs/specs/v3-rebuild/SPEC.md](../../docs/specs/v3-rebuild/SPEC.md) for configuration details.
