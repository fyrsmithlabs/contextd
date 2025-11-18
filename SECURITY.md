# Security Policy

## Supported Versions

The following versions of contextd are currently receiving security updates:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 2.0.x   | :white_check_mark: | Active development, security patches applied |
| < 2.0   | :x:                | End of life, no security patches |

**Important**: Version 2.0.0 includes critical security fixes and removes the legacy mode that had known vulnerabilities. All users should upgrade to v2.0.0 or later.

## Reporting a Vulnerability

We take the security of contextd seriously. If you discover a security vulnerability, please follow these steps:

### 1. DO NOT Publicly Disclose

Please **DO NOT** open a public GitHub issue for security vulnerabilities. This helps protect users who haven't upgraded yet.

### 2. Report via Email

Send a detailed report to: **security@fyrsmithlabs.com**

Include:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact assessment
- Suggested fix (if available)
- Your contact information for follow-up

### 3. Expected Response Timeline

- **Initial Response**: Within 48 hours acknowledging receipt
- **Status Update**: Within 7 days with initial assessment
- **Resolution Timeline**: Depends on severity
  - **Critical**: Patch within 7 days
  - **High**: Patch within 14 days
  - **Medium**: Patch within 30 days
  - **Low**: Addressed in next release cycle

### 4. Disclosure Policy

- We will work with you to understand and validate the issue
- Once a fix is ready, we will coordinate disclosure timing
- You will be credited in the security advisory (unless you prefer anonymity)
- We follow coordinated disclosure practices

## Security Update Policy

### Release Schedule

- **Critical Security Patches**: Released immediately via patch version
- **High Severity Issues**: Released within 14 days
- **Security Advisories**: Published on GitHub Security Advisories
- **User Notification**: Via GitHub releases and mailing list (when available)

### Upgrading for Security

```bash
# Check your version
contextd --version

# Upgrade to latest
# See release notes at: https://github.com/fyrsmithlabs/contextd/releases
go install github.com/fyrsmithlabs/contextd/cmd/contextd@latest

# Or download binaries from GitHub Releases
```

## Known Security Considerations

### Authentication and Authorization

- HTTP transport on configurable port (default: 8080)
- Remote access supported (0.0.0.0 binding)
- MVP: No authentication (trusted network assumption)
- Production: Use reverse proxy with TLS + auth (Bearer/JWT/OAuth)
- Recommendation: Deploy behind VPN or SSH tunnel for MVP security

### Multi-Tenant Isolation

- **v2.0.0+**: Database-per-project architecture provides complete isolation
- Each project gets a dedicated vector database
- No filter-based isolation (eliminates filter injection attacks)
- Projects cannot access each other's data

### Rate Limiting

- Per-tool rate limiting implemented in MCP server (v2.0.0+)
- Configurable rate limits per operation type
- Default: 10 requests per minute for most operations
- Checkpoint operations: 30 requests per minute

### API Key Security

- OpenAI API keys stored in `~/.config/contextd/openai_api_key` (0600)
- Never logged or included in error messages
- Never committed to version control
- API key validation before use

### Data Storage

- Vector embeddings stored in Qdrant (local by default)
- No sensitive data in embeddings (text only)
- Checkpoints include session context (review before storing sensitive data)
- Local storage at `~/.local/share/qdrant/` with user-only access

### Network Security

- HTTP server with standard security headers
- CORS disabled by default (same-origin only)
- Reverse proxy recommended for production (TLS, auth, rate limiting)
- HTTPS for external OTEL endpoint (optional)
- TEI embedding service can run locally (recommended)
- No external dependencies for core functionality

## MVP vs Production Security Posture

### MVP (Current - Trusted Network)
- ✅ HTTP server on port 8080
- ✅ No authentication required
- ⚠️  Deploy on trusted network only (VPN, internal network, or localhost)
- ⚠️  Use SSH tunnel for remote access: `ssh -L 8080:localhost:8080 user@server`

### Production (Post-MVP)
- ✅ All MVP features
- ✅ Bearer token or JWT authentication
- ✅ TLS via reverse proxy (nginx/Caddy)
- ✅ Rate limiting and DDoS protection
- ✅ OAuth/SSO for team environments
- ✅ Audit logging

### Migration Path
1. Start with MVP on trusted network
2. Add reverse proxy with TLS
3. Implement authentication (Bearer token → JWT → OAuth)
4. Add rate limiting and monitoring
5. Enable audit logging

## Recent Security Fixes

### v2.0.0 (Critical Security Release)

**Filter Injection Vulnerability (CVE-TBD)**
- **Severity**: Critical
- **Impact**: Cross-project data access in legacy mode
- **Fix**: Complete removal of legacy mode, mandatory multi-tenant architecture
- **Migration**: Required for users on v1.x
- **Details**: See `docs/MIGRATION-FROM-LEGACY.md`

**Rate Limiting**
- **Severity**: Medium
- **Impact**: Potential for API abuse
- **Fix**: Per-tool rate limiting in MCP server
- **Details**: See `docs/adr/003-rate-limiting-strategy.md`

## Security Best Practices

### For Users

1. **Keep contextd Updated**: Always run the latest version
2. **Protect API Keys**: Never commit keys, use 0600 permissions
3. **Review Checkpoints**: Don't store sensitive passwords or secrets
4. **Use TEI Locally**: Avoid sending data to external APIs when possible
5. **Monitor Logs**: Watch for unusual activity in service logs

### For Contributors

1. **Code Review**: All PRs require security review
2. **Dependency Scanning**: Automated via GitHub Dependabot
3. **SAST**: Static analysis in CI/CD pipeline
4. **Test Coverage**: Maintain >80% coverage including security tests
5. **Follow ADRs**: Architecture Decision Records document security choices

### For Deployers

1. **Filesystem Permissions**: Verify API key files are 0600
2. **User Isolation**: Run contextd as dedicated user (not root)
3. **Firewall Rules**:
   - MVP: Restrict port 8080 to trusted networks only
   - Production: Use reverse proxy (nginx/Caddy) with TLS
   - Option: SSH tunnel for remote access without exposing port
4. **Log Monitoring**: Enable systemd logging or equivalent
5. **Backup Security**: Encrypt backups if storing sensitive checkpoints

## Security Audits

- **Internal**: Continuous security testing via `@security-tester` agent
- **Community**: Security-focused code reviews on all PRs
- **External**: Open to independent security audits (contact security@fyrsmithlabs.com)

## Security-Related Documentation

- [Architecture Decision Records](/docs/adr/) - Security design decisions
- [Multi-Tenant Architecture](/docs/adr/002-universal-multi-tenant-architecture.md)
- [Rate Limiting Strategy](/docs/adr/003-rate-limiting-strategy.md)
- [Migration Guide](/docs/MIGRATION-FROM-LEGACY.md) - Security migration from v1.x

## Security Hall of Fame

We recognize and thank security researchers who help improve contextd:

- [Your name here] - Report responsibly to be listed

## Questions?

For security questions or concerns:
- **Vulnerabilities**: security@fyrsmithlabs.com
- **General Security**: Open a GitHub Discussion (for non-sensitive topics)
- **Documentation**: Suggest improvements via PR

## Acknowledgments

We follow industry best practices including:
- OWASP Top 10 guidelines
- CWE/SANS Top 25 awareness
- Secure coding standards for Go
- GitHub Security Best Practices

Thank you for helping keep contextd and its users secure!
