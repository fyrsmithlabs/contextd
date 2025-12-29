# Version Management

contextd uses semantic versioning with a single source of truth approach.

## Version File

The `VERSION` file at the project root is the **single source of truth** for the current version.

## Semantic Versioning

Versions follow the format: `MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]`

Examples:
- `0.3.0` - Release version
- `0.3.1-rc.1` - Release candidate
- `0.3.0+20230115` - Build metadata

### When to Increment

- **MAJOR**: Breaking changes, incompatible API changes
- **MINOR**: New features, backwards-compatible
- **PATCH**: Bug fixes, backwards-compatible

## Workflow

### 1. Update Version

Edit the `VERSION` file:
```bash
echo "0.4.0" > VERSION
```

### 2. Sync Version Across Files

Run the sync script to update all version references:
```bash
./scripts/sync-version.sh
```

This updates:
- `.claude-plugin/plugin.json`
- Any other version references

### 3. Commit Changes

```bash
git add VERSION .claude-plugin/plugin.json
git commit -m "chore: bump version to 0.4.0"
```

### 4. Create Tag

```bash
git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0
```

### 5. Release Workflow

When a tag is pushed, the GitHub Actions release workflow:
1. Validates the version format
2. Builds release binaries
3. Creates GitHub release
4. Updates Homebrew formula

## Pre-Release Workflow

For release candidates and testing:

```bash
echo "0.4.0-rc.1" > VERSION
./scripts/sync-version.sh
git commit -am "chore: prepare 0.4.0-rc.1"
git tag -a v0.4.0-rc.1 -m "Release candidate 0.4.0-rc.1"
git push origin v0.4.0-rc.1
```

## CI/CD Integration

The release workflow (`.github/workflows/release.yml`):
- Extracts version from git tag (`v0.3.0` → `0.3.0`)
- Validates version format (semantic versioning regex)
- Uses version for artifact naming and release notes

## Version Validation

Version format is validated in CI/CD:
```bash
^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$
```

This ensures:
- Three numeric components (MAJOR.MINOR.PATCH)
- Optional pre-release identifier after `-`
- Optional build metadata after `+`

## Troubleshooting

### Version Mismatch Between Components

If plugin version and VERSION file don't match:
```bash
./scripts/sync-version.sh
```

### Invalid Version Format

Version must match semantic versioning:
- ✅ `1.2.3`
- ✅ `1.2.3-rc.1`
- ✅ `1.2.3+build.123`
- ❌ `v1.2.3` (no 'v' prefix in VERSION file)
- ❌ `1.2` (must have three components)
