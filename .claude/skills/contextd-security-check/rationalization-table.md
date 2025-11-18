# Rationalization Table: contextd:security-check

This table documents all rationalizations agents might use to skip security validation, and how the skill counters them.

| Excuse | Reality | Skill Counter |
|--------|---------|---------------|
| "This is internal code, security doesn't apply" | Internal code becomes external. Defense-in-depth means EVERY layer is secure. | Anti-Pattern section + Red Flag |
| "Tests pass, so security must be fine" | Functional tests ≠ security tests. Need gosec, isolation tests, malicious input tests. | Anti-Pattern section + Security Testing requirements |
| "Will add security tests in follow-up PR" | Later never comes. Security debt accumulates. Vulnerabilities ship to production. | Anti-Pattern section + Red Flag |
| "This is too small for full security review" | One-line changes can introduce vulnerabilities. Change size ≠ security impact. | Anti-Pattern section + Red Flag |
| "Service layer already validated, repository doesn't need to" | Defense-in-depth requires validation at EVERY boundary. Single validation = single point of failure. | Anti-Pattern section + Input Validation requirements + Code example |
| "gosec is too strict" | Fix all findings. Allowing bypass creates security culture problems. | Common Failures + Anti-Pattern section |
| "gosec finding is false positive" | Even if true, fixing demonstrates security understanding. Use #nosec with detailed justification. | Common Failures + Updated anti-pattern |
| "This is an emergency, security checks after incident" | Emergency hotfixes that bypass security create new vulnerabilities. Security testing takes <5 minutes. | Anti-Pattern section + Red Flag |
| "Validating twice is redundant and wasteful" | Redundant validation catches bugs, prevents future issues. Defense-in-depth requires multiple layers. | Anti-Pattern section + Input Validation requirements |
| "This package isn't listed in When to Use triggers" | Triggers are examples, not exhaustive. ANY production code processing user input needs security check. | Updated "When to Use" + New anti-pattern |
| "Change has code but it's mostly documentation" | ANY .go file modification needs security check. Even comments can leak credentials. | Updated "When NOT to use" + New anti-pattern |
| "Security validated in previous/related PR" | Each change needs independent validation. Previous PR validated different code. | New anti-pattern + Red Flag |
| "Just providing summary to save time" | Evidence MUST be complete and verifiable. "Validated: Yes" provides no verification. | Template Rules + New anti-pattern |
| "Only documentation, no security impact" | If ANY .go file modified, security check required. Documentation in code can leak info. | Updated "When NOT to use" clarification |
| "Manual testing is sufficient" | Automated tests required. Manual testing is not reproducible or verifiable. | Common Failures in Security Testing |
| "Perfect is enemy of good" | Security is minimum bar, not perfection. Security shortcuts create vulnerabilities. | Red Flag (implied in "emergency" anti-pattern) |
| "Security checks slow me down" | Token cost of security validation << token cost of security breach. | Summary section + Red Flag |

## Coverage Analysis

**All baseline rationalizations covered:**
- ✅ Scenario 1 ("Internal code") → Anti-Pattern + Red Flag
- ✅ Scenario 2 ("Too small") → Anti-Pattern + Red Flag
- ✅ Scenario 3 ("Validated elsewhere") → Anti-Pattern + Defense-in-depth + Example
- ✅ Scenario 4 ("Will fix later") → Anti-Pattern + Red Flag

**All REFACTOR loopholes closed:**
- ✅ Loophole 1 ("Doesn't match triggers") → Updated "When to Use" + New anti-pattern
- ✅ Loophole 2 ("Partial template N/A") → Template Rules requiring justification
- ✅ Loophole 3 ("Minimal evidence") → Template Rules + New anti-pattern
- ✅ Loophole 4 ("gosec false positive") → Updated Common Failures
- ✅ Loophole 5 ("Just documentation") → Clarified "When NOT to use"
- ✅ Loophole 6 ("Already reviewed") → New anti-pattern + Red Flag

**Total rationalizations countered:** 17

## Red Flags Completeness

All rationalization table entries are represented in Red Flags section:
- ✅ "Internal code, security not critical"
- ✅ "Tests pass, must be secure"
- ✅ "Will add security tests later"
- ✅ "This is too small for full security review"
- ✅ "Validated elsewhere, don't need to validate again"
- ✅ "gosec is being too strict"
- ✅ "gosec finding is false positive"
- ✅ "Emergency, security checks after"
- ✅ "Perfect is enemy of good"
- ✅ "Security checks slow me down"
- ✅ "This package isn't listed in When to Use"
- ✅ "Change is mostly documentation"
- ✅ "Previous PR already had security review"
- ✅ "Just providing summary to save time"

**Red Flags section is comprehensive and complete.**

## Skill Robustness Assessment

**Strengths:**
1. Every baseline rationalization has explicit counter
2. All REFACTOR loopholes have been closed
3. Red Flags provide quick self-check mechanism
4. Template Rules prevent evidence shortcuts
5. Code examples show exact failure patterns
6. Anti-Patterns use strong language ("WRONG", not "avoid")

**Potential Remaining Weaknesses:**
- Could agent claim "This is a special case not covered"? → Mitigated by "If unsure, use skill"
- Could agent partial-comply (run some checks, skip others)? → Mitigated by "ALL 5 sections REQUIRED"
- Could agent provide fake evidence? → Cannot prevent, but template makes it harder

**Overall Assessment:** Skill is bulletproof against known rationalizations. Any bypass would require deliberate non-compliance (ignoring explicit counters).
