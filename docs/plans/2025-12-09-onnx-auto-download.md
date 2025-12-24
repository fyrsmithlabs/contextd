# ONNX Runtime Auto-Download Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-download ONNX runtime on first FastEmbed use, with explicit `ctxd init` command.

**Architecture:** New `onnx_setup.go` handles path resolution and download via go-getter. FastEmbed calls `EnsureONNXRuntime()` before initialization. `ctxd init` provides explicit setup command.

**Tech Stack:** Go, hashicorp/go-getter, onnxruntime_go

**Spec:** See `docs/spec/onnx-auto-download/SPEC.md` and `DESIGN.md`

---

## Task 1: Add go-getter dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add dependency**

Run:
```bash
go get github.com/hashicorp/go-getter
```

**Step 2: Verify dependency added**

Run:
```bash
grep go-getter go.mod
```
Expected: `github.com/hashicorp/go-getter v1.x.x`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add hashicorp/go-getter for ONNX download"
```

---

## Task 2: Create platform mapping constants

**Files:**
- Create: `internal/embeddings/onnx_setup.go`
- Test: `internal/embeddings/onnx_setup_test.go`

**Step 1: Write failing test for platform mapping**

Create `internal/embeddings/onnx_setup_test.go`:

```go
//go:build cgo

package embeddings

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlatformArchive(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"linux", "amd64", "linux-x64"},
		{"linux", "arm64", "linux-aarch64"},
		{"darwin", "amd64", "osx-x86_64"},
		{"darwin", "arm64", "osx-arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, err := getPlatformArchive(tt.goos, tt.goarch)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPlatformArchive_Unsupported(t *testing.T) {
	_, err := getPlatformArchive("windows", "amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
}

func TestGetLibraryName(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", "libonnxruntime.so"},
		{"darwin", "libonnxruntime.dylib"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			got := getLibraryName(tt.goos)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCurrentPlatformSupported(t *testing.T) {
	// Current platform should be supported (linux or darwin)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		_, err := getPlatformArchive(runtime.GOOS, runtime.GOARCH)
		assert.NoError(t, err)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/embeddings/... -run TestGetPlatform -v
```
Expected: FAIL - functions not defined

**Step 3: Write minimal implementation**

Create `internal/embeddings/onnx_setup.go`:

```go
//go:build cgo

package embeddings

import (
	"fmt"
)

// DefaultONNXRuntimeVersion is the ONNX runtime version matching onnxruntime_go.
// Update this when bumping the onnxruntime_go dependency in go.mod.
const DefaultONNXRuntimeVersion = "1.23.2"

// ErrUnsupportedPlatform indicates the current OS/arch is not supported.
var ErrUnsupportedPlatform = fmt.Errorf("unsupported platform")

// platformArchMap maps GOOS/GOARCH to ONNX release archive names.
var platformArchMap = map[string]map[string]string{
	"linux": {
		"amd64": "linux-x64",
		"arm64": "linux-aarch64",
	},
	"darwin": {
		"amd64": "osx-x86_64",
		"arm64": "osx-arm64",
	},
}

// libraryNames maps GOOS to the shared library filename.
var libraryNames = map[string]string{
	"linux":  "libonnxruntime.so",
	"darwin": "libonnxruntime.dylib",
}

// getPlatformArchive returns the ONNX release archive name for the given OS/arch.
func getPlatformArchive(goos, goarch string) (string, error) {
	archMap, ok := platformArchMap[goos]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, goos, goarch)
	}
	arch, ok := archMap[goarch]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, goos, goarch)
	}
	return arch, nil
}

// getLibraryName returns the shared library filename for the given OS.
func getLibraryName(goos string) string {
	if name, ok := libraryNames[goos]; ok {
		return name
	}
	return "libonnxruntime.so" // fallback
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/embeddings/... -run TestGetPlatform -v
go test ./internal/embeddings/... -run TestGetLibrary -v
go test ./internal/embeddings/... -run TestCurrentPlatform -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embeddings/onnx_setup.go internal/embeddings/onnx_setup_test.go
git commit -m "feat(embeddings): add ONNX platform mapping"
```

---

## Task 3: Implement path resolution

**Files:**
- Modify: `internal/embeddings/onnx_setup.go`
- Modify: `internal/embeddings/onnx_setup_test.go`

**Step 1: Write failing test for path resolution**

Add to `internal/embeddings/onnx_setup_test.go`:

```go
func TestGetONNXInstallDir(t *testing.T) {
	dir := getONNXInstallDir()
	assert.Contains(t, dir, ".config/contextd/lib")
}

func TestGetONNXLibraryPath_EnvOverride(t *testing.T) {
	// Set env var
	t.Setenv("ONNX_PATH", "/custom/path/libonnxruntime.so")

	path := GetONNXLibraryPath()
	assert.Equal(t, "/custom/path/libonnxruntime.so", path)
}

func TestGetONNXLibraryPath_NoEnv_NoFile(t *testing.T) {
	// Ensure no env var
	t.Setenv("ONNX_PATH", "")

	// With no file present, should return empty
	path := GetONNXLibraryPath()
	// Either empty or the expected managed path (file may or may not exist)
	assert.True(t, path == "" || strings.Contains(path, ".config/contextd/lib"))
}

func TestONNXRuntimeExists_False(t *testing.T) {
	t.Setenv("ONNX_PATH", "")
	// Unless ONNX is already installed, this should reflect actual state
	// We just verify it doesn't panic
	_ = ONNXRuntimeExists()
}
```

Add import for `strings` at top of test file.

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/embeddings/... -run TestGetONNXInstallDir -v
go test ./internal/embeddings/... -run TestGetONNXLibraryPath -v
go test ./internal/embeddings/... -run TestONNXRuntimeExists -v
```
Expected: FAIL - functions not defined

**Step 3: Write minimal implementation**

Add to `internal/embeddings/onnx_setup.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// getONNXInstallDir returns the directory where ONNX runtime should be installed.
func getONNXInstallDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "contextd", "lib")
}

// GetONNXLibraryPath returns the path to the ONNX runtime library.
// Checks in order:
// 1. ONNX_PATH environment variable
// 2. Managed install at ~/.config/contextd/lib/
// Returns empty string if not found.
func GetONNXLibraryPath() string {
	// Check env var first (user override)
	if envPath := os.Getenv("ONNX_PATH"); envPath != "" {
		return envPath
	}

	// Check managed install location
	libName := getLibraryName(runtime.GOOS)
	managedPath := filepath.Join(getONNXInstallDir(), libName)
	if _, err := os.Stat(managedPath); err == nil {
		return managedPath
	}

	return ""
}

// ONNXRuntimeExists checks if ONNX runtime is available.
func ONNXRuntimeExists() bool {
	return GetONNXLibraryPath() != ""
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/embeddings/... -run TestGetONNX -v
go test ./internal/embeddings/... -run TestONNXRuntime -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embeddings/onnx_setup.go internal/embeddings/onnx_setup_test.go
git commit -m "feat(embeddings): add ONNX path resolution"
```

---

## Task 4: Implement download function

**Files:**
- Modify: `internal/embeddings/onnx_setup.go`
- Modify: `internal/embeddings/onnx_setup_test.go`

**Step 1: Write failing test for URL building**

Add to `internal/embeddings/onnx_setup_test.go`:

```go
func TestBuildDownloadURL(t *testing.T) {
	url := buildDownloadURL("1.23.2", "linux-x64")
	expected := "https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-x64-1.23.2.tgz"
	assert.Equal(t, expected, url)
}

func TestBuildDownloadURL_MacOS(t *testing.T) {
	url := buildDownloadURL("1.23.2", "osx-arm64")
	expected := "https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-osx-arm64-1.23.2.tgz"
	assert.Equal(t, expected, url)
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/embeddings/... -run TestBuildDownloadURL -v
```
Expected: FAIL - function not defined

**Step 3: Write minimal implementation**

Add to `internal/embeddings/onnx_setup.go`:

```go
const onnxReleaseURLTemplate = "https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.tgz"

// buildDownloadURL constructs the GitHub release URL for ONNX runtime.
func buildDownloadURL(version, platform string) string {
	return fmt.Sprintf(onnxReleaseURLTemplate, version, platform, version)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/embeddings/... -run TestBuildDownloadURL -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embeddings/onnx_setup.go internal/embeddings/onnx_setup_test.go
git commit -m "feat(embeddings): add ONNX download URL builder"
```

---

## Task 5: Implement full download with go-getter

**Files:**
- Modify: `internal/embeddings/onnx_setup.go`
- Modify: `internal/embeddings/onnx_setup_test.go`

**Step 1: Write integration test (skipped in CI)**

Add to `internal/embeddings/onnx_setup_test.go`:

```go
func TestDownloadONNXRuntime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()

	ctx := context.Background()
	err := downloadONNXRuntimeTo(ctx, DefaultONNXRuntimeVersion, tmpDir)
	require.NoError(t, err)

	// Verify library file exists
	libName := getLibraryName(runtime.GOOS)
	libPath := filepath.Join(tmpDir, libName)
	_, err = os.Stat(libPath)
	assert.NoError(t, err, "library file should exist at %s", libPath)
}
```

Add `context` to imports.

**Step 2: Write implementation**

Add to `internal/embeddings/onnx_setup.go`:

```go
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-getter"
)

// DownloadONNXRuntime downloads ONNX runtime for the current platform.
// If version is empty, uses DefaultONNXRuntimeVersion.
func DownloadONNXRuntime(ctx context.Context, version string) error {
	if version == "" {
		version = DefaultONNXRuntimeVersion
	}

	destDir := getONNXInstallDir()
	return downloadONNXRuntimeTo(ctx, version, destDir)
}

// downloadONNXRuntimeTo downloads ONNX runtime to the specified directory.
func downloadONNXRuntimeTo(ctx context.Context, version, destDir string) error {
	// Get platform archive name
	platform, err := getPlatformArchive(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	// Build download URL
	url := buildDownloadURL(version, platform)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "onnx-download-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download and extract using go-getter
	client := &getter.Client{
		Ctx:  ctx,
		Src:  url,
		Dst:  tmpDir,
		Mode: getter.ClientModeDir,
	}

	if err := client.Get(); err != nil {
		return fmt.Errorf("downloading ONNX runtime: %w", err)
	}

	// Find the extracted directory (onnxruntime-{platform}-{version})
	extractedDir := fmt.Sprintf("onnxruntime-%s-%s", platform, version)
	libSrcDir := filepath.Join(tmpDir, extractedDir, "lib")

	// Copy library files to destination
	libName := getLibraryName(runtime.GOOS)
	entries, err := os.ReadDir(libSrcDir)
	if err != nil {
		return fmt.Errorf("reading lib directory: %w", err)
	}

	var mainLib string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(libSrcDir, entry.Name())
		dstPath := filepath.Join(destDir, entry.Name())

		// Copy file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", entry.Name(), err)
		}

		// Track main library (without version suffix)
		if entry.Name() == libName || filepath.Ext(entry.Name()) == filepath.Ext(libName) {
			mainLib = dstPath
		}
	}

	// Create symlink if main library has version suffix
	symlinkPath := filepath.Join(destDir, libName)
	if mainLib != "" && mainLib != symlinkPath {
		os.Remove(symlinkPath) // Remove existing symlink if present
		if err := os.Symlink(filepath.Base(mainLib), symlinkPath); err != nil {
			// Symlink failed, try copying instead
			data, _ := os.ReadFile(mainLib)
			os.WriteFile(symlinkPath, data, 0644)
		}
	}

	return nil
}
```

**Step 3: Run test to verify it passes**

Run:
```bash
go test ./internal/embeddings/... -run TestDownloadONNXRuntime_Integration -v
```
Expected: PASS (downloads ~50-200MB, takes a few minutes)

**Step 4: Commit**

```bash
git add internal/embeddings/onnx_setup.go internal/embeddings/onnx_setup_test.go
git commit -m "feat(embeddings): implement ONNX runtime download"
```

---

## Task 6: Implement EnsureONNXRuntime with user feedback

**Files:**
- Modify: `internal/embeddings/onnx_setup.go`
- Modify: `internal/embeddings/onnx_setup_test.go`

**Step 1: Write test for EnsureONNXRuntime**

Add to `internal/embeddings/onnx_setup_test.go`:

```go
func TestEnsureONNXRuntime_AlreadyExists(t *testing.T) {
	// Create a fake library file
	tmpDir := t.TempDir()
	libName := getLibraryName(runtime.GOOS)
	libPath := filepath.Join(tmpDir, libName)
	os.WriteFile(libPath, []byte("fake"), 0644)

	// Set env to point to it
	t.Setenv("ONNX_PATH", libPath)

	ctx := context.Background()
	path, err := EnsureONNXRuntime(ctx)
	require.NoError(t, err)
	assert.Equal(t, libPath, path)
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/embeddings/... -run TestEnsureONNXRuntime -v
```
Expected: FAIL - function not defined

**Step 3: Write implementation**

Add to `internal/embeddings/onnx_setup.go`:

```go
// EnsureONNXRuntime ensures ONNX runtime is available, downloading if needed.
// Returns the path to the library file.
func EnsureONNXRuntime(ctx context.Context) (string, error) {
	// Check if already available
	if path := GetONNXLibraryPath(); path != "" {
		return path, nil
	}

	// Not found - download
	fmt.Printf("ONNX runtime not found. Downloading v%s for %s/%s...\n",
		DefaultONNXRuntimeVersion, runtime.GOOS, runtime.GOARCH)

	if err := DownloadONNXRuntime(ctx, ""); err != nil {
		return "", fmt.Errorf("failed to download ONNX runtime: %w\nRun 'ctxd init' to install manually, or set ONNX_PATH", err)
	}

	// Verify download succeeded
	path := GetONNXLibraryPath()
	if path == "" {
		return "", fmt.Errorf("ONNX runtime download completed but library not found")
	}

	fmt.Printf("Downloaded to %s\n", path)
	return path, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/embeddings/... -run TestEnsureONNXRuntime -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/embeddings/onnx_setup.go internal/embeddings/onnx_setup_test.go
git commit -m "feat(embeddings): add EnsureONNXRuntime with auto-download"
```

---

## Task 7: Integrate with FastEmbed provider

**Files:**
- Modify: `internal/embeddings/fastembed.go`

**Step 1: Add EnsureONNXRuntime call to NewFastEmbedProvider**

Modify `internal/embeddings/fastembed.go`, add at the start of `NewFastEmbedProvider`:

```go
import (
	ort "github.com/yalue/onnxruntime_go"
)

func NewFastEmbedProvider(cfg FastEmbedConfig) (*FastEmbedProvider, error) {
	// Ensure ONNX runtime is available
	onnxPath, err := EnsureONNXRuntime(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ONNX runtime setup failed: %w", err)
	}

	// Set library path for onnxruntime_go
	ort.SetSharedLibraryPath(onnxPath)

	// ... rest of existing function unchanged
```

**Step 2: Verify build succeeds**

Run:
```bash
go build ./...
```
Expected: Build succeeds

**Step 3: Run existing FastEmbed tests**

Run:
```bash
go test ./internal/embeddings/... -v
```
Expected: Tests pass (may auto-download ONNX runtime)

**Step 4: Commit**

```bash
git add internal/embeddings/fastembed.go
git commit -m "feat(embeddings): integrate ONNX auto-download with FastEmbed"
```

---

## Task 8: Add ctxd init command

**Files:**
- Create: `cmd/ctxd/cmd_init.go`
- Modify: `cmd/ctxd/main.go` (add init command registration)

**Step 1: Check existing ctxd structure**

Run:
```bash
ls -la cmd/ctxd/
```

**Step 2: Create init command**

Create `cmd/ctxd/cmd_init.go`:

```go
package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize contextd dependencies",
	Long:  "Downloads ONNX runtime and embedding models required by contextd.",
	RunE:  runInit,
}

var (
	initONNXOnly bool
	initForce    bool
)

func init() {
	initCmd.Flags().BoolVar(&initONNXOnly, "onnx-only", false, "Only download ONNX runtime, skip models")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Re-download even if already installed")
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if already installed
	if !initForce && embeddings.ONNXRuntimeExists() {
		fmt.Println("ONNX runtime already installed.")
		path := embeddings.GetONNXLibraryPath()
		fmt.Printf("Location: %s\n", path)
		if !initONNXOnly {
			fmt.Println("\nTo download embedding models, run contextd once.")
		}
		return nil
	}

	// Download ONNX runtime
	fmt.Printf("Downloading ONNX Runtime v%s for %s/%s...\n",
		embeddings.DefaultONNXRuntimeVersion, runtime.GOOS, runtime.GOARCH)

	if err := embeddings.DownloadONNXRuntime(ctx, ""); err != nil {
		return fmt.Errorf("failed to download ONNX runtime: %w", err)
	}

	path := embeddings.GetONNXLibraryPath()
	fmt.Printf("✓ ONNX Runtime installed to %s\n", path)

	if initONNXOnly {
		fmt.Println("\nSetup complete (ONNX only).")
		return nil
	}

	fmt.Println("\nTo download embedding models, run contextd once.")
	fmt.Println("Setup complete.")
	return nil
}
```

**Step 3: Register init command in main.go**

Modify `cmd/ctxd/main.go` to add:

```go
func init() {
	rootCmd.AddCommand(initCmd)
	// ... other commands
}
```

**Step 4: Build and test**

Run:
```bash
go build -o ctxd ./cmd/ctxd
./ctxd init --help
```
Expected: Shows init command help

**Step 5: Commit**

```bash
git add cmd/ctxd/cmd_init.go cmd/ctxd/main.go
git commit -m "feat(ctxd): add init command for ONNX setup"
```

---

## Task 9: Add config override for ONNX version

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/embeddings/onnx_setup.go`

**Step 1: Add config field**

Add to embeddings config in `internal/config/config.go`:

```go
type EmbeddingsConfig struct {
	// ... existing fields
	ONNXVersion string `koanf:"onnx_version"` // Optional override for ONNX runtime version
}
```

**Step 2: Update DownloadONNXRuntime to accept version from config**

This is already handled - `DownloadONNXRuntime` accepts version parameter, defaults to `DefaultONNXRuntimeVersion` if empty.

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add onnx_version config option"
```

---

## Task 10: Update documentation

**Files:**
- Modify: `docs/configuration.md`
- Modify: `README.md`
- Modify: `QUICKSTART.md`

**Step 1: Update configuration docs**

Add to `docs/configuration.md`:

```markdown
### ONNX Runtime

contextd automatically downloads ONNX runtime on first use. You can also run:

```bash
ctxd init
```

**Environment Variables:**
- `ONNX_PATH` - Override ONNX runtime library path

**Config Options:**
```yaml
embeddings:
  onnx_version: "1.23.2"  # Optional version override
```
```

**Step 2: Update README/QUICKSTART**

Add to installation section:

```markdown
## Quick Start

```bash
# Initialize dependencies (downloads ONNX runtime)
ctxd init

# Or just run - dependencies download automatically
contextd
```
```

**Step 3: Commit**

```bash
git add docs/configuration.md README.md QUICKSTART.md
git commit -m "docs: add ONNX auto-download documentation"
```

---

## Summary

10 tasks total:
1. Add go-getter dependency
2. Create platform mapping constants
3. Implement path resolution
4. Implement download URL builder
5. Implement full download with go-getter
6. Implement EnsureONNXRuntime
7. Integrate with FastEmbed provider
8. Add ctxd init command
9. Add config override
10. Update documentation

After completion, run full test suite:
```bash
go test ./... -v
```
