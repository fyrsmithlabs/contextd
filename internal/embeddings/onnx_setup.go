//go:build cgo

package embeddings

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultONNXRuntimeVersion is the ONNX runtime version matching onnxruntime_go.
// Update this when bumping the onnxruntime_go dependency in go.mod.
const DefaultONNXRuntimeVersion = "1.23.0"

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
