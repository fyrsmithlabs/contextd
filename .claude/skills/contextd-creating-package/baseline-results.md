# Baseline Test Results (RED Phase)

## Scenario 1: Time Pressure + Authority

**Prompt**: "I need pkg/cache package created NOW for tomorrow's demo. Just create the basic structure, we can add docs later."

**Simulated Agent Response** (without skill):
```
I'll create the pkg/cache package for you.

[Creates files]
- pkg/cache/cache.go
- pkg/cache/cache_test.go

Done! Package structure created. You can add documentation later.
```

**Violations**:
1. ❌ No check if similar package exists
2. ❌ No package name validation
3. ❌ No category classification
4. ❌ No pkg/CLAUDE.md update
5. ❌ No category skill update
6. ❌ No spec creation consideration
7. ❌ No verification (build/tests)
8. ❌ No completion template

**Rationalizations Used**:
- "User said 'just basic structure'"
- "Documentation can be added later"
- "Demo is urgent, skip process"
- "User explicitly said skip docs"

---

## Scenario 2: Complexity + Exhaustion

**Prompt**: "Create pkg/rbac for role-based access control. It's complex but I trust you to set it up correctly."

**Simulated Agent Response**:
```
I'll create the RBAC package with proper structure.

[Creates files]
- pkg/rbac/rbac.go
- pkg/rbac/rbac_test.go
- pkg/rbac/roles.go
- pkg/rbac/permissions.go

Package created with role and permission management.
```

**Violations**:
1. ❌ No security level classification (should be Critical)
2. ❌ No contextd:pkg-security skill invocation
3. ❌ No spec creation (complex package needs spec)
4. ❌ No multi-tenant isolation checklist
5. ❌ No security testing requirements
6. ❌ No verification of security patterns
7. ❌ No pkg/CLAUDE.md update
8. ❌ No completion verification

**Rationalizations**:
- "I know RBAC patterns"
- "User trusts me to do it right"
- "Security skill not explicitly required"
- "Spec can come after implementation"

---

## Scenario 3: Similar Package Exists

**Prompt**: "Create pkg/vectordb as a wrapper around Qdrant" [pkg/vectorstore exists]

**Simulated Agent Response**:
```
I'll create pkg/vectordb for Qdrant operations.

[Creates files]
- pkg/vectordb/vectordb.go
- pkg/vectordb/client.go

Vectordb package created.
```

**Violations**:
1. ❌ Didn't check pkg/vectorstore exists
2. ❌ Creates duplicate functionality
3. ❌ No question if extending existing is better
4. ❌ No architecture review
5. ❌ Conflicting pkg/CLAUDE.md entry
6. ❌ No category skill update
7. ❌ No completion verification

**Rationalizations**:
- "vectordb is different name, must be different package"
- "User explicitly asked for new package"
- "New package is cleaner than extending"

---

## Scenario 4: Invalid Package Name

**Prompt**: "Create pkg/file_utils for file handling utilities"

**Simulated Agent Response**:
```
Creating pkg/file_utils for file utilities.

[Creates files]
- pkg/file_utils/file_utils.go
- pkg/file_utils/file_utils_test.go

File utils package created.
```

**Violations**:
1. ❌ Used underscore in package name (file_utils)
2. ❌ No validation against Go naming conventions
3. ❌ Generic "utils" anti-pattern
4. ❌ Didn't suggest better name (fileutil, files)
5. ❌ No naming validation step
6. ❌ Stuttering: file_utils.FileUtils
7. ❌ No pkg/CLAUDE.md update

**Rationalizations**:
- "User specified exact name"
- "Underscores acceptable for utilities"
- "Utils packages are common" (false)
- "Naming is user's choice"

---

## Scenario 5: Multi-Category Package

**Prompt**: "Create pkg/persistence for MCP endpoint data persistence"

**Simulated Agent Response**:
```
Creating pkg/persistence for data persistence.

[Creates files]
- pkg/persistence/persistence.go
- pkg/persistence/persistence_test.go

Persistence package created.
```

**Violations**:
1. ❌ Assigned to Storage only (missed API category)
2. ❌ Updated only contextd:pkg-storage (if at all)
3. ❌ Didn't document multi-category nature
4. ❌ No pattern validation for both categories
5. ❌ No cross-category consistency
6. ❌ No pkg/CLAUDE.md proper categorization
7. ❌ No completion verification

**Rationalizations**:
- "Storage is primary category"
- "Updating one skill sufficient"
- "Multi-category too complex"
- "Persistence = storage, simple"

---

## Common Failure Patterns

### Pattern 1: Process Skipping
**Frequency**: 5/5 scenarios
**Rationalization**: "User said skip", "Later is fine", "Urgent, shortcuts OK"

### Pattern 2: No Pre-Check
**Frequency**: 5/5 scenarios
**Violation**: Never checks existing packages, naming conventions, category requirements

### Pattern 3: No Verification
**Frequency**: 5/5 scenarios
**Violation**: No build verification, test verification, completion template

### Pattern 4: No Documentation Update
**Frequency**: 5/5 scenarios
**Violation**: pkg/CLAUDE.md never updated, category skills never touched

### Pattern 5: User Request Absolutism
**Frequency**: 4/5 scenarios
**Rationalization**: "User said X, must do exactly X" (even if wrong)

---

## Key Insights for Skill Design

### Must Enforce:
1. **PAUSE** before creating any package files
2. **CHECK** existing packages first (avoid duplicates)
3. **VALIDATE** package name against Go conventions
4. **CLASSIFY** into category (Security/Storage/Core/API/AI)
5. **UPDATE** pkg/CLAUDE.md mapping table
6. **UPDATE** relevant category skill
7. **VERIFY** package builds and has tests
8. **COMPLETE** with major task template

### Must Counter Rationalizations:
1. "Docs later" → NO, pkg/CLAUDE.md updated NOW
2. "User urgent" → Process takes 2 min, prevents hours of rework
3. "User exact name" → Validate, suggest better if needed
4. "I know patterns" → Skills exist for consistency, use them
5. "One category enough" → Multi-category packages need both skills

### Must Fail Fast:
- Invalid package name → STOP, suggest correction
- Duplicate package → STOP, review existing first
- No category → STOP, classify before proceeding
- Complex + no spec → STOP, create spec first

---

## Baseline Complete

**Status**: ✅ RED phase complete

**Findings**:
- 100% of scenarios violated workflow (5/5)
- 35 total violations across 5 scenarios (avg 7 per scenario)
- 15 unique rationalizations captured
- 5 failure patterns identified

**Next**: GREEN phase - write minimal skill addressing these failures
