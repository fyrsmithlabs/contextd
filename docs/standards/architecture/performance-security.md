# Performance & Security Considerations

**Parent**: [Architecture Standards](../architecture.md)

This document describes performance targets and security considerations for contextd.

---

## Performance Considerations

### Response Time Targets

- **Health checks**: <10ms
- **Checkpoint save**: <100ms
- **Checkpoint search**: <200ms
- **Remediation search**: <300ms (hybrid matching)
- **AI troubleshoot**: <2s (OpenAI API dependency)
- **Repository indexing**: Variable (depends on size)

### Optimization Strategies

1. **Local-First**: All operations hit local Qdrant
2. **Batch Operations**: Upsert multiple points at once
3. **Concurrent Processing**: Use goroutines for independent tasks
4. **Connection Pooling**: Reuse HTTP clients and connections
5. **Caching**: Cache embeddings for repeated content (future)

---

## Scalability Considerations

### Current Architecture (Single-User)

- **Design**: Single-user localhost service
- **Concurrency**: Handles concurrent requests via goroutines
- **Storage**: Local Qdrant (grows with usage)
- **Memory**: Bounded by Qdrant configuration

### Future Multi-User (If Needed)

- **Auth**: Move to JWT with user claims
- **Database**: User-specific databases (extend multi-tenant)
- **Transport**: Add TLS via reverse proxy (nginx/Caddy)
- **Rate Limiting**: Per-user rate limits

---

## Security Considerations

### Threat Model

**In Scope:**
- Local privilege escalation
- File permission issues
- Timing attacks on auth
- Log injection

**Out of Scope (MVP only - add post-MVP):**
- Authentication/authorization (MVP uses trusted network)
- Rate limiting (add for production)
- DDoS protection (use reverse proxy for production)

**Out of Scope (by design):**
- SQL injection (no SQL)
- XSS (no web UI)
- CSRF (no web sessions)

### Security Checklist

- ✅ HTTP server with configurable port and host
- ✅ CORS disabled by default (same-origin only)
- ✅ Rate limiting recommended for production
- ⚠️  MVP: No authentication (use VPN/SSH tunnel for security)
- ⚠️  Production: Add auth layer (Bearer token, JWT, OAuth)
- ✅ No credentials in code/config
- ✅ No credential logging
- ✅ Graceful error handling (no stack traces in responses)
- ✅ Input validation at service boundary
- ✅ Context propagation for tracing
- ✅ OTEL for security event monitoring

---

## Documentation References

- **ADRs**: `docs/adr/` - Architectural decision records
- **Research**: `docs/research/` - Investigation and analysis
- **Migration**: `docs/MIGRATION-FROM-LEGACY.md`
- **Multi-Tenant**: `docs/MULTI-TENANT-COMPLETION-STATUS.md`
- **TEI Deployment**: `docs/TEI-DEPLOYMENT.md`
