// Package logging provides structured logging with OpenTelemetry integration.
//
// # Overview
//
// Logging package wraps Zap with:
//   - Custom Trace level (-2, below Debug)
//   - Dual output (stdout + OpenTelemetry)
//   - Automatic context field injection (trace_id, tenant, session)
//   - Defense-in-depth secret redaction
//   - Level-aware sampling (errors never sampled)
//
// # Usage
//
// Create logger from config:
//
//	cfg := logging.NewDefaultConfig()
//	logger, err := logging.NewLogger(cfg, otelProvider)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer logger.Sync()
//
// Log with context:
//
//	ctx := logging.WithTenant(ctx, &logging.Tenant{OrgID: "acme"})
//	ctx = logging.WithSessionID(ctx, "sess_123")
//	logger.Info(ctx, "request processed", zap.Duration("duration", d))
//
// Output includes automatic correlation:
//
//	{
//	  "ts": "2025-11-24T10:15:30Z",
//	  "level": "info",
//	  "msg": "request processed",
//	  "trace_id": "abc123",
//	  "tenant.org": "acme",
//	  "session.id": "sess_123",
//	  "duration": "45ms"
//	}
//
// # Configuration Precedence
//
// Configuration follows standard contextd precedence:
//  1. Defaults (NewDefaultConfig)
//  2. File (config.yaml)
//  3. Environment variables (CONTEXTD_LOGGING_*)
//
// # Secret Redaction
//
// Secrets are redacted at multiple layers:
//  1. Domain primitives (config.Secret type)
//  2. Encoder-level field name filtering
//  3. Encoder-level pattern matching
//
// Use helpers for manual redaction:
//
//	logger.Info(ctx, "auth received",
//	    logging.RedactedString("authorization", authHeader))
//
// # Sampling
//
// Level-aware sampling prevents log floods:
//   - Trace: first 1 per second, drop rest
//   - Debug: first 10 per second, drop rest
//   - Info: first 100, then 1 every 10
//   - Warn: first 100, then 1 every 100
//   - Error+: never sampled
//
// Disable for debugging:
//
//	cfg.Sampling.Enabled = false
//
// # Testing
//
// Use TestLogger for test assertions:
//
//	tl := logging.NewTestLogger()
//	tl.Info(ctx, "test message", zap.String("key", "value"))
//	tl.AssertLogged(t, zapcore.InfoLevel, "test message")
//	tl.AssertField(t, "test message", "key", "value")
//	tl.AssertNoSecrets(t)
//
// # Concurrency Safety
//
// Logger is safe for concurrent use. Child loggers (With, Named) are
// independent and do not affect parent or siblings.
//
// # Performance
//
// Logging overhead: <1ms per entry in hot paths
// Zero allocations when level disabled
// Sampling reduces volume by ~90% in high-throughput scenarios
package logging
