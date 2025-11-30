# internal/secrets

Secret scrubbing using gitleaks for all contextd output.

**Last Updated**: 2025-11-29

---

## What This Package Is

**Purpose**: Prevent secret leakage in tool responses using gitleaks detection

**Spec**: @../../docs/spec/interface/SPEC.md (Security section)

**Integration**: gRPC interceptor + direct API

---

## Architecture

**Two-layer scrubbing**:
1. **Server-level**: gRPC interceptor scrubs all responses
2. **Tool-level**: Direct scrubbing for specific tools

**gitleaks Integration**: SDK, not CLI (in-process)

---

## Key Components

| Component | Purpose |
|-----------|---------|
| `Scrubber` | Main interface, wraps gitleaks |
| `Interceptor` | gRPC interceptor for response scrubbing |
| `Config` | Scrubbing patterns, rules |
| `Result` | Scrubbing result (redacted content + findings) |

---

## Scrubbing Rules

**What gets scrubbed**:
- API keys, tokens, passwords
- Private keys, certificates
- Database credentials
- AWS keys, secrets
- Custom patterns (configurable)

**What's preserved**:
- Rule IDs (for telemetry)
- Finding types (for metrics)
- Finding counts (for observability)

**Never scrub**: Error messages, exit codes, file paths, command summaries

---

## Testing

**Critical Tests**:
- Secret detection (all gitleaks rules)
- No false positives (code samples, placeholders)
- Performance (scrubbing <10ms for 1KB)
- Telemetry (metrics capture findings)

**Coverage Target**: >90% (security-critical)

---

## References

- gitleaks: https://github.com/gitleaks/gitleaks
- Security requirements: @../../docs/spec/interface/SPEC.md#security
