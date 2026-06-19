//go:build cgo

package embeddings

import (
	"archive/zip"
	"bytes"
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
		{"windows", "amd64", "win-x64"},
		{"windows", "arm64", "win-arm64"},
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
	// Unknown OS and unknown arch on a known OS both fail.
	_, err := getPlatformArchive("plan9", "amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")

	_, err = getPlatformArchive("windows", "386")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
}

func TestGetArchiveExtension(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", "tgz"},
		{"darwin", "tgz"},
		{"windows", "zip"},
		{"plan9", "tgz"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			assert.Equal(t, tt.want, getArchiveExtension(tt.goos))
		})
	}
}

func TestGetLibraryName(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", "libonnxruntime.so"},
		{"darwin", "libonnxruntime.dylib"},
		{"windows", "onnxruntime.dll"},
		{"plan9", "libonnxruntime.so"}, // fallback
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
	tests := []struct {
		name     string
		platform string
		ext      string
		expected string
	}{
		{
			name:     "linux",
			platform: "linux-x64",
			ext:      "tgz",
			expected: "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-linux-x64-1.23.0.tgz",
		},
		{
			name:     "macos",
			platform: "osx-arm64",
			ext:      "tgz",
			expected: "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-osx-arm64-1.23.0.tgz",
		},
		{
			name:     "windows-amd64",
			platform: "win-x64",
			ext:      "zip",
			expected: "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-win-x64-1.23.0.zip",
		},
		{
			name:     "windows-arm64",
			platform: "win-arm64",
			ext:      "zip",
			expected: "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-win-arm64-1.23.0.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildDownloadURL("1.23.0", tt.platform, tt.ext))
		})
	}
}

func TestFindBundledLibrary(t *testing.T) {
	t.Run("not found in empty dir", func(t *testing.T) {
		assert.Empty(t, findBundledLibrary(t.TempDir(), "linux"))
	})

	t.Run("empty exe dir returns empty", func(t *testing.T) {
		assert.Empty(t, findBundledLibrary("", "linux"))
	})

	t.Run("found directly next to executable", func(t *testing.T) {
		dir := t.TempDir()
		libPath := filepath.Join(dir, getLibraryName("linux"))
		require.NoError(t, os.WriteFile(libPath, []byte("fake"), 0644))
		assert.Equal(t, libPath, findBundledLibrary(dir, "linux"))
	})

	t.Run("found in lib subdir next to executable", func(t *testing.T) {
		dir := t.TempDir()
		libDir := filepath.Join(dir, "lib")
		require.NoError(t, os.MkdirAll(libDir, 0755))
		libPath := filepath.Join(libDir, getLibraryName("windows"))
		require.NoError(t, os.WriteFile(libPath, []byte("fake"), 0644))
		assert.Equal(t, libPath, findBundledLibrary(dir, "windows"))
	})

	t.Run("ignores directories named like the lib", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, getLibraryName("linux")), 0755))
		assert.Empty(t, findBundledLibrary(dir, "linux"))
	})
}

func TestGetONNXLibraryPath_ResolutionOrder(t *testing.T) {
	// Restore the executableDir seam after the test.
	origExeDir := executableDir
	t.Cleanup(func() { executableDir = origExeDir })

	t.Run("ONNX_PATH wins over bundled and managed", func(t *testing.T) {
		exeDir := t.TempDir()
		// Place a bundled lib that should be ignored in favor of ONNX_PATH.
		bundled := filepath.Join(exeDir, getLibraryName(runtime.GOOS))
		require.NoError(t, os.WriteFile(bundled, []byte("fake"), 0644))
		executableDir = func() string { return exeDir }

		t.Setenv("ONNX_PATH", "/explicit/override/lib")
		assert.Equal(t, "/explicit/override/lib", GetONNXLibraryPath())
	})

	t.Run("bundled exe-dir lib wins over managed when no ONNX_PATH", func(t *testing.T) {
		t.Setenv("ONNX_PATH", "")
		exeDir := t.TempDir()
		bundled := filepath.Join(exeDir, getLibraryName(runtime.GOOS))
		require.NoError(t, os.WriteFile(bundled, []byte("fake"), 0644))
		executableDir = func() string { return exeDir }

		assert.Equal(t, bundled, GetONNXLibraryPath())
	})

	t.Run("falls back to managed path when nothing bundled", func(t *testing.T) {
		t.Setenv("ONNX_PATH", "")
		// Point exe dir at an empty directory so no bundled lib is found.
		executableDir = func() string { return t.TempDir() }

		path := GetONNXLibraryPath()
		// Either empty (managed lib absent) or the managed cache path.
		assert.True(t, path == "" || strings.Contains(path, ".config/contextd/lib"),
			"expected empty or managed path, got %q", path)
	})
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

// buildTestZip constructs an in-memory zip mirroring the Microsoft windows
// release layout: <prefix>/lib/<entries>.
func buildTestZip(t *testing.T, prefix string, entries map[string][]byte) ([]byte, int64) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(prefix + "lib/" + name)
		require.NoError(t, err)
		_, err = w.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes(), int64(buf.Len())
}

func TestExtractZip(t *testing.T) {
	const (
		version  = "1.23.0"
		platform = "win-x64"
	)
	prefix := "onnxruntime-" + platform + "-" + version + "/"

	t.Run("extracts dll and sidecars, skips lib and pdb", func(t *testing.T) {
		data, size := buildTestZip(t, prefix, map[string][]byte{
			"onnxruntime.dll":                  []byte("core-dll"),
			"onnxruntime_providers_shared.dll": []byte("provider-dll"),
			"onnxruntime.lib":                  []byte("import-lib"),
			"onnxruntime.pdb":                  []byte("debug-symbols"),
		})

		destDir := t.TempDir()
		err := extractZip(bytes.NewReader(data), size, destDir, version, platform)
		// On windows the main-lib (onnxruntime.dll) check passes; on other hosts
		// getLibraryName(runtime.GOOS) differs so it reports the lib missing.
		// Either way the DLLs from lib/ must have been written and non-DLLs skipped.
		if runtime.GOOS == "windows" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}

		// DLLs extracted regardless of host.
		got, err := os.ReadFile(filepath.Join(destDir, "onnxruntime.dll"))
		require.NoError(t, err)
		assert.Equal(t, "core-dll", string(got))
		_, err = os.Stat(filepath.Join(destDir, "onnxruntime_providers_shared.dll"))
		assert.NoError(t, err)

		// Non-DLL artifacts skipped.
		_, err = os.Stat(filepath.Join(destDir, "onnxruntime.lib"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(destDir, "onnxruntime.pdb"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("errors when main lib missing", func(t *testing.T) {
		data, size := buildTestZip(t, prefix, map[string][]byte{
			"onnxruntime_providers_shared.dll": []byte("provider-only"),
		})
		err := extractZip(bytes.NewReader(data), size, t.TempDir(), version, platform)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in archive")
	})
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
