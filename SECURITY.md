# Security Policy

## Supported Versions

We actively support the following versions of contextd with security updates:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 1.0.x   | :white_check_mark: | Active support with security patches |
| 0.3.x   | :warning:          | Security fixes only (until 1.1.0 release) |
| < 0.3   | :x:                | No longer supported |

## Reporting a Vulnerability

**IMPORTANT: DO NOT open public GitHub issues for security vulnerabilities.**

We take security seriously and appreciate responsible disclosure. If you discover a security vulnerability in contextd, please follow these steps:

### How to Report

**Email**: security@fyrsmithlabs.com

**Include in your report**:
1. **Description** - Clear explanation of the vulnerability
2. **Impact** - Potential security impact and attack scenarios
3. **Steps to Reproduce** - Detailed reproduction steps
4. **Proof of Concept** - Code, commands, or configuration demonstrating the issue
5. **Suggested Fix** - (Optional) Proposed remediation

### What to Expect

- **Initial Response**: Within 48 hours of your report
- **Status Updates**: Regular updates as we investigate and develop a fix
- **Disclosure Timeline**: We follow a 90-day coordinated disclosure policy
- **Credit**: We acknowledge security researchers in our release notes (unless you prefer anonymity)

### Security Update Process

1. **Validation**: We validate and assess the severity of the report
2. **Fix Development**: We develop and test a fix in a private branch
3. **CVE Assignment**: For high-severity issues, we request a CVE identifier via GitHub Security Advisories
4. **Release**: Security patches are released as patch versions (e.g., 1.0.x)
5. **Disclosure**: Public disclosure after users have had time to update (typically 7-14 days)

## Security Features

Contextd implements defense-in-depth security with multiple layers:

### Multi-Tenant Isolation
- **Payload-based tenant filtering** with fail-closed behavior
- **Context enforcement** via `TenantFromContext()` pattern
- **Filter injection protection** - blocks user-supplied `tenant_id`/`team_id` fields
- **Error on missing tenant** - operations fail rather than exposing data

### Secret Protection
- **gitleaks integration** with 97% test coverage
- **Automatic scrubbing** of all MCP tool responses
- **Pattern detection** for API keys, tokens, credentials, PII
- **Defense-in-depth** - scrubbing at both tool and server levels

### Production Security
- **Production mode** enforcement with `CONTEXTD_PRODUCTION_MODE=1`
- **Authentication requirements** in production deployments
- **TLS support** for external services (Qdrant, OpenTelemetry)
- **Non-root containers** with minimal attack surface

### Supply Chain Security
- **Dependency verification** via `go mod verify`
- **Vulnerability scanning** with govulncheck
- **SBOM generation** for release transparency
- **Signed releases** (planned for 1.1)

## Security Best Practices

When deploying contextd in production:

### Required Configuration
```bash
# Enable production mode (enforces security constraints)
export CONTEXTD_PRODUCTION_MODE=1

# Never disable tenant isolation in production
# (NoIsolation mode blocked when PRODUCTION_MODE=1)

# Use TLS for external services
export QDRANT_TLS_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=https://...
```

### Recommended Hardening
- Run containers as non-root user (UID 1000)
- Use read-only root filesystem with writable `/data` volume
- Enable seccomp profile for syscall filtering
- Drop unnecessary Linux capabilities
- Configure rate limiting for external HTTP server
- Enable audit logging for compliance requirements

### Monitoring
- Monitor `contextd_scrubber_secrets_detected` metric for secret exposure attempts
- Alert on `contextd_tenant_isolation_violations` (if any)
- Track `contextd_production_mode_enabled` gauge to verify production config

## Security Advisories

Security advisories are published via:
- **GitHub Security Advisories**: https://github.com/fyrsmithlabs/contextd/security/advisories
- **Release Notes**: Security fixes noted in CHANGELOG.md
- **Email Notifications**: (If you've subscribed to repository notifications)

## Security Audit History

| Date | Scope | Auditor | Report |
|------|-------|---------|--------|
| 2025-12-25 | Pre-1.0 consensus review | Internal (4-agent analysis) | See docs/1.0-RELEASE-GAPS.md |
| TBD | External security audit | TBD | Planned for post-1.0 |

## Contact

For security concerns, questions, or to request our PGP key:
- **Email**: security@fyrsmithlabs.com
- **Security Team**: Available 24/7 for critical vulnerabilities

---

**Last Updated**: 2025-12-25
**Policy Version**: 1.0
