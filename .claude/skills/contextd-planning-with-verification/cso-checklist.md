# CSO (Claude Search Optimization) Checklist

## 1. Frontmatter Validation

### Name Field
**Current**: `contextd-planning-with-verification`
**Validation**: ✅ PASS
- Uses only letters, numbers, and hyphens
- No parentheses or special characters
- Descriptive and follows contextd naming convention

### Description Field
**Current**: "Use when creating TodoWrite for major work (features, bugs, refactoring, security, multi-file changes) - automatically adds verification subtasks to prevent unverified completion claims and forgotten evidence requirements"

**Validation**: ✅ PASS
- Starts with "Use when..." (triggers explicitly stated)
- Includes specific situations: "features, bugs, refactoring, security, multi-file changes"
- Explains what it does: "automatically adds verification subtasks"
- Explains why it helps: "prevent unverified completion claims and forgotten evidence requirements"
- Third person
- Under 500 characters (approx 230 characters)
- Total frontmatter well under 1024 character limit

## 2. Keyword Coverage

### Technology-Specific Keywords
✅ TodoWrite (tool name)
✅ completing-major-task (skill reference)
✅ completing-minor-task (skill reference)
✅ contextd:code-review (skill reference)

### Symptom Keywords
✅ "unverified completion" (problem symptom)
✅ "forgotten evidence" (problem symptom)
✅ "verification subtasks" (solution)
✅ "major work" (trigger)

### Problem Keywords
✅ "features, bugs, refactoring, security" (task types)
✅ "multi-file changes" (complexity indicator)
✅ "verification" (core concept, appears 58 times in skill)
✅ "evidence" (core concept)

### Action Keywords
✅ "implement", "fix", "refactor", "add", "update" (trigger verbs)
✅ "verify", "check", "validate" (verification actions)

## 3. Descriptive Naming

**Current name**: `contextd-planning-with-verification`

**Analysis**:
- ✅ Uses gerund form: "planning" (active voice)
- ✅ Describes what you DO: "planning with verification"
- ✅ Project-specific prefix: "contextd-" (indicates project scope)
- ✅ Clear focus: verification aspect of planning

**Alternative names considered**:
- `contextd-verification-planning` - Less clear (verification OF planning vs planning WITH verification)
- `contextd-adding-verification-subtasks` - Too specific to implementation
- `contextd-todo-verification` - Misses planning aspect

**Verdict**: ✅ Current name is optimal

## 4. Token Efficiency

**Target**: <500 words for frequently-loaded skills

**Current word count**:
```bash
wc -w SKILL.md
# 994 words
```

**Analysis**: ⚠️ EXCEEDS target (994 vs 500 target)

**Mitigation**:
- Skill is NOT in "frequently-loaded" category (project-specific, used during planning phase)
- Skill provides comprehensive examples (critical for understanding)
- Rationalization table is essential (prevents loopholes)
- Length justified by complexity of verification enforcement

**Optimization opportunities**:
- Examples are compressed (JSON only, minimal narrative)
- Rationalization table is concise (2 columns, direct counters)
- No redundant content
- No verbose explanations

**Verdict**: ⚠️ ACCEPTABLE for this skill type (not frequently-loaded, justifiable length)

## 5. Cross-Reference Quality

### Referenced Skills
✅ `contextd:completing-major-task` - Clear requirement marker
✅ `contextd:completing-minor-task` - Clear requirement marker
✅ `contextd:code-review` - Clear integration point

### Format
✅ Uses skill name only (no @ links that force-load)
✅ Explicit about when to invoke ("Invoke X")
✅ Explains integration workflow

## 6. Flowchart Usage

**Current flowcharts in skill**: NONE

**Should we add flowcharts?**

**Decision tree for TodoWrite**:
- "When to Use" section → Could benefit from flowchart
- "Task Classification" → Could benefit from flowchart

**Analysis**:
- Current bullet lists are scannable
- Decision criteria are clear (keyword-based)
- Flowchart might add visual clarity

**Verdict**: ⚠️ OPTIONAL - Could add small flowchart for task classification (major vs minor)

**Proposed flowchart** (if added):
```
Is task affecting functionality? → YES → Major task (completing-major-task)
                                → NO → Is cosmetic only? → YES → Minor task (completing-minor-task)
                                                         → NO → Major task (completing-major-task)
```

## 7. Discovery Optimization

### Will Claude find this skill when needed?

**Test query**: "How do I create todos for implementing a feature?"

**Triggers in description**:
✅ "creating TodoWrite" - Exact match
✅ "major work (features...)" - Keyword match
✅ "automatically adds verification subtasks" - Solution match

**Verdict**: ✅ HIGH discoverability

### Will Claude load this skill at the right time?

**Triggering conditions**:
- User requests TodoWrite creation
- Agent about to create TodoWrite for work
- Planning phase of task execution

**Description clarity**: ✅ CLEAR - "Use when creating TodoWrite for major work"

**Verdict**: ✅ Will load at correct time

## 8. Pressure Resistance (From TDD Phase)

### Rationalizations Addressed
✅ All 5 baseline rationalizations countered
✅ 3 additional loopholes from REFACTOR closed
✅ Red flags section for self-checking
✅ "Spirit vs Letter" violation explicitly countered

### Enforcement Strength
✅ 5 enforcement rules (non-negotiable)
✅ Multiple examples showing correct pattern
✅ Rationalization table with direct counters
✅ Quick reference table for scanning

**Verdict**: ✅ HIGH pressure resistance

## 9. Integration Quality

### With Other Skills
✅ completing-major-task integration clear
✅ completing-minor-task integration clear
✅ code-review integration explained
✅ Verification subtask ordering specified

### With TodoWrite Tool
✅ Exact JSON format examples
✅ Before/after comparisons
✅ All required fields shown (content, status, activeForm)

**Verdict**: ✅ EXCELLENT integration clarity

## 10. Overall CSO Score

| Criterion | Score | Notes |
|-----------|-------|-------|
| Frontmatter | ✅ PASS | Name and description optimal |
| Keyword Coverage | ✅ PASS | Comprehensive, searchable |
| Descriptive Naming | ✅ PASS | Clear, active, focused |
| Token Efficiency | ⚠️ ACCEPTABLE | 994 words (justified by complexity) |
| Cross-References | ✅ PASS | No force-loading, clear markers |
| Flowchart Usage | ⚠️ OPTIONAL | Could add task classification flowchart |
| Discovery | ✅ PASS | High discoverability |
| Pressure Resistance | ✅ PASS | Comprehensive rationalization counters |
| Integration | ✅ PASS | Excellent clarity |

**Overall Verdict**: ✅ CSO COMPLIANCE - HIGH QUALITY

**Optional improvements**:
1. Add small flowchart for task classification (major vs minor decision)
2. Consider minor length reduction (target ~800 words if possible)

**Recommendation**: DEPLOY AS-IS (optional improvements can be added in future iterations)
