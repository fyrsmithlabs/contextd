//go:build cgo

package embeddings

import (
	"fmt"
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
