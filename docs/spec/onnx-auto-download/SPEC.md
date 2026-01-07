# ONNX Runtime Auto-Download Specification

**Status**: ✅ Implemented
**Updated**: 2026-01-06
**Implementation**: `internal/embeddings/onnx_setup.go`, `cmd/ctxd/init.go`

**Related Documents:**
- [DESIGN.md](DESIGN.md) - Architecture and components

## Problem Statement

ONNX runtime must be manually installed for FastEmbed to work. This causes:
- Test failures in environments without ONNX configured
- Poor developer experience requiring manual setup
- Homebrew dependency not reliably picked up

## Goals

1. contextd works out-of-the-box without manual ONNX installation
2. Explicit `ctxd init` command for controlled setup
3. Auto-download fallback on first FastEmbed use
4. User can override with their own ONNX installation via `ONNX_PATH`

## Functional Requirements

### FR-001: Library Path Resolution
The system shall resolve ONNX runtime path in order:
1. `ONNX_PATH` environment variable (user override)
2. `~/.config/contextd/lib/libonnxruntime.{so,dylib}` (managed install)

### FR-002: Auto-Download on First Use
When FastEmbed initializes and no ONNX runtime is found:
1. Display notice: "ONNX runtime not found. Downloading..."
2. Download appropriate version for OS/architecture
3. Extract to `~/.config/contextd/lib/`
4. Continue with FastEmbed initialization

### FR-003: Explicit Init Command
`ctxd init` shall:
1. Download ONNX runtime for current platform
2. Download default embedding model
3. Display progress and completion status

### FR-004: Platform Support
Support these platform/architecture combinations:
| OS | Arch | Status |
|----|------|--------|
| linux | amd64 | Required |
| linux | arm64 | Required |
| darwin | amd64 | Required |
| darwin | arm64 | Required |

### FR-005: Version Management
- Default version hardcoded as constant (currently `1.23.2`)
- Config file can override: `embeddings.onnx_version`
- Version must match onnxruntime_go dependency

### FR-006: Error Handling
| Scenario | Behavior |
|----------|----------|
| No network, no cached lib | Error with actionable message |
| Download fails mid-way | Clean up partial files, allow retry |
| Unsupported platform | Clear error message |
| User has ONNX_PATH set | Skip download, use their path |

## Non-Functional Requirements

### NFR-001: Download Size
ONNX runtime archives are 50-200MB. Download shall:
- Show progress indicator
- Support resume on failure (via go-getter)

### NFR-002: No Sudo Required
All operations work without elevated privileges.

### NFR-003: Idempotent
Running `ctxd init` multiple times is safe. Use `--force` to re-download.

## Success Criteria

- ✅ SC-001: `ctxd init` downloads and installs ONNX runtime
- ✅ SC-002: FastEmbed auto-downloads when runtime missing
- ✅ SC-003: `ONNX_PATH` override works correctly
- ✅ SC-004: All supported platforms can download correct binary
- ✅ SC-005: Embedding tests pass without pre-installed ONNX

## Implementation Notes

All success criteria have been met. The implementation includes:
- `internal/embeddings/onnx_setup.go` - Download and extraction logic
- `cmd/ctxd/init.go` - CLI command for manual initialization
- Auto-download fallback on first FastEmbed use
- Platform detection and appropriate binary selection

## Out of Scope

- Windows support
- Building ONNX from source
- GPU/CUDA variants (CPU only)
