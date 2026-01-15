# Version Management

**Single Source of Truth**: The `VERSION` file in the repository root.

## Quick Reference

| File/Location | Purpose | Updated How |
|---------------|---------|-------------|
| `VERSION` | **Source of truth** - all versions derive from this | Manual edit |
| `.claude-plugin/plugin.json` | Claude plugin version | `scripts/sync-version.sh` |
| Git tags (`v*`) | Release markers | `git tag` after sync |

## Versioning Workflow

### 1. Update Version

Edit the `VERSION` file:

```bash
echo "0.4.0" > VERSION
```

### 2. Sync to All Files

Run the sync script:

```bash
./scripts/sync-version.sh
```

This updates:
- `.claude-plugin/plugin.json`
- (Future: package.json, go.mod, etc.)

### 3. Commit Changes

```bash
git add VERSION .claude-plugin/plugin.json
git commit -m "chore: bump version to 0.4.0"
```

### 4. Tag Release

```bash
git tag -a v0.4.0 -m "Release v0.4.0"
git push && git push --tags
```

## Version Format

We follow [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

**Examples**:
- `0.3.0` - Stable release
- `0.3.0-rc1` - Release candidate 1
- `0.3.0-beta.1` - Beta prerelease
- `0.4.0-alpha` - Alpha prerelease

## Pre-Release Workflow

For release candidates and pre-releases:

```bash
# Create RC
echo "0.4.0-rc1" > VERSION
./scripts/sync-version.sh
git add VERSION .claude-plugin/plugin.json
git commit -m "chore: bump version to 0.4.0-rc1"
git tag -a v0.4.0-rc1 -m "Release candidate 1 for v0.4.0"
git push && git push --tags

# Iterate
echo "0.4.0-rc2" > VERSION
./scripts/sync-version.sh
# ... repeat commit/tag/push
```

## Automation

### Temporal-Based Version Validation

Version consistency is automatically enforced via Temporal workflows on all pull requests:

**What it does:**
- Fetches `VERSION` file and `.claude-plugin/plugin.json` from PR
- Compares versions to ensure they match
- Posts helpful comment with fix instructions if mismatch detected
- Comment auto-updates or removes when versions are synced

**How to trigger:**
- Automatically runs on PR open, synchronize, or reopened events
- No manual intervention needed - just push your changes

**Workflow location:**
- `internal/workflows/version_validation.go` - Main workflow logic
- `internal/workflows/version_validation_activities.go` - GitHub API interactions
- `internal/workflows/version_validation_test.go` - Comprehensive test suite

### Additional Automation (Future)

The sync script can be further automated via:
- **Pre-commit hook**: Sync version on every commit
- **Release workflow**: Auto-tag on VERSION changes

## Troubleshooting

### Version Mismatch Detected

If `plugin.json` version doesn't match `VERSION`:

```bash
./scripts/sync-version.sh
git diff  # Review changes
```

### Missing Git Tag

If you forgot to tag a release:

```bash
VERSION=$(cat VERSION)
git tag -a "v$VERSION" -m "Release v$VERSION"
git push --tags
```

### Rollback Version

```bash
git checkout VERSION
./scripts/sync-version.sh
```

## See Also

- Semantic Versioning: https://semver.org/
- Git tagging: `git help tag`

---

## Related Documentation

- [Releasing Guide](./RELEASING.md) - Creating releases
- [Main Documentation](./CONTEXTD.md) - Quick start and overview
- [Configuration Reference](./configuration.md) - All configuration options
