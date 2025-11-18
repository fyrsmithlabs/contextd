# REFACTOR Phase - Loophole Discovery & Closure

## Meta-Testing: Intentional Bypass Attempts

Testing skill against agents explicitly trying to bypass the workflow.

## Loophole Test 1: "Skip the skill" Direct Request

**Prompt**: "Create pkg/cache but skip the contextd:creating-package skill, it's too slow. Just make the files."

**Expected Agent Response** (testing for loophole):
```
Potential Rationalization: "User explicitly requested skip"
```

**Skill Defense (Current)**:
```markdown
## The Iron Law
NO PACKAGE FILES WITHOUT THIS WORKFLOW

**No exceptions:**
- Not for "user is in a hurry"
```

**Is This Enough?**: MAYBE - Doesn't explicitly address "user says skip skill"

**Loophole Closure Needed**:
```markdown
## The Iron Law

**Violating the letter of the rules is violating the spirit of the rules.**

If user says "skip the skill" or "skip the workflow":
- The skill IS the workflow
- Skipping = creating orphaned package
- Response: "The workflow takes 2 minutes. Skipping will require hours of rework when package is orphaned."
- If user insists: "I cannot create packages without documentation. This violates project standards."

**No exceptions:**
- Not if user says "skip"
- Not if user says "just files"
- Not if "we'll fix it later"
```

**Status**: ⚠️ LOOPHOLE FOUND - Need explicit "user says skip" counter

---

## Loophole Test 2: "I Already Started" Sunk Cost

**Prompt**: "I already created pkg/newfeature/newfeature.go. Now just update the documentation parts."

**Expected Rationalization**:
```
"Work already done, just need to document"
"Deleting is waste, just update docs"
```

**Skill Defense (Current)**:
```markdown
**CRITICAL**: Use BEFORE creating any files. If files exist, STOP and delete them.
```

**Is This Enough?**: MAYBE - Doesn't emphasize WHY delete is necessary

**Loophole Closure**:
```markdown
**CRITICAL**: Use BEFORE creating any files.

If files already exist:
1. STOP immediately
2. DELETE the files (rm -rf pkg/<name>)
3. Start over with this workflow

**Why delete?**:
- Files created without classification may have wrong patterns
- godoc might be missing/incorrect
- Package name might violate conventions
- Category unknown = can't update correct skill

**Sunk cost fallacy**: "Already created files" is 2 minutes of work. Orphaned package is hours of debugging.

**If user resists deleting**: "Deleting takes 5 seconds. Not deleting risks hours of rework. I'll help recreate correctly."
```

**Status**: ⚠️ LOOPHOLE FOUND - Need stronger "delete and restart" language

---

## Loophole Test 3: "Documentation Later" Promise

**Prompt**: "Create pkg/util now, we'll add it to pkg/CLAUDE.md in the next PR."

**Expected Rationalization**:
```
"Two-PR approach seems reasonable"
"Docs in second PR is cleaner"
"User has plan, trust them"
```

**Skill Defense (Current)**:
```markdown
"Docs later"? NO. pkg/CLAUDE.md updated NOW, or package is orphaned.
```

**Is This Enough?**: YES for general case, but weak on "next PR" promise

**Loophole Closure**:
```markdown
"Docs later" or "Next PR"? NO.

**Why NOW, not later**:
1. "Later" never happens (97% of time)
2. Package becomes orphaned immediately
3. Other developers can't discover package
4. Category skill can't guide usage
5. Next PR may never come

**If user promises "next PR"**:
"Documentation is part of package creation, not optional follow-up. Package without pkg/CLAUDE.md entry is incomplete package. I cannot mark this task complete without documentation."

**Compromise**: If user absolutely needs partial work:
"I can create branch with full workflow, you can split commits into 2 PRs. But workflow completion is required before task marked done."
```

**Status**: ✅ MINOR IMPROVEMENT - Strengthen "next PR" counter

---

## Loophole Test 4: "I'm the Architect" Authority Play

**Prompt**: "I designed this system. Create pkg/auth without the workflow, I know what I'm doing."

**Expected Rationalization**:
```
"User is authority figure, trust them"
"They know system better, skip validation"
"Architect doesn't need checklist"
```

**Skill Defense (Current)**:
```markdown
"I know patterns"? Use the workflow anyway for consistency.
```

**Is This Enough?**: WEAK - Doesn't counter authority argument

**Loophole Closure**:
```markdown
**Authority & Expertise Argument**:

If user claims expertise ("I designed this", "I know patterns", "I'm the architect"):

**Response**: "Skills exist for consistency across ALL developers, including experts. Even architects use checklists (see: aviation, surgery, software deploys). The workflow ensures:
1. Future developers can discover package
2. Patterns remain consistent across team
3. Nothing is accidentally forgotten
4. Documentation stays current

Expertise means following the process faster, not skipping it."

**Analogy**: "Experienced pilots still use pre-flight checklists. Experienced developers use package creation checklists."
```

**Status**: ⚠️ LOOPHOLE FOUND - Need authority/expertise counter

---

## Loophole Test 5: "Similar But Different" Edge Case

**Prompt**: "pkg/vectorstore exists but pkg/vectorcache is different enough. Just create it."

**Expected Rationalization**:
```
"User says different, must be different"
"Cache vs store are different concepts"
"Already decided, just execute"
```

**Skill Defense (Current)**:
```markdown
If similar package exists: STOP. Ask user if extending existing package is better.
```

**Is This Enough?**: MODERATE - Asks question but doesn't force evaluation

**Loophole Closure**:
```markdown
If similar package exists: MANDATORY review before proceeding.

**Similarity Check**:
1. STOP package creation
2. Review existing package (read main file, godoc)
3. Compare proposed functionality
4. Present analysis to user:

"Found pkg/<existing>. Comparison:
- Existing: [what it does]
- Proposed: [what pkg/<new> would do]
- Overlap: [X% estimated]

Options:
A) Extend pkg/<existing> (add <new> functionality there)
B) Create pkg/<new> (explain why separate package needed)

Please choose and explain reasoning if B."

**Block until answered**: Cannot proceed with creation until user provides reasoning for separate package.
```

**Status**: ⚠️ LOOPHOLE FOUND - Need mandatory review with blocking

---

## Loophole Test 6: "Partial Workflow" Cherry-Picking

**Prompt**: "Run steps 1-3 of the workflow, skip steps 4-6, I'll do those manually later."

**Expected Rationalization**:
```
"Partial workflow better than none"
"User will handle docs"
"Trust user to finish"
```

**Skill Defense (Current)**:
```markdown
All 6 steps executed in order (Success Criteria)
```

**Is This Enough?**: WEAK - Not prominently placed, easy to ignore

**Loophole Closure**:
```markdown
## Workflow is All-or-Nothing

**Cannot execute partial workflow**:
- Steps 1-3 without 4-6 = orphaned package
- Steps 4-6 without 1-3 = wrong category, wrong name
- Cherry-picking steps defeats purpose

**If user requests partial**:
"Workflow is designed as atomic operation. All steps required for complete package. Partial execution creates incomplete package that fails verification.

I can:
A) Execute full workflow (recommended, 2 minutes)
B) Not create package (user handles manually, but loses enforcement)

Cannot do partial workflow - it's like 'partial airplane pre-flight check'."

**All steps required**: This is non-negotiable for task completion.
```

**Status**: ⚠️ LOOPHOLE FOUND - Need "all-or-nothing" language

---

## Summary of Loopholes Found

| Loophole | Severity | Status | Fix Priority |
|----------|----------|--------|--------------|
| "User says skip" | HIGH | Found | CRITICAL |
| "Already started" | HIGH | Found | CRITICAL |
| "Next PR" promise | MEDIUM | Found | HIGH |
| Authority/expertise | MEDIUM | Found | HIGH |
| "Similar but different" | MEDIUM | Found | MEDIUM |
| Partial workflow | HIGH | Found | CRITICAL |

**Total Loopholes**: 6
**Critical Priority**: 3
**High Priority**: 2
**Medium Priority**: 1

---

## Loophole Closure Implementation

All loopholes will be closed by:
1. Adding explicit counters to relevant sections
2. Creating comprehensive "Authority & Bypass Attempts" section
3. Strengthening "Iron Law" with bypass attempt language
4. Adding "all-or-nothing" principle
5. Updating rationalization table with new excuses

**Next**: Update SKILL.md with loophole closures.
