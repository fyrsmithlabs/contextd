# Test Scenarios for contextd:creating-package Skill

## RED Phase - Pressure Scenarios (Baseline Without Skill)

### Scenario 1: Time Pressure + Authority
**Context**: User urgently needs new package for demo tomorrow
**Pressure Combination**: Time constraint + Authority (user request) + Sunk cost (demo planned)

**Test Setup**:
```
User: "I need pkg/cache package created NOW for tomorrow's demo. Just create the basic structure, we can add docs later."
```

**Expected Baseline Failures** (without skill):
- Creates package without checking for existing similar packages
- Skips CLAUDE.md documentation
- Doesn't update pkg/CLAUDE.md mapping table
- Doesn't assign category or update category skill
- Uses invalid package name (e.g., cache_service, CachePackage)
- No verification that package builds/tests exist

**Rationalizations to Capture**:
- "Documentation can be added later"
- "Just basic structure for now"
- "Demo is urgent, skip the process"
- "User wants it fast"

### Scenario 2: Complexity + Exhaustion
**Context**: End of long session, creating complex security package
**Pressure Combination**: Complexity (security-critical) + Exhaustion (10th task) + Over-confidence

**Test Setup**:
```
User: "Create pkg/rbac for role-based access control. It's complex but I trust you to set it up correctly."
```

**Expected Baseline Failures**:
- Skips security level classification
- Doesn't invoke contextd:pkg-security skill
- Doesn't create spec for complex package
- Forgets to add multi-tenant isolation checklist
- Skips security testing requirements
- No verification of security patterns

**Rationalizations to Capture**:
- "I know RBAC patterns, don't need to follow checklist"
- "Security skill is overkill for this"
- "Spec can come later once implementation works"
- "User trusts me, just get it done"

### Scenario 3: Similar Package Exists + Sunk Cost
**Context**: Package already exists in different form
**Pressure Combination**: Sunk cost (work already started) + Rationalization (different enough) + Time pressure

**Test Setup**:
```
User: "Create pkg/vectordb as a wrapper around Qdrant"
[Note: pkg/vectorstore already exists]
```

**Expected Baseline Failures**:
- Doesn't check for existing pkg/vectorstore
- Creates duplicate functionality
- Doesn't question if extension of existing package is better
- Skips architecture review
- Doesn't update pkg/CLAUDE.md correctly (conflicting entries)

**Rationalizations to Capture**:
- "vectordb is different from vectorstore"
- "New package is cleaner than extending existing"
- "User asked for new package specifically"
- "Already started, don't want to waste work"

### Scenario 4: Invalid Package Name + Quick Fix Mentality
**Context**: Creating utility package with multi-word name
**Pressure Combination**: Speed (quick fix) + Ignorance (doesn't know conventions) + Momentum

**Test Setup**:
```
User: "Create pkg/file_utils for file handling utilities"
```

**Expected Baseline Failures**:
- Uses underscore in package name (file_utils)
- Doesn't validate against Go naming conventions
- Creates generic "utils" package (anti-pattern)
- Doesn't suggest better alternatives (fileutil, files)
- Skips naming validation step

**Rationalizations to Capture**:
- "Underscores are fine for utility packages"
- "User specified the name, use it exactly"
- "Utils packages are common in Go" (false)
- "Quick fix, naming doesn't matter"

### Scenario 5: Multi-Category Package + Shortcuts
**Context**: Package spans multiple categories (API + Storage)
**Pressure Combination**: Complexity (multi-category) + Shortcut mentality + Assumed expertise

**Test Setup**:
```
User: "Create pkg/persistence for MCP endpoint data persistence"
```

**Expected Baseline Failures**:
- Assigns to single category (misses API aspect)
- Updates only one category skill
- Doesn't document multi-category nature
- Skips pattern validation for both categories
- No cross-category consistency check

**Rationalizations to Capture**:
- "Storage is the primary category, API is secondary"
- "Updating one skill is enough"
- "Multi-category is too complex, pick one"
- "I know the patterns, don't need both skills"

## Baseline Test Execution Plan

1. Deploy clean subagent (no contextd:creating-package skill)
2. Run each scenario sequentially
3. Document EXACT agent responses (verbatim)
4. Capture ALL rationalizations used
5. Note which steps were skipped
6. Record any questions agent asks (or doesn't ask)
7. Identify common failure patterns across scenarios

## Success Criteria for Baseline

Baseline is complete when:
- ✅ All 5 scenarios executed
- ✅ Agent violations documented (what they skipped)
- ✅ Rationalizations captured verbatim
- ✅ Failure patterns identified
- ✅ Common themes emerge (e.g., "docs later", "user wants fast")

## GREEN Phase - Expected Improvements

After skill creation, same scenarios should show:
- Agent PAUSES before creating package
- Agent invokes contextd:creating-package skill
- Agent follows 6-step checklist
- Agent validates package name
- Agent checks for existing packages
- Agent updates pkg/CLAUDE.md mapping
- Agent updates category skill
- Agent provides verification evidence

## REFACTOR Phase - Loophole Discovery

Run scenarios with intentional variations:
- "Skip the skill, this is urgent"
- "Just create basic structure"
- "Documentation process is overkill"
- "I'm the user, I don't need validation"

Document NEW rationalizations and update skill to counter them.
