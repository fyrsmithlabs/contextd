# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.3.x   | :white_check_mark: |
| < 0.3   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to security@fyrsmith.com (or appropriate security contact).

You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

Please include the following information in your report:

- Type of vulnerability (e.g., path traversal, command injection, etc.)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Security Features

contextd implements several security measures:

### Multi-Tenant Isolation

- **Payload-based tenant isolation** with fail-closed behavior
- Tenant context required for all operations (prevents data leakage)
- Filter injection protection (user-provided filters cannot bypass tenant boundaries)

### Configuration Security

- **File Permission Validation**: Config files must have 0600 or 0400 permissions
- **Path Validation**: Only allows config files in `~/.config/contextd/` or `/etc/contextd/`
- **Symlink Resolution**: Prevents symlink-based path traversal attacks
- **Size Limits**: Config files limited to 1MB to prevent resource exhaustion
- **Environment Variable Validation**: Validates paths, hostnames, and URLs from env vars

### Production Mode

Production mode enforces additional security:

- **NoIsolation Blocking**: Prevents disabling tenant isolation in production
- **Authentication Requirements**: Can require authentication to be configured
- **TLS Requirements**: Can enforce TLS for production deployments
- **Local Mode Override**: Allows local development without auth/TLS when explicitly acknowledged

Enable production mode:
```bash
export CONTEXTD_PRODUCTION_MODE=1
```

### Secret Scrubbing

- **Automatic Secret Detection**: Uses gitleaks SDK to detect secrets in all tool responses
- **High Coverage**: 97% test coverage for secret scrubbing functionality
- **Fail-Closed**: Blocks responses containing detected secrets

### Input Validation

- **Hostname Validation**: Prevents command injection in hostnames
- **Path Validation**: Prevents path traversal in file paths
- **URL Validation**: Only allows http:// and https:// schemes

## Security Best Practices

When deploying contextd:

1. **Use Production Mode** in all non-development environments
2. **Enable TLS** for all network communication
3. **Set Strict File Permissions** on configuration files (0600)
4. **Validate Environment Variables** - avoid untrusted input in env vars
5. **Monitor Logs** for security events and anomalies
6. **Keep Updated** to the latest patch version
7. **Use Authentication** in production deployments

## Known Limitations

- **Browser-based attacks**: If using the MCP protocol over untrusted networks, ensure TLS is enabled
- **Supply chain**: Verify checksums of downloaded binaries against published hashes
- **Secrets in memory**: Secrets may remain in memory until garbage collected

## Security Update Process

1. Security reports are triaged within 48 hours
2. Severity is assessed (Critical, High, Medium, Low)
3. Fixes are developed and tested
4. Security advisory is published
5. Patch release is created and announced
6. Users are notified via GitHub Security Advisories

## Disclosure Policy

- **Coordinated Disclosure**: We follow a 90-day coordinated disclosure policy
- **Credit**: Security researchers who report valid vulnerabilities will be credited (unless they prefer to remain anonymous)
- **No Bounty Program**: We currently do not offer a bug bounty program

## Contact

- **Security Email**: security@fyrsmith.com
- **PGP Key**: Available upon request
- **Response Time**: Within 48 hours

## Acknowledgments

We thank the security researchers who have helped improve contextd's security.

---

Last updated: 2025-12-26
