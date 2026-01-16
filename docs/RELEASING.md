# Releasing ContextD

This document describes how to create releases for ContextD.

## Prerequisites

### 1. Create the Homebrew Tap Repository

Create a new repository at `github.com/fyrsmithlabs/homebrew-tap`:

```bash
gh repo create fyrsmithlabs/homebrew-tap --public --description "Homebrew tap for FyrSmith Labs projects"
```

### 2. Create a Personal Access Token for Homebrew

GoReleaser needs a token with access to the homebrew-tap repository:

1. Go to https://github.com/settings/tokens
2. Create a new fine-grained token with:
   - Repository access: `fyrsmithlabs/homebrew-tap`
   - Permissions: `Contents: Read and write`
3. Copy the token

### 3. Add the Token as a Repository Secret

1. Go to https://github.com/fyrsmithlabs/contextd/settings/secrets/actions
2. Add a new secret named `HOMEBREW_TAP_TOKEN`
3. Paste the token value

## Creating a Release

### Release Candidate (RC)

```bash
git tag -a v1.0.0-rc1 -m "Release candidate v1.0.0-rc1"
git push origin v1.0.0-rc1
```

RC releases are automatically marked as pre-release on GitHub.

### Stable Release

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## What Gets Built

GoReleaser automatically builds:

### Binaries

| Platform | Architecture | File |
|----------|--------------|------|
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `contextd_*_linux_arm64.tar.gz` |
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| Windows | x64 | `contextd_*_windows_amd64.zip` |

### Docker Images

| Tag | Architectures | Description |
|-----|---------------|-------------|
| `ghcr.io/fyrsmithlabs/contextd:X.Y.Z` | amd64, arm64/v8 | Versioned release |
| `ghcr.io/fyrsmithlabs/contextd:latest` | amd64, arm64/v8 | Latest stable |
| `ghcr.io/fyrsmithlabs/contextd:X.Y.Z-amd64` | amd64 | Architecture-specific |
| `ghcr.io/fyrsmithlabs/contextd:X.Y.Z-arm64v8` | arm64/v8 | Architecture-specific |

### Homebrew Formula

Updated automatically at `fyrsmithlabs/homebrew-tap`.

## Testing Locally

```bash
# Install GoReleaser
brew install goreleaser

# Test build without publishing
goreleaser release --snapshot --clean

# Check artifacts
ls -la dist/
```

## Troubleshooting

### Homebrew Formula Not Updated

- Verify `HOMEBREW_TAP_TOKEN` secret is set
- Check token has write access to `fyrsmithlabs/homebrew-tap`
- Ensure the repository exists

### Docker Push Failed

- Verify GitHub Actions has `packages: write` permission
- Check package visibility settings

### Build Failed

Common issues:
- Run `go mod tidy` if module errors
- Check `CGO_ENABLED=0` for cross-compilation issues

---

## Related Documentation

- [Versioning Guide](./VERSIONING.md) - Version management
- [Main Documentation](./CONTEXTD.md) - Quick start and overview
- [Docker Guide](./DOCKER.md) - Running contextd in Docker
