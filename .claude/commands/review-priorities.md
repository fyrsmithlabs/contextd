# Review and Update Priorities

**Command**: `/review-priorities [timeframe]`

**Description**: Have the product-manager agent review completed work, evaluate priorities, and update project status.

**Usage**:
```
/review-priorities          # Review last 24 hours (default)
/review-priorities 7d       # Review last 7 days
/review-priorities manual   # Manual review with user input
```

## Purpose

This command triggers the product-manager agent to:
1. Review completed work in the specified timeframe
2. Analyze open issues and PRs
3. Evaluate current priorities
4. Update GitHub Projects priorities
5. Create new issues for emerging needs
6. Archive completed items

## Automation

This command can be run:
- **Manually** by users when needed
- **Scheduled** via GitHub Actions cron (e.g., daily at 1 AM)
- **On-demand** after major milestones

## Agent Workflow

When this command is invoked, delegate to the product-manager agent:

```
Have the product-manager agent review priorities and update projects:

Timeframe: [timeframe or "last 24 hours"]

Tasks:
1. Review completed issues/PRs in timeframe
2. Analyze current open issues and their status
3. Evaluate priority alignment with goals
4. Identify gaps or new feature needs
5. Update GitHub Projects (priorities, status, milestones)
6. Create new issues if needed
7. Provide summary report
```

## What the Product Manager Reviews

### Completed Work Analysis
- Issues closed in timeframe
- PRs merged in timeframe
- Feature completions
- Bug fixes resolved
- Technical debt addressed

### Current State Assessment
- Open issues by priority
- Blocked issues and blockers
- In-progress work status
- PR review queue
- Upcoming milestones

### Priority Evaluation
- High priority items on track?
- New high-priority needs?
- Low priority items to deprioritize?
- Feature requests to prioritize?
- Technical debt to address?

### Project Updates
- Update issue priorities (labels: priority/critical, priority/high, priority/medium, priority/low)
- Update project board columns
- Assign/reassign issues
- Update milestones
- Archive completed items

## Expected Output

The product-manager agent will provide a summary report:

```markdown
## Priority Review Summary
**Timeframe**: Last 24 hours
**Date**: YYYY-MM-DD HH:MM

### Completed Work
- ✅ 3 issues closed
- ✅ 2 PRs merged
- ✅ Feature: User authentication completed
- ✅ Bug: Race condition in cache fixed

### Current Status
- 🔴 5 high priority issues open
- 🟡 8 medium priority issues open
- 🟢 12 low priority issues open
- ⏸️ 2 blocked issues

### Priority Changes
- ↗️ Upgraded: Issue #45 to high priority (security concern)
- ↘️ Downgraded: Issue #23 to low priority (can defer)
- 🆕 Created: Issue #78 - Implement rate limiting (high priority)

### Action Items
- [ ] Address blocked issue #34 (waiting on external API)
- [ ] Review stale PR #56 (no activity in 7 days)
- [ ] Create spec for rate limiting feature

### Recommendations
- Focus next: Complete high priority security items
- Consider: Scheduling technical debt sprint
- Note: On track for Q1 milestone

### Next Review
Scheduled: Tomorrow at 1 AM (automated)
```

## GitHub Actions Integration

Create `.github/workflows/review-priorities.yml`:

```yaml
name: Daily Priority Review

on:
  schedule:
    # Run daily at 1 AM UTC
    - cron: '0 1 * * *'
  workflow_dispatch: # Allow manual trigger

jobs:
  review-priorities:
    runs-on: ubuntu-latest
    steps:
      - name: Review Priorities
        uses: actions/github-script@v7
        with:
          script: |
            const issue = await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: '🤖 Daily Priority Review',
              body: `Automated priority review triggered by scheduled workflow.

              @axyzlabs-bot please run: /review-priorities

              This will analyze work completed in the last 24 hours and update priorities accordingly.`,
              labels: ['automated', 'priority-review', 'product-management']
            });

            console.log('Created priority review issue:', issue.data.number);
```

## Integration with Product Manager Agent

The product-manager agent (from project-management-suite) will:

1. **Gather Data**
   - Query GitHub API for completed work
   - Review open issues and PRs
   - Check project board status
   - Analyze labels and milestones

2. **Analyze Patterns**
   - Velocity (issues completed per day)
   - Bottlenecks (blocked issues, slow PRs)
   - Priority distribution
   - Feature vs. bug ratio

3. **Make Decisions**
   - Adjust priorities based on data
   - Create new issues for gaps
   - Update project board
   - Reassign if needed

4. **Report Findings**
   - Summary of changes
   - Recommendations
   - Action items

## Manual Usage Examples

### Daily Review
```
User: /review-priorities
```
Reviews last 24 hours, updates priorities, provides summary.

### Weekly Review
```
User: /review-priorities 7d
```
Reviews last 7 days for broader perspective.

### Post-Milestone Review
```
User: /review-priorities manual

Product Manager: What timeframe should I review?
User: Review since milestone v1.0.0 release (last 14 days)
Product Manager: [Performs comprehensive review]
```

## Benefits

### Automated (Scheduled)
- ✅ Continuous priority alignment
- ✅ No manual tracking overhead
- ✅ Early issue detection
- ✅ Consistent review cadence

### Manual (On-Demand)
- ✅ Review after major events
- ✅ Ad-hoc priority adjustments
- ✅ Strategic planning sessions
- ✅ Stakeholder updates

## Notes

- **Non-disruptive**: If everything is on track, agent simply confirms status
- **Data-driven**: Decisions based on actual completed work and patterns
- **Transparent**: All changes tracked in issue comments and project history
- **Flexible**: Works with scheduled automation or manual invocation
