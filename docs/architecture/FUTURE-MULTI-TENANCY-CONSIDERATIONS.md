# Future Multi-Tenancy Considerations

**Date**: 2025-01-08
**Status**: Deferred (post-MVP)
**Context**: Brainstorming session on advanced multi-tenancy requirements

---

## Current MVP Scope (v2.1)

**Problem**: Same developer, multiple projects, context confusion
- Example: Go/Qdrant patterns from contextd appearing in Java/Minecraft project
- Example: Terraform patterns mixed with Docker patterns

**Solution**: Config-first tag-based filtering
- `.contextd.yaml` defines tech_stack, infrastructure, domain
- Search prioritizes matching facets (domain → tech_stack → owner)
- Owner-based database isolation (owner_personal, owner_workcorp, etc.)

**Security Model**: Physical database boundaries per git remote owner
- `owner_personal/` (git@github.com:you/*)
- `owner_workcorp/` (git@github.com:workcorp/*)
- No cross-owner access via vector DB isolation

---

## Future Enterprise Requirements (0.9.0-rc-1+)

### 1. Cross-System Access Control

**Problem**: Employee uses contextd on both personal and work laptops
- Personal laptop should NOT access work patterns
- Work laptop should NOT access personal patterns
- Same person, different physical systems

**Requirements**:
- Device identity (not just git remote owner)
- Network-based access control (VPN, corp network)
- Authentication tokens scoped to system/device

**Potential Solutions**:
- **A. Separate Instances**: Different contextd per system (network isolation)
- **B. Token-Based Auth**: Device-specific auth tokens with RBAC
- **C. Device Posture**: MDM integration, certificate-based auth (corp PKI)

**Status**: Deferred - requires OAuth/SSO, device management
**Rationale**: MVP targets single-developer workflow, not corp compliance

---

### 2. Shared Patterns with Access Control

**Problem**: Team wants to share patterns but restrict by role/clearance
- Junior devs see "approved patterns" only
- Senior devs can publish to org knowledge base
- Contractors have read-only access

**Requirements**:
- Role-based access control (RBAC)
- Pattern approval workflow
- Visibility levels: private → team → org → public

**Status**: Deferred - requires team/org detection from CODEOWNERS
**Rationale**: MVP focuses on individual developer isolation

---

### 3. Tag Namespace Collision Prevention

**Problem**: Different orgs use same tag names differently
- "kubernetes" at Company A = GKE + Istio
- "kubernetes" at Company B = EKS + plain k8s
- Tag collision causes wrong pattern retrieval

**Current MVP**: Owner-scoped databases prevent collision
**Future Enhancement**: Tag namespacing (personal:kubernetes vs workcorp:kubernetes)

**Status**: Partially solved by database isolation, enhancement deferred

---

### 4. Compliance Requirements

**Problem**: Enterprise compliance (SOC 2, HIPAA, GDPR)
- Audit logs for pattern access
- Data retention policies
- Encryption at rest/transit
- Right to be forgotten (GDPR)

**Status**: Deferred - MVP targets personal use
**Rationale**: Single-user deployment doesn't require compliance

---

## Decision: MVP First, Enterprise Later

**Prioritization**:
1. **v2.1 MVP**: Config-first tag filtering + owner-based isolation
2. **v2.2**: Team/org detection from CODEOWNERS (if needed)
3. **0.9.0-rc-1**: Enterprise features (RBAC, compliance, cross-system auth)

**Rationale**:
- 90% of users are individual developers (personal projects)
- Solve immediate context confusion with simple config
- Enterprise features add months of complexity
- Can iterate based on real user feedback

---

## Open Questions for Future

1. **Device Identity**: How to detect/enforce device boundaries?
   - MAC address? (spoofable)
   - TPM/Secure Enclave? (platform-specific)
   - Network location? (VPN detection)
   - User decision: "This is my work laptop" flag?

2. **Shared Patterns**: How to handle collaborative teams?
   - Push-based (publish to team DB)?
   - Pull-based (subscribe to team patterns)?
   - Hybrid (local + remote sync)?

3. **Tag Governance**: Who controls tag vocabulary?
   - Personal: user defines tags
   - Team: team lead approves tags
   - Org: admin manages taxonomy

4. **Migration Path**: How to move patterns between scopes?
   - Personal → Team (promotion)
   - Team → Org (approval workflow)
   - Org → Public (opt-in contribution)

---

## References

- **ADR-003**: Single-developer multi-repo isolation (owner-based)
- **ADR-004**: Full security implementation v2.1 (team/org)
- **ReasoningBank**: Multi-tenant pattern storage design
- **Brainstorming Session**: 2025-01-08 multi-tenancy discussion

---

**Next Steps**: Focus on MVP implementation
- Implement `.contextd.yaml` config loading
- Add faceted search (domain, tech_stack, infrastructure)
- Git remote owner detection
- Owner-scoped database creation

**Defer**: Enterprise auth, cross-system isolation, RBAC
