# pkg/logging

Structured logging with Uber Zap, log rotation, sampling, and OTEL integration.

## Purpose
Production-grade logging for contextd with:
- 4-10x faster than stdlib (zero-allocation)
- Structured JSON logs
- Automatic rotation via lumberjack
- Sampling for high-volume logs
- OTEL bridge for unified observability

## Key Components
- logger.go: Zap logger factory with config
- audit.go: Security audit event logging
- http.go: HTTP request/response middleware
- rotation.go: Log rotation configuration

## Usage
```go
import "github.com/axyzlabs/contextd/pkg/logging"

config := logging.Config{
    Level: "info",
    Encoding: "json",
    OutputPaths: []string{"stdout", "/var/log/contextd/app.log"},
}
logger, err := logging.NewLogger(config)
logger.Info("server_started", zap.Int("port", 8080))
```

## Specification
See [docs/specs/logging/SPEC.md](../../docs/specs/logging/SPEC.md)
