# ONNX Runtime Auto-Download Design

**Related Documents:**
- [SPEC.md](SPEC.md) - Requirements and success criteria

## Package Structure

```
internal/embeddings/
├── onnx_setup.go          # Download & path management
├── onnx_setup_test.go     # Tests
├── fastembed.go           # MODIFY: Call EnsureONNXRuntime() before init

cmd/ctxd/
└── cmd_init.go            # ctxd init command
```

## Data Flow

```
User runs contextd
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  GetONNXLibraryPath()                                       │
│  1. Check ONNX_PATH env var                                 │
│  2. Check ~/.config/contextd/lib/libonnxruntime.*           │
└─────────────────────────────────────────────────────────────┘
    │
    ▼ (if not found)
┌─────────────────────────────────────────────────────────────┐
│  EnsureONNXRuntime()                                        │
│  - Display download notice                                  │
│  - Call DownloadONNXRuntime()                               │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  DownloadONNXRuntime()                                      │
│  - Detect OS/arch                                           │
│  - Build GitHub release URL                                 │
│  - Download via go-getter                                   │
│  - Extract to ~/.config/contextd/lib/                       │
│  - Create symlinks                                          │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  ort.SetSharedLibraryPath(path)                             │
│  - Configure onnxruntime_go                                 │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  fastembed.NewFlagEmbedding()                               │
│  - Initialize embedding model                               │
└─────────────────────────────────────────────────────────────┘
```

## Interfaces

### onnx_setup.go

```go
const DefaultONNXRuntimeVersion = "1.23.2"

// GetONNXLibraryPath returns the path to ONNX runtime library.
// Checks ONNX_PATH env var first, then managed install location.
// Returns empty string if not found.
func GetONNXLibraryPath() string

// EnsureONNXRuntime ensures ONNX runtime is available.
// Downloads if missing. Returns path to library.
func EnsureONNXRuntime(ctx context.Context) (string, error)

// DownloadONNXRuntime downloads ONNX runtime for current platform.
// Version defaults to DefaultONNXRuntimeVersion if empty.
func DownloadONNXRuntime(ctx context.Context, version string) error

// ONNXRuntimeExists checks if ONNX runtime is installed.
func ONNXRuntimeExists() bool
```

## Platform Mapping

```go
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

var libraryName = map[string]string{
    "linux":  "libonnxruntime.so",
    "darwin": "libonnxruntime.dylib",
}
```

## Download URL Pattern

```
https://github.com/microsoft/onnxruntime/releases/download/v{version}/onnxruntime-{platform}-{version}.tgz
```

**Examples:**
- `onnxruntime-linux-x64-1.23.2.tgz`
- `onnxruntime-osx-arm64-1.23.2.tgz`

## FastEmbed Integration

```go
func NewFastEmbedProvider(cfg FastEmbedConfig) (*FastEmbedProvider, error) {
    // Ensure ONNX runtime is available before init
    onnxPath, err := EnsureONNXRuntime(context.Background())
    if err != nil {
        return nil, fmt.Errorf("ONNX runtime setup failed: %w (run 'ctxd init' to install)", err)
    }

    // Set library path for onnxruntime_go
    ort.SetSharedLibraryPath(onnxPath)

    // ... rest of existing init
}
```

## ctxd init Command

```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize contextd dependencies",
    Long:  "Downloads ONNX runtime and embedding models required by contextd.",
    RunE:  runInit,
}

func init() {
    initCmd.Flags().Bool("onnx-only", false, "Only download ONNX runtime, skip models")
    initCmd.Flags().Bool("force", false, "Re-download even if already installed")
    rootCmd.AddCommand(initCmd)
}
```

## Configuration

```yaml
# ~/.config/contextd/config.yaml
embeddings:
  onnx_version: "1.23.2"  # Optional override
```

## Dependencies

Add to go.mod:
```
github.com/hashicorp/go-getter v1.7.x
```

## Storage Layout

```
~/.config/contextd/
└── lib/
    ├── libonnxruntime.so         # Linux (symlink)
    ├── libonnxruntime.so.1.23.2  # Linux (actual)
    ├── libonnxruntime.dylib      # macOS (symlink)
    └── libonnxruntime.1.23.2.dylib  # macOS (actual)
```

## Error Messages

```go
var (
    ErrUnsupportedPlatform = errors.New("unsupported platform")
    ErrDownloadFailed      = errors.New("failed to download ONNX runtime")
    ErrExtractionFailed    = errors.New("failed to extract ONNX runtime")
)

// User-facing messages
const (
    msgDownloading    = "ONNX runtime not found. Downloading v%s for %s/%s..."
    msgDownloaded     = "Downloaded to %s"
    msgUseExplicit    = "Run 'ctxd init' to install, or set ONNX_PATH environment variable"
)
```
