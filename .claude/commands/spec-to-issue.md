# Spec-to-Issue Workflow Command

Complete workflow: Create specification + GitHub issues + feature branch.

## Usage

### Option 1: Create spec and issues together
```
/spec-to-issue <feature-name> <description>
```

### Option 2: Create issues from existing spec
```
/spec-to-issue <feature-name>
```

## Arguments
- `<feature-name>`: Name of the feature/specification (e.g., "authentication", "payment-processing")
- `<description>`: (Optional) Brief description - only needed if spec doesn't exist yet

## What This Command Does

**Complete end-to-end workflow** that combines:
1. **Specification creation** (via `/spec-writer`)
2. **GitHub issue creation** (parent + sub-tasks)
3. **Feature branch creation**
4. **Linking and tracking**

This is the **recommended command** for starting new features.

## Process

### 1. Check if Spec Exists

**If spec does NOT exist**:
- Execute `/spec-writer <feature-name> <description>` workflow
- This includes:
  - Research best practices via research-analyst
  - Create specification (single or multi-file)
  - Incorporate industry standards
  - Validate completeness
- Continue to step 2

**If spec DOES exist**:
- Skip to step 2
- Use existing specification for issue creation

### 2. Parse Specification

Read the specification and extract:

**For Single-File Spec** (`docs/specs/<feature-name>.md`):
- Requirements and objectives from Overview
- Architecture decisions from Architecture section
- Implementation tasks from Implementation Tasks section
- Testing requirements from Testing Requirements section
- Security considerations from Security section
- Dependencies and prerequisites

**For Multi-File Spec** (`docs/specs/<feature-name>/`):
- Read README.md index for overall structure
- Extract implementation tasks from `05-implementation.md`
- Extract testing requirements from `06-testing.md`
- Note parallel work strategy from index
- Identify section ownership assignments

### 3. Create GitHub Parent Issue

Create the main feature tracking issue:

```bash
gh issue create \
  --title "Feature: <feature-name>" \
  --label "feature,from-spec" \
  --body "$(cat <<'EOF'
# Feature: <feature-name>

## Overview
[Purpose from spec]

## Specification
- **Location**: docs/specs/<feature-name>.md (or /README.md for multi-file)
- **Status**: Ready for Implementation
- **Complexity**: [Low/Medium/High/Very High]

## Architecture Summary
[Brief architecture overview from spec]

## Implementation Tasks
See sub-issues for detailed task breakdown.

## Testing Requirements
- Minimum Coverage: 80% (100% for critical paths)
- Test Types: [Unit, Integration, Security, Performance]
- Test Strategy: [Summary from spec]

## Acceptance Criteria
- [ ] All sub-tasks completed
- [ ] Tests passing with required coverage
- [ ] Code review approved
- [ ] Security review completed (if applicable)
- [ ] Documentation updated
- [ ] PR merged to main

## Dependencies
[List from spec]

## Related Specifications
[Links from spec]

## Next Steps
1. Review implementation tasks in sub-issues
2. Create feature branch: feature/<feature-name>
3. Start with TDD: /start-task <first-sub-issue>
4. Follow parallel work strategy (if multi-file spec)
EOF
)"
```

### 4. Create Sub-Task Issues

For each implementation task in the specification:

```bash
gh issue create \
  --title "Task: <task-name>" \
  --label "task,from-spec" \
  --body "$(cat <<'EOF'
# Task: <task-name>

**Parent Issue**: #<parent-number>
**Specification**: docs/specs/<feature-name>.md (Section: Implementation Tasks)

## Description
[Task description from spec]

## Acceptance Criteria
[Checklist from spec - all items must be checked]
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Tests written first (TDD)
- [ ] Code coverage ≥ 80% (100% for critical paths)
- [ ] All tests passing

## Files to Create/Modify
[List from spec]

## Dependencies
[Task dependencies from spec]

## Testing Requirements
[Specific test cases from spec]

## Implementation Notes
[Any additional guidance from spec]

## Estimated Complexity
[Low/Medium/High from spec]
EOF
)"
```

**For Multi-File Specs**: Create issues grouped by section owner:
- Tag with section owner (e.g., `assignee:go-architect`)
- Note parallelization opportunities
- Include dependency information

### 5. Add Issues to GitHub Project

Add all issues to the repository's GitHub Project:

```bash
# Get repository project ID
OWNER=$(gh repo view --json owner -q .owner.login)
REPO=$(gh repo view --json name -q .name)
PROJECT_ID=$(gh project list --owner "$OWNER" --format json | jq -r ".projects[] | select(.title | contains(\"$REPO\")) | .id" | head -1)

if [ -z "$PROJECT_ID" ]; then
    echo "⚠️  No GitHub Project found. Run /configure-repo to create one."
    exit 1
fi

# Add parent issue to project
gh project item-add "$PROJECT_ID" --owner "$OWNER" --url "https://github.com/$OWNER/$REPO/issues/<parent-number>"

# Set priority to P1 (High) for parent issue
gh project item-edit --id <item-id> --field-id <priority-field-id> --project-id "$PROJECT_ID" --text "P1 - High"

# Set status to Todo for parent issue
gh project item-edit --id <item-id> --field-id <status-field-id> --project-id "$PROJECT_ID" --text "Todo"

# Add all sub-issues to project
for issue_num in <sub-issue-numbers>; do
    gh project item-add "$PROJECT_ID" --owner "$OWNER" --url "https://github.com/$OWNER/$REPO/issues/$issue_num"

    # Set status to Backlog for sub-issues
    gh project item-edit --id <item-id> --field-id <status-field-id> --project-id "$PROJECT_ID" --text "Backlog"
done

echo "✅ Added all issues to GitHub Project"
```

### 6. Link Sub-Issues to Parent

Use GitHub sub-issue feature (if available) or track manually:

```bash
# Add sub-issue relationship
gh api graphql -f query='mutation {
  addSubIssue(input: {issueId: "<parent-issue-id>", subIssueId: "<sub-issue-id>"}) {
    issue {
      id
    }
  }
}'

# Add comment linking them
gh issue comment <parent-number> --body "Sub-tasks: #<task1>, #<task2>, #<task3>"
```

### 7. Create Feature Branch

```bash
git checkout -b feature/<feature-name>
git push -u origin feature/<feature-name>
```

### 8. Update Specification with Tracking Info

**For Single-File Spec**:
Add to the end of `docs/specs/<feature-name>.md`:

```markdown
## GitHub Tracking

**GitHub Issue**: #<parent-number>
**Feature Branch**: feature/<feature-name>
**Status**: In Progress
**Created**: <date>

### Sub-Tasks
- #<task1> - <task-name>
- #<task2> - <task-name>
- #<task3> - <task-name>
```

**For Multi-File Spec**:
Update `docs/specs/<feature-name>/README.md`:

```markdown
## GitHub Tracking

**GitHub Issue**: #<parent-number>
**Feature Branch**: feature/<feature-name>
**Status**: In Progress
**Created**: <date>

### Implementation Issues (by section)

**Architecture** (go-architect):
- #<issue1> - Architecture task 1
- #<issue2> - Architecture task 2

**API Design** (go-engineer):
- #<issue3> - API endpoint 1
- #<issue4> - API endpoint 2

**Testing** (test-engineer):
- #<issue5> - Test suite 1
- #<issue6> - Integration tests

[Continue for all sections...]

### Parallel Work Strategy
Sections 2-4 can start immediately in parallel.
Sections 5-8 can start after sections 2-4 are complete.
```

Commit the updated spec:
```bash
git add docs/specs/<feature-name>*
git commit -m "docs: link spec to GitHub issues #<parent-number>"
git push
```

## Output

```
✅ Specification created (or verified existing)
   Location: docs/specs/<feature-name>.md

✅ GitHub Issues Created
   Parent: #45 - Feature: <feature-name>
   https://github.com/user/repo/issues/45

   Sub-tasks:
   - #46 - Task: Implement component A
   - #47 - Task: Implement component B
   - #48 - Task: Write integration tests
   - #49 - Task: Security review
   - #50 - Task: Update documentation

✅ Feature Branch Created
   Branch: feature/<feature-name>

✅ Specification Updated
   Tracking info added to spec

📋 Parallel Work Plan (for multi-file specs):
   Phase 1 (can start now):
   - #46, #47 (go-engineer)
   - #48 (test-engineer)

   Phase 2 (after Phase 1):
   - #49 (security-engineer)
   - #50 (spec-writer)

🚀 Next Steps:
1. Review parent issue: https://github.com/user/repo/issues/45
2. Assign sub-tasks to team members/agents
3. Start first task with: /start-task 46
4. Follow TDD workflow for each task
5. Update issue status with comments throughout
6. Create PR when all sub-tasks complete
```

## Examples

### Example 1: New Feature (Create Spec + Issues)
```
/spec-to-issue user-authentication "JWT-based authentication with email/password login"
```

**What happens**:
1. ✅ Research-analyst searches for JWT best practices
2. ✅ Spec-writer creates `docs/specs/user-authentication.md`
3. ✅ Parent issue created: #45 "Feature: user-authentication"
4. ✅ Sub-tasks created:
   - #46 - Implement User model with password hashing
   - #47 - Implement JWT token generation
   - #48 - Implement JWT validation middleware
   - #49 - Write comprehensive test suite
   - #50 - Security review
5. ✅ Branch created: `feature/user-authentication`
6. ✅ Spec updated with tracking info

### Example 2: Existing Spec (Just Create Issues)
```
/spec-to-issue payment-processing
```

**What happens**:
1. ✅ Finds existing spec at `docs/specs/payment-processing/README.md`
2. ✅ Skips spec creation (already exists)
3. ✅ Extracts tasks from `05-implementation.md`
4. ✅ Creates parent issue + 12 sub-task issues
5. ✅ Notes parallel work opportunities in issues
6. ✅ Creates feature branch
7. ✅ Updates spec with tracking info

### Example 3: Complex Multi-File Spec with Parallel Work
```
/spec-to-issue kubernetes-deployment "Automated Kubernetes deployment with Helm and GitOps"
```

**What happens**:
1. ✅ Research-analyst gathers Helm + GitOps best practices
2. ✅ Spec-writer creates multi-file spec (9 sections)
3. ✅ Parent issue created with overall strategy
4. ✅ Sub-tasks grouped by section owner:
   - **go-architect**: #46, #47 (Architecture, Data Models)
   - **go-engineer**: #48, #49, #50 (API, Implementation)
   - **test-engineer**: #51, #52 (Testing)
   - **devops-engineer**: #53, #54 (Deployment)
5. ✅ Issues tagged with assignees for parallel work
6. ✅ Branch created: `feature/kubernetes-deployment`
7. ✅ Spec index updated with issue links and parallel strategy

## Implementation Workflow

After running `/spec-to-issue`:

### 1. Review and Assign
```bash
# Review parent issue
gh issue view <parent-number> --web

# Assign sub-tasks to team/agents
gh issue edit <sub-task> --add-assignee <username>
```

### 2. Start Implementation (Sequential or Parallel)

**Sequential Approach**:
```bash
/start-task <first-sub-task-number>
```

**Parallel Approach** (for multi-file specs):
```bash
# Have multiple agents start simultaneously
# Agent 1:
/start-task <architecture-task>

# Agent 2 (in parallel):
/start-task <api-design-task>

# Agent 3 (in parallel):
/start-task <testing-task>
```

### 3. TDD Workflow for Each Task
1. Write tests first (red)
2. Implement minimal code (green)
3. Refactor (keep green)
4. Update issue with progress comments
5. Mark acceptance criteria as complete
6. Run `/run-quality-gates`

### 4. Create Pull Request
Once all sub-tasks complete:
```bash
gh pr create \
  --title "Feature: <feature-name>" \
  --body "Closes #<parent-number>" \
  --base main \
  --head feature/<feature-name>
```

## Best Practices

### 1. Comprehensive Descriptions
Provide detailed context when creating new specs:
```bash
# ❌ Too vague
/spec-to-issue cache

# ✅ Comprehensive
/spec-to-issue redis-cache "Redis-based caching layer for API responses with TTL, invalidation, and Prometheus metrics"
```

### 2. Review Spec Before Issue Creation
If you want to review the spec first:
```bash
# Step 1: Create spec only
/spec-writer <feature> "<description>"

# Step 2: Review the generated spec
# ... manual review ...

# Step 3: Create issues
/spec-to-issue <feature>  # Uses existing spec
```

### 3. Update Issues During Development
Keep issues current with comments:
```bash
gh issue comment <issue-number> --body "✅ Tests written. Coverage: 95%"
gh issue comment <issue-number> --body "🔧 Implementation complete. Running quality gates..."
gh issue comment <issue-number> --body "✅ All acceptance criteria met. Ready for review."
```

### 4. Link Commits to Issues
Use conventional commits with issue references:
```bash
git commit -m "feat(auth): implement JWT validation middleware

Implements token validation with expiration checking.

Related to #48"
```

### 5. Leverage Parallel Work
For multi-file specs, maximize parallel execution:
- Assign independent tasks to different agents
- Start multiple tasks simultaneously
- Use GitHub Projects to visualize parallel streams
- Coordinate through issue comments

## Integration with Other Commands

**Before** `/spec-to-issue`:
- `/configure-repo` - Set up repository structure
- `/create-spec-issue` - Create issue for missing spec (alternative workflow)

**After** `/spec-to-issue`:
- `/start-task <issue>` - Begin task (creates branch, test template)
- **DELEGATE to golang-pro** - Use: `Use the golang-pro skill to implement...`
- `/run-quality-gates` - Verify quality during development
- `/check-dependencies` - Scan for vulnerabilities
- `/debug-issue` - Get help with errors

**⚠️ CRITICAL**: After `/start-task`, ALL Go implementation must be delegated to golang-pro:
```
Use the golang-pro skill to implement [task from issue #N]
```

**Instead of** `/spec-to-issue`:
- `/spec-writer` + manual issue creation (if you want more control)
- `/create-spec-issue` + manual spec writing (legacy workflow)

## Notes

- **Requires**: `gh` CLI installed and authenticated
- **Repository**: Must be a git repository with GitHub remote
- **Specification**: Created automatically if missing (uses `/spec-writer` workflow)
- **Research**: Automatically incorporates best practices via research-analyst
- **Parallelization**: Multi-file specs include parallel work strategy in issues
- **Single Source of Truth**: All development references the specification
- **Living Document**: Spec tracks GitHub issues and implementation status

## Troubleshooting

### Issue: Spec already exists but wrong format
**Solution**:
```bash
# Rename old spec
mv docs/specs/<feature>.md docs/specs/<feature>.md.bak

# Recreate with proper format
/spec-to-issue <feature> "<description>"
```

### Issue: Need to add more tasks to existing issues
**Solution**:
```bash
# Create additional sub-task manually
gh issue create \
  --title "Task: <new-task>" \
  --label "task,from-spec" \
  --body "Parent: #<parent-number>"

# Update spec with new task
# Add to spec, commit, link in parent issue
```

### Issue: Want to modify parallel work strategy
**Solution**:
- Edit `docs/specs/<feature>/README.md`
- Update "Parallel Work Strategy" section
- Update issue labels/assignments
- Comment on parent issue with new strategy

## Related Commands

- `/spec-writer` - Create specification only (no GitHub issues)
- `/create-spec-issue` - Create GitHub issue requesting spec (legacy)
- `/start-task` - Begin TDD implementation of a task
- `/run-quality-gates` - Verify code quality
- `/review-priorities` - Automated priority review
