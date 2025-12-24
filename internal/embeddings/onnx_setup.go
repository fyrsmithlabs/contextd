//go:build cgo

package embeddings

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

const onnxReleaseURLTemplate = "https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.tgz"

// buildDownloadURL constructs the GitHub release URL for ONNX runtime.
func buildDownloadURL(version, platform string) string {
	return fmt.Sprintf(onnxReleaseURLTemplate, version, platform, version)
}

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
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Perform download
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading ONNX runtime: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Extract tarball directly from response
	if err := extractTarGz(resp.Body, destDir, version, platform); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}

// extractTarGz extracts library files from the ONNX runtime tarball.
// The archive contains libonnxruntime.so/.dylib plus symlinks and related files.
// We extract everything from the lib/ directory as-is.
func extractTarGz(r io.Reader, destDir, version, platform string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Expected directory prefix in the archive
	expectedPrefix := fmt.Sprintf("onnxruntime-%s-%s/lib/", platform, version)
	libName := getLibraryName(runtime.GOOS)

	var foundMainLib bool

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Normalize path - strip leading "./" if present
		name := header.Name
		if strings.HasPrefix(name, "./") {
			name = name[2:]
		}

		// Only extract files from the lib directory
		if !strings.HasPrefix(name, expectedPrefix) {
			continue
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Get filename from normalized path
		filename := filepath.Base(name)
		destPath := filepath.Join(destDir, filename)

		// Handle symlinks from the archive
		if header.Typeflag == tar.TypeSymlink {
			// Remove existing file/symlink if present
			os.Remove(destPath)
			if err := os.Symlink(header.Linkname, destPath); err != nil {
				// If symlink fails, we'll rely on the actual file being extracted
				continue
			}
			if filename == libName {
				foundMainLib = true
			}
			continue
		}

		// Extract regular file
		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("creating file %s: %w", filename, err)
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("writing file %s: %w", filename, err)
		}
		outFile.Close()

		// Track if we found the main library
		if filename == libName || strings.HasPrefix(filename, libName+".") {
			foundMainLib = true
		}
	}

	if !foundMainLib {
		return fmt.Errorf("library %s not found in archive", libName)
	}

	return nil
}

// setONNXPathEnv sets the ONNX_PATH environment variable.
// This is used by fastembed-go to locate the library.
// Separated into a function for testability.
var setONNXPathEnv = func(path string) error {
	return os.Setenv("ONNX_PATH", path)
}

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
