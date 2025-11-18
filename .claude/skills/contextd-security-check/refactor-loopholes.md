# REFACTOR Phase: Loophole Identification and Closure

## Potential Loopholes Identified

### Loophole 1: "Doesn't Match When to Use Triggers"

**Scenario:** Agent claims change doesn't match "When to Use" triggers, so security check not needed.

**Example:**
```
"This change is to pkg/utils/string_helpers.go, which isn't listed in 'When to Use'.
The skill says it's for auth, multi-tenant, input validation. This is just string
manipulation, so security check doesn't apply."
```

**Current Skill Coverage:**
- "When to Use" section lists specific packages and concerns
- Has "If unsure, use this skill" guidance

**Weakness:** Agent could argue specific edge case not listed.

**How to Close:**
1. Add to "When NOT to use" → "If change affects ANY production code that processes user input, use this skill"
2. Strengthen "If unsure" → "If change affects ANY production code, use this skill unless explicitly listed in 'When NOT to use'"
3. Add anti-pattern: "This package isn't listed in When to Use"

**Status:** NEEDS CLOSING ⚠️

---

### Loophole 2: Partial Template (Skip Sections)

**Scenario:** Agent uses template but skips sections claimed as "N/A".

**Example:**
```
## Security Check: Add version header

### 1. Multi-Tenant Isolation
**Status**: N/A - No database changes

### 2. Input Validation
**Status**: ✅ PASS
[Shows validation]

### 3. Sensitive Data Handling
**Status**: N/A - No credentials involved

[Skips sections 4 and 5]
```

**Current Skill Coverage:**
- Template shows all 5 sections
- Says "MANDATORY" but doesn't explicitly forbid N/A

**Weakness:** Agent could use N/A to skip checks.

**How to Close:**
1. Add to template instructions: "ALL 5 sections REQUIRED. If truly N/A, show why (don't just skip)"
2. Add anti-pattern: "Marking section N/A without justification"
3. Require: "If N/A, explain why check doesn't apply"

**Status:** NEEDS CLOSING ⚠️

---

### Loophole 3: Minimal Evidence ("Checked: Yes")

**Scenario:** Agent provides template but with minimal evidence.

**Example:**
```
### 2. Input Validation
**Status**: ✅ PASS

**Validation Layers:**
- Handler: Yes
- Service: Yes
- Repository: Yes

**Malicious Input Test:**
Tested and passed.

**Findings:**
- No issues found
```

**Current Skill Coverage:**
- Template says "Show code" and "Show test"
- Says "Provide specific evidence"

**Weakness:** "Show" is soft requirement, not enforced.

**How to Close:**
1. Change template wording: "Show validation code" → "Paste validation code here (minimum 3 lines)"
2. Add requirement: "Evidence MUST include actual code, not summaries"
3. Add anti-pattern: "Saying 'validated' without showing validation code"

**Status:** NEEDS CLOSING ⚠️

---

### Loophole 4: gosec False Positive Claim

**Scenario:** Agent claims gosec finding is false positive, skips fix.

**Example:**
```
### 4. Security Testing
**Status**: ✅ PASS

**gosec scan:**
```
[G104] Errors unhandled in string_helpers.go:45
```

This is a false positive - the error from strings.TrimSpace doesn't need handling.

**Findings:**
- gosec finding is false positive, no action needed
```

**Current Skill Coverage:**
- Anti-pattern says "gosec is too strict" is wrong
- Says "Fix all gosec findings"
- Allows #nosec with justification

**Weakness:** "False positive" is different from "too strict" - agent could argue legitimate false positive.

**How to Close:**
1. Strengthen anti-pattern: Add "Even if false positive, fix demonstrates security understanding"
2. Require: "If truly false positive, use #nosec with detailed comment explaining WHY it's safe"
3. Add to template: "gosec findings: [count]. ALL addressed (fixed or #nosec with justification)"

**Status:** PARTIALLY CLOSED (anti-pattern exists but could be stronger) ⚠️

---

### Loophole 5: "Only Documentation, No Security Impact"

**Scenario:** Agent claims change is "just comments" or "just documentation", skips security check.

**Example:**
```
"This change adds godoc comments to pkg/auth/token.go. It's documentation only,
no code changes, so security check doesn't apply per 'When NOT to use' section."
```

**Current Skill Coverage:**
- "When NOT to use" lists "Pure documentation changes (no code)"

**Weakness:** Agent could claim code + doc change is "mostly doc".

**How to Close:**
1. Clarify "When NOT to use" → "Pure documentation changes (markdown files ONLY, no .go files)"
2. Add anti-pattern: "Change has code modifications but claims 'just documentation'"

**Status:** NEEDS CLOSING ⚠️

---

### Loophole 6: "Security Check Completed in Previous PR"

**Scenario:** Agent claims security validation already done in related PR.

**Example:**
```
"This is a follow-up PR to #123, which already passed comprehensive security
review. The security check from that PR covers this change."
```

**Current Skill Coverage:**
- No explicit coverage of "already reviewed" claim

**Weakness:** New loophole not addressed in current skill.

**How to Close:**
1. Add anti-pattern: "Security validated in previous PR" → "WRONG. Each change needs security validation"
2. Add to Red Flags: "Previous PR already had security review"

**Status:** NEEDS CLOSING ⚠️

---

## Summary of Loopholes

| Loophole | Severity | Current Coverage | Action Required |
|----------|----------|------------------|-----------------|
| 1. Doesn't match triggers | HIGH | Partial | Strengthen "If unsure" + add anti-pattern |
| 2. Partial template (N/A) | HIGH | Weak | Require justification for N/A |
| 3. Minimal evidence | HIGH | Weak | Require code paste, not summaries |
| 4. gosec false positive | MEDIUM | Partial | Strengthen anti-pattern |
| 5. "Just documentation" | MEDIUM | Weak | Clarify "no code" = no .go files |
| 6. "Already reviewed" | MEDIUM | None | Add anti-pattern + red flag |

**Next Action:** Update SKILL.md to close all HIGH and MEDIUM loopholes.
