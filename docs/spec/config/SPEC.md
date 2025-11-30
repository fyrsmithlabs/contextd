# Configuration Specification

**Feature**: Configuration Management
**Status**: Draft
**Created**: 2025-11-23

---

## Overview

Unified configuration for contextd using Koanf with layered sources. Supports YAML config files, environment variable overrides, and struct-based validation.

| Goal | Description |
|------|-------------|
| **Layered** | Defaults -> file -> env -> flags (later overrides earlier) |
| **Type-safe** | Unmarshal to Go structs with validation |
| **12-Factor** | Environment variables for production deployment |
| **Fail-fast** | Validate on startup, reject invalid config |
| **No secrets in files** | Secrets via env vars only |

### Non-Goals (MVP)

- Remote configuration (Consul, Vault) - Phase 2
- Live reload in production - restart for config changes
- GUI configuration

---

## Quick Reference

| Component | Technology |
|-----------|------------|
| Library | Koanf v2 |
| Format | YAML |
| Validation | go-playground/validator |
| Secret handling | Custom `Secret` type (auto-redacted) |

### Config File Locations (Priority Order)

1. `--config` flag (explicit path)
2. `./config.yaml`
3. `./contextd.yaml`
4. `~/.config/contextd/config.yaml`
5. `/etc/contextd/config.yaml`

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| Koanf over Viper | 3x smaller binary, modular, correct key handling |
| YAML only | Simplicity, readable, widely supported |
| Env prefix | `CONTEXTD_` for all environment variables |
| Fail-fast validation | Reject invalid config at startup |

---

## Package Structure

```
internal/config/
├── config.go       # Root Config struct, Load function
├── defaults.go     # Default values
├── server.go       # ServerConfig
├── qdrant.go       # QdrantConfig
├── telemetry.go    # TelemetryConfig
├── logging.go      # LoggingConfig
├── scrubber.go     # ScrubberConfig
├── types.go        # Duration, Secret types
├── validation.go   # Custom validators
└── testing.go      # Test helpers
```

---

## Architecture

@./architecture.md

---

## Detailed Documentation

| Document | Contents |
|----------|----------|
| @./schema.md | Full YAML configuration reference |
| @./structs.md | Go struct definitions |
| @./validation.md | Validation rules, custom validators |
| @./environment.md | Environment variable mapping |
| @./testing.md | Test helpers |

---

## Configuration Sections

| Section | Purpose |
|---------|---------|
| `server` | gRPC and HTTP server settings |
| `qdrant` | Vector database connection |
| `telemetry` | OpenTelemetry observability |
| `logging` | Structured logging configuration |
| `scrubber` | Secret detection (gitleaks) |
| `session` | Session management, checkpoints |
| `memory` | ReasoningBank settings |
| `tools` | Tool execution limits |
| `tenancy` | Multi-tenant configuration |

---

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-001 | Load from defaults -> file -> env -> flags |
| FR-002 | YAML format for config files |
| FR-003 | All keys settable via `CONTEXTD_` env vars |
| FR-004 | Unmarshal to strongly-typed Go structs |
| FR-005 | Validate on load, reject invalid values |
| FR-006 | Redact secrets in logs/serialization |
| FR-007 | Sensible defaults for development |
| FR-008 | Cross-field validation (e.g., TLS requires certs) |
| FR-009 | Clear validation error messages |
| FR-010 | Test helpers for config creation |

---

## Success Criteria

| ID | Criterion |
|----|-----------|
| SC-001 | Config loading < 100ms |
| SC-002 | 0 secret values in log output |
| SC-003 | All fields have appropriate validation |
| SC-004 | > 80% test coverage |

---

## Implementation Phases

| Phase | Contents |
|-------|----------|
| **1** | Koanf setup, YAML loading, struct definitions |
| **2** | Environment variables, Secret type |
| **3** | CLI flags, pflag integration |
| **4** | Validation, custom validators |
| **5** | Test helpers, documentation |

---

## Future Capabilities (Post-MVP)

Koanf natively supports for future integration:

| Provider | Use Case | Priority |
|----------|----------|----------|
| Consul | Non-secret config, service discovery | P2 |
| Vault | Secrets management, rotation | P2 |
| etcd | Distributed config | P3 |
| AWS Parameter Store | Cloud-native secrets | P3 |

---

## References

- [Koanf Documentation](https://github.com/knadh/koanf)
- [go-playground/validator](https://github.com/go-playground/validator)
- [12-Factor App Config](https://12factor.net/config)
