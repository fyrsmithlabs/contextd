//go:build cgo

package embeddings

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	// With no file present, should return empty or the expected managed path (file may or may not exist)
	path := GetONNXLibraryPath()
	assert.True(t, path == "" || strings.Contains(path, ".config/contextd/lib"))
}

func TestONNXRuntimeExists_False(t *testing.T) {
	t.Setenv("ONNX_PATH", "")
	// Unless ONNX is already installed, this should reflect actual state
	// We just verify it doesn't panic
	_ = ONNXRuntimeExists()
}

func TestBuildDownloadURL(t *testing.T) {
	url := buildDownloadURL("1.23.0", "linux-x64")
	expected := "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-linux-x64-1.23.0.tgz"
	assert.Equal(t, expected, url)
}

func TestBuildDownloadURL_MacOS(t *testing.T) {
	url := buildDownloadURL("1.23.0", "osx-arm64")
	expected := "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-osx-arm64-1.23.0.tgz"
	assert.Equal(t, expected, url)
}

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

func TestEnsureONNXRuntime_AlreadyExists(t *testing.T) {
	// Create a fake library file
	tmpDir := t.TempDir()
	libName := getLibraryName(runtime.GOOS)
	libPath := filepath.Join(tmpDir, libName)
	err := os.WriteFile(libPath, []byte("fake"), 0644)
	require.NoError(t, err)

	// Set env to point to it
	t.Setenv("ONNX_PATH", libPath)

	ctx := context.Background()
	path, err := EnsureONNXRuntime(ctx)
	require.NoError(t, err)
	assert.Equal(t, libPath, path)
}
