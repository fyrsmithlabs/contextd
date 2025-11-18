# Feature: Bearer Token Authentication System

**Version**: 1.0.0
**Status**: ⏸ Deferred (Post-MVP)
**Last Updated**: 2025-11-18

---

## Overview

> **MVP STATUS**: ⚠️ NOT APPLICABLE TO MVP
>
> This specification documents the Bearer token authentication system.
> **MVP does not implement authentication** (trusted network assumption).
>
> **Use this spec only if implementing post-MVP authentication.**
>
> For MVP architecture, see `docs/standards/architecture.md`.

The authentication system (`pkg/auth`) provides secure, lightweight bearer token authentication for contextd's Unix domain socket API. Designed for single-user, localhost scenarios with automatic token generation, constant-time comparison, and strict file permissions.

---

## Quick Reference

**Key Facts**:
- **Package**: `pkg/auth`
- **Transport**: Unix domain socket (MVP uses HTTP on port 8080)
- **Auth Method**: Bearer token (64 hex characters, 256 bits entropy)
- **Token Storage**: `~/.config/contextd/token` (0600 permissions)
- **Security**: Constant-time comparison, auto-generation, fail-secure
- **MVP Decision**: No authentication (trusted network model)

**Core Components**:
- Token Generator: Secure random token creation (crypto/rand)
- Token Loader: Auto-generation and validation
- Echo Middleware: HTTP Bearer token authentication
- Security Hardening: Constant-time, TOCTOU prevention, DoS protection

**Key Characteristics**:

| Characteristic | Value |
|---------------|-------|
| Token Length | 64 hexadecimal characters |
| Token Entropy | 256 bits (32 random bytes) |
| File Permissions | 0600 (owner read/write only) |
| Comparison Method | Constant-time (timing attack prevention) |
| Network Exposure | None (Unix socket only) |

---

## Detailed Documentation

**Requirements & Design**:
@./auth/requirements.md - Design philosophy, features, capabilities matrix
@./auth/architecture.md - System architecture, components, data flow, performance

**Security**:
@./auth/security.md - Threat model, defense layers, constant-time comparison, security checklist

**Implementation**:
@./auth/implementation.md - API specification, protocol, middleware integration, error handling, testing
@./auth/workflows.md - Usage examples, authentication flows, file operations, client patterns

---

## Summary

**Current Status**: Specification complete, implementation deferred post-MVP.

**MVP Decision**: Trusted network assumption (no authentication). Deploy behind VPN or SSH tunnel for security.

**Post-MVP Implementation**: When implementing authentication, follow this spec for Bearer token system with constant-time comparison, auto-generation, and strict security hardening.

**Related Documentation**:
- Architecture: `/docs/standards/architecture.md`
- Package Guidelines: `/docs/standards/package-guidelines.md`
- Security Audit: `/docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md`
- MCP Specification: `/docs/specs/mcp/SPEC.md`
