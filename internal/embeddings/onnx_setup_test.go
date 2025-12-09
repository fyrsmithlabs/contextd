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
