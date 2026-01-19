# Version Management

**Single Source of Truth**: The `VERSION` file in the repository root.

## Quick Reference

| File/Location | Purpose | Updated How |
|---------------|---------|-------------|
| `VERSION` | **Source of truth** - all versions derive from this | Manual edit |
| Git tags (`v*`) | Release markers | `git tag` after sync |

> **Note:** The Claude plugin has moved to `fyrsmithlabs/marketplace`. Plugin versioning is managed separately in that repository.

## Versioning Workflow

### 1. Update Version

Edit the `VERSION` file:

```bash
echo "0.4.0" > VERSION
```

### 2. Commit Changes

```bash
git add VERSION
git commit -m "chore: bump version to 0.4.0"
```

### 3. Tag Release

```bash
git tag -a v0.4.0 -m "Release v0.4.0"
git push && git push --tags
```

### 4. Update Marketplace Plugin

After releasing contextd, update the plugin in `fyrsmithlabs/marketplace`:
- Update command descriptions if MCP tools changed
- Update skill documentation if workflows changed
- Bump marketplace plugin version

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
git add VERSION
git commit -m "chore: bump version to 0.4.0-rc1"
git tag -a v0.4.0-rc1 -m "Release candidate 1 for v0.4.0"
git push && git push --tags

# Iterate
echo "0.4.0-rc2" > VERSION
# ... repeat commit/tag/push
```

## Troubleshooting

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
```

## See Also

- Semantic Versioning: https://semver.org/
- Git tagging: `git help tag`
- Plugin repository: https://github.com/fyrsmithlabs/marketplace

---

## Related Documentation

- [Releasing Guide](./RELEASING.md) - Creating releases
- [Main Documentation](./CONTEXTD.md) - Quick start and overview
- [Configuration Reference](./configuration.md) - All configuration options
