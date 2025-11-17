# Create GitHub Issues from Roadmap

Create GitHub issues from the Qdrant roadmap document.

## Instructions

This command reads `/docs/QDRANT-ROADMAP.md` and creates GitHub issues for each task.

### Prerequisites

1. **GitHub repository exists**: `axyzlabs/contextd`
2. **GitHub CLI installed**: `gh` command available
3. **Authenticated**: `gh auth login` completed

### Execution

```bash
cd /home/dahendel/projects/contextd && \
echo "Creating GitHub issues from Qdrant Roadmap..." && \
echo "" && \
./scripts/create-roadmap-issues.sh && \
echo "" && \
echo "✓ Issues created successfully!" && \
echo "" && \
echo "View issues: gh issue list" && \
echo "Or visit: https://github.com/axyzlabs/contextd/issues"
```

### What Gets Created

**Phase 1 - Foundation (P0 Blockers):**
1. Qdrant Client Implementation
2. Vector Database Abstraction Layer
3. Local Qdrant Deployment

**Phase 2 - Migration (P1 High):**
4. Migration Tool - Data Export
5. Migration Tool - Data Import
6. Migration Tool - Verification
7. Automatic Migration Workflow

**Phase 3 - Sync (P1-P3):**
8. Configuration Management
9. Sync Command - Status
10. Sync Command - Manual Trigger
11. Background Sync Service

**Phase 4 - Documentation (P1-P2):**
12. User Documentation
13. CLI Help & Usability
14. Integration Tests

**Phase 5 - Launch (P0-P2):**
15. Rollout Plan
16. Monitoring & Alerting
17. Performance Optimization

### Labels Applied

Each issue gets labeled with:
- **Priority**: `priority:P0`, `priority:P1`, `priority:P2`, `priority:P3`
- **Phase**: `phase:1-foundation`, `phase:2-migration`, etc.
- **Component**: `component:qdrant`, `component:migration`, `component:sync`
- **Effort**: `effort:1-day`, `effort:2-days`, `effort:3-days`, `effort:5-days`

### Manual Creation

If you prefer to create issues manually:

```bash
# View generated issue templates
ls -la issues/qdrant-*.md

# Create via GitHub CLI
gh issue create --title "..." --body-file issues/qdrant-task-1-1.md --label "priority:P0"
```

### Troubleshooting

**Repository not found?**
- Create the repository first: `gh repo create axyzlabs/contextd --public`
- Or make it accessible

**Authentication failed?**
- Run: `gh auth login`
- Follow the prompts

**Labels don't exist?**
- The script will create labels automatically
- Or create manually in GitHub UI

## Next Steps

After creating issues:
1. Review and adjust priorities
2. Assign to team members
3. Add to project board
4. Start with Phase 1 tasks
