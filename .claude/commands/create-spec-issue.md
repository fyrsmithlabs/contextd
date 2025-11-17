# Create Spec Issue Command

Create a GitHub issue for a missing specification using GitHub MCP tools.

## Usage
```
/create-spec-issue <description>
```

## Process

1. **Check if spec exists**
   - Look for relevant specs in `docs/specs/`
   - If found, notify user and exit

2. **Create GitHub issue using MCP**
   - Use `mcp__github__create_issue` to create the issue
   - Issue will include:
     - Title: "Spec Needed: <feature-name>"
     - Labels: `documentation`, `spec-required`, `needs-prioritization`
     - Body: Comprehensive template with requirements and acceptance criteria
     - Proper formatting and structure

3. **Add to Project Board**
   - Assign issue to appropriate project board
   - Set initial status (e.g., "Backlog" or "To Do")

4. **Product Manager Prioritization**
   - Have product-manager agent analyze the issue
   - Agent will:
     - Assess business value and impact
     - Determine priority level (P0/P1/P2/P3)
     - Suggest milestone assignment
     - Add priority label (e.g., `priority:high`)
     - Comment with prioritization rationale

5. **Notify user**
   - Display created issue number and URL
   - Show prioritization results
   - Suggest next steps

## Example
```
/create-spec-issue authentication system with JWT tokens
```

This creates an issue for the authentication specification with:
- Automatic label assignment
- Product manager prioritization
- Project board placement

## Labels Applied

**Automatic Labels**:
- `documentation` - Spec is documentation
- `spec-required` - Specification needed
- `needs-prioritization` - Awaiting PM review

**Priority Labels** (added by product-manager agent):
- `priority:critical` (P0) - Blocking/Critical
- `priority:high` (P1) - Important/High value
- `priority:medium` (P2) - Standard priority
- `priority:low` (P3) - Nice to have

**Additional Labels** (context-dependent):
- `breaking-change` - If spec involves breaking changes
- `security` - If security-related
- `performance` - If performance-related

## Issue Template Structure

The created issue will follow this structure:

```markdown
## Specification Required

### Package/Feature
<feature-name>

### Description
<detailed description from arguments>

#### Core Requirements
- <extracted requirements>

#### Deliverables
1. **Main Specification** (`docs/specs/<name>/<feature>.md`)
   - Feature overview and architecture
   - Configuration options
   - API contracts
   - Error handling

2. **Implementation Work Spec** (`docs/specs/<name>/implementation-tasks.md`)
   - Task breakdown
   - Dependencies
   - Testing requirements

### Acceptance Criteria
- [ ] Spec follows template from `docs/standards/package-guidelines.md`
- [ ] All requirements documented
- [ ] Implementation tasks defined
- [ ] Saved to `docs/specs/<name>/`

### Research Areas
<context-specific research topics>

### Prioritization
[Product Manager Analysis - Added by agent]

### Next Steps
1. Product manager reviews and prioritizes
2. Spec-writer agent researches and drafts
3. Documentation-writer agent reviews
4. Save to `docs/specs/`
5. Create implementation issues with `/spec-to-issue`
```

## Next Steps After Issue Creation

1. **Wait for Product Manager Prioritization** (automatic)
2. **Spec Writer** - Have spec-writer agent draft the specification
3. **Documentation Review** - Have documentation-writer agent polish
4. **Save Specs** - Save to `docs/specs/<package-name>/`
5. **Update References** - Update CLAUDE.md if needed
6. **Close Issue** - Mark complete when specs saved
7. **Create Tasks** - Use `/spec-to-issue <package-name>` for implementation

## Technical Implementation

Uses GitHub MCP tools exclusively:
- `mcp__github__create_issue` - Create the issue
- `mcp__github__update_issue` - Add labels and project
- `mcp__github__add_issue_comment` - Add PM analysis

Falls back to `gh` CLI only if:
- Project board assignment (not yet in MCP)
- Advanced project operations

## Notes

- **Primary**: GitHub MCP tools
- **Fallback**: gh CLI for unsupported features
- Issue created in current repository (contextd)
- Spec should follow template from `docs/standards/package-guidelines.md`
- Product manager agent runs automatically after issue creation
- Priority labels help with roadmap planning
